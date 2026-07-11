package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Les tags sont persistés dans ~/.ssh/config sous forme de commentaire
// structuré juste au-dessus du bloc Host :
//
//	# tmssh:tags=prod,web
//	Host web-01
//	    HostName 10.0.0.5
//	    User deploy
//
// Cela survit aux éditions manuelles et reste invisible pour ssh.
const tagPrefix = "# tmssh:tags="

func sshConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".ssh", "config"), nil
}

// importFromSSHConfig remplit cfg depuis ~/.ssh/config (premier lancement).
func importFromSSHConfig(cfg *Config) error {
	hosts, err := parseSSHConfig()
	if err != nil {
		return err
	}
	cfg.Hosts = hosts
	return nil
}

// parseSSHConfig lit ~/.ssh/config. Parseur volontairement minimal :
// blocs Host + HostName/User/Port + commentaire tmssh:tags.
// Les Host avec wildcards (* ?) sont ignorés.
func parseSSHConfig() ([]Host, error) {
	path, err := sshConfigPath()
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var (
		hosts       []Host
		cur         *Host
		pendingTags []string
	)
	flush := func() {
		if cur != nil {
			hosts = append(hosts, *cur)
			cur = nil
		}
	}

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())

		if strings.HasPrefix(line, tagPrefix) {
			pendingTags = splitTags(strings.TrimPrefix(line, tagPrefix))
			continue
		}
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, val, ok := splitKeyValue(line)
		if !ok {
			continue
		}
		switch strings.ToLower(key) {
		case "host":
			flush()
			if strings.ContainsAny(val, "*?") {
				pendingTags = nil
				continue // pattern, pas un host concret
			}
			cur = &Host{Name: val, Tags: pendingTags}
			pendingTags = nil
		case "hostname":
			if cur != nil {
				cur.HostName = val
			}
		case "user":
			if cur != nil {
				cur.User = val
			}
		case "port":
			if cur != nil {
				cur.Port = val
			}
		}
	}
	flush()
	return hosts, sc.Err()
}

func splitKeyValue(line string) (key, val string, ok bool) {
	// ssh_config accepte "Key Value" et "Key=Value".
	if i := strings.IndexAny(line, " \t="); i > 0 {
		return line[:i], strings.TrimSpace(strings.TrimLeft(line[i:], " \t=")), true
	}
	return "", "", false
}

func splitTags(s string) []string {
	var tags []string
	for _, t := range strings.Split(s, ",") {
		if t = strings.TrimSpace(t); t != "" {
			tags = append(tags, t)
		}
	}
	return tags
}

// Sync réconcilie hosts.json et ~/.ssh/config :
//   - les hosts présents seulement dans ssh_config sont importés dans le JSON ;
//   - les tags du JSON sont réécrits dans ssh_config (commentaires tmssh:tags).
//
// Le JSON est la source de vérité pour les tags ; ssh_config l'est pour
// les paramètres de connexion.
func Sync(cfg *Config) error {
	fromSSH, err := parseSSHConfig()
	if err != nil {
		return err
	}

	// 1. Import des nouveaux hosts + rafraîchissement des params de connexion.
	for _, h := range fromSSH {
		if existing := cfg.Find(h.Name); existing != nil {
			existing.HostName, existing.User, existing.Port = h.HostName, h.User, h.Port
		} else {
			cfg.Hosts = append(cfg.Hosts, h)
		}
	}
	if err := cfg.Save(); err != nil {
		return err
	}

	// 2. Réécriture des commentaires de tags dans ssh_config.
	return writeTagsToSSHConfig(cfg)
}

// writeTagsToSSHConfig réécrit ~/.ssh/config en préservant tout son contenu,
// en remplaçant uniquement les commentaires "# tmssh:tags=" de chaque bloc Host.
func writeTagsToSSHConfig(cfg *Config) error {
	path, err := sshConfigPath()
	if err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil // rien à réécrire
	}
	if err != nil {
		return err
	}

	var out []string
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)

		// On supprime les anciens commentaires de tags ; ils seront régénérés.
		if strings.HasPrefix(line, tagPrefix) {
			continue
		}

		// Devant chaque "Host <name>" concret, on injecte le commentaire.
		if key, val, ok := splitKeyValue(line); ok &&
			strings.EqualFold(key, "Host") && !strings.ContainsAny(val, "*?") {
			if h := cfg.Find(val); h != nil && len(h.Tags) > 0 {
				out = append(out, tagPrefix+strings.Join(h.Tags, ","))
			}
		}
		out = append(out, raw)
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), ".config-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	if _, err := fmt.Fprint(tmp, strings.Join(out, "\n")); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
