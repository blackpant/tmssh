// Package config gère hosts.json et sa synchro avec ~/.ssh/config.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
)

// Host représente une entrée SSH enrichie de tags.
type Host struct {
	Name     string   `json:"name"`               // alias (Host dans ssh_config)
	HostName string   `json:"hostname,omitempty"` // adresse réelle
	User     string   `json:"user,omitempty"`
	Port     string   `json:"port,omitempty"`
	Tags     []string `json:"tags,omitempty"`
}

// Config est le contenu de hosts.json.
type Config struct {
	Hosts []Host `json:"hosts"`

	// Icons associe un tag à une icône Nerd Font ; DefaultIcon est utilisée
	// quand aucun tag du host n'a d'icône. Les deux sont éditables dans
	// hosts.json et pré-remplis au premier lancement.
	Icons       map[string]string `json:"icons,omitempty"`
	DefaultIcon string            `json:"default_icon,omitempty"`

	// Groups : groupes de connexion nommés (voir groups.go).
	Groups map[string]Group `json:"groups,omitempty"`

	path string // chemin du fichier, non sérialisé
}

// IconFor retourne l'icône du premier tag du host qui en possède une,
// sinon l'icône par défaut.
func (c *Config) IconFor(h Host) string {
	for _, t := range h.Tags {
		if ic, ok := c.Icons[t]; ok {
			return ic
		}
	}
	if c.DefaultIcon != "" {
		return c.DefaultIcon
	}
	return ""
}

// Dir retourne ~/.config/tmssh (créé si absent).
func Dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "tmssh")
	return dir, os.MkdirAll(dir, 0o755)
}

// Load charge hosts.json ; un fichier absent donne une config vide.
// Une sync initiale depuis ~/.ssh/config est faite si le JSON n'existe pas.
func Load() (*Config, error) {
	dir, err := Dir()
	if err != nil {
		return nil, err
	}
	cfg := &Config{path: filepath.Join(dir, "hosts.json")}

	data, err := os.ReadFile(cfg.path)
	switch {
	case os.IsNotExist(err):
		// Premier lancement : on importe ~/.ssh/config et on pré-remplit
		// les icônes (modifiables ensuite directement dans hosts.json).
		if err := importFromSSHConfig(cfg); err != nil {
			return nil, err
		}
		cfg.Icons = map[string]string{
			"prod":    "",
			"staging": "",
			"dev":     "",
			"web":     "",
			"db":      "",
		}
		cfg.DefaultIcon = ""
		return cfg, cfg.Save()
	case err != nil:
		return nil, err
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("hosts.json invalide: %w", err)
	}
	return cfg, nil
}

// Save écrit hosts.json de façon atomique (tmp + rename) pour éviter
// la corruption si plusieurs popups écrivent en même temps.
func (c *Config) Save() error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(c.path), ".hosts-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // no-op si le rename a réussi

	if _, err := tmp.Write(append(data, '\n')); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), c.path)
}

// Find retourne le host nommé name, ou nil.
func (c *Config) Find(name string) *Host {
	for i := range c.Hosts {
		if c.Hosts[i].Name == name {
			return &c.Hosts[i]
		}
	}
	return nil
}

// AllTags retourne l'ensemble trié des tags connus.
func (c *Config) AllTags() []string {
	set := map[string]struct{}{}
	for _, h := range c.Hosts {
		for _, t := range h.Tags {
			set[t] = struct{}{}
		}
	}
	tags := make([]string, 0, len(set))
	for t := range set {
		tags = append(tags, t)
	}
	sort.Strings(tags)
	return tags
}

// AddTag ajoute tag aux hosts nommés, puis sauvegarde + resync ssh_config.
func (c *Config) AddTag(tag string, names []string) error {
	for _, n := range names {
		h := c.Find(n)
		if h == nil {
			continue
		}
		if !slices.Contains(h.Tags, tag) {
			h.Tags = append(h.Tags, tag)
			sort.Strings(h.Tags)
		}
	}
	if err := c.Save(); err != nil {
		return err
	}
	return Sync(c)
}

// RemoveTag retire tag des hosts nommés, puis sauvegarde + resync ssh_config.
func (c *Config) RemoveTag(tag string, names []string) error {
	for _, n := range names {
		h := c.Find(n)
		if h == nil {
			continue
		}
		h.Tags = slices.DeleteFunc(h.Tags, func(t string) bool { return t == tag })
	}
	if err := c.Save(); err != nil {
		return err
	}
	return Sync(c)
}

// CommonTags retourne les tags présents sur TOUS les hosts nommés
// (utilisé par `tag rm` pour ne proposer que des suppressions valides).
func (c *Config) CommonTags(names []string) []string {
	var common []string
	for i, n := range names {
		h := c.Find(n)
		if h == nil {
			return nil
		}
		if i == 0 {
			common = slices.Clone(h.Tags)
			continue
		}
		common = slices.DeleteFunc(common, func(t string) bool {
			return !slices.Contains(h.Tags, t)
		})
	}
	sort.Strings(common)
	return common
}
