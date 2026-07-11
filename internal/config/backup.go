package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const maxBackups = 10

// backupSSHConfig crée une copie datée de ~/.ssh/config dans
// ~/.config/tmssh/backups/ avant toute opération d'écriture.
//
// Format du nom : ssh_config.2006-01-02T150405.bak
//
// Les backups les plus anciens sont supprimés au-delà de maxBackups,
// pour ne pas saturer le disque.
//
// Appelée par writeTagsToSSHConfig (sync active ou passive via tag add/rm)
// et par Sync. Silencieuse si ~/.ssh/config n'existe pas encore.
func backupSSHConfig() error {
	src, err := sshConfigPath()
	if err != nil {
		return err
	}

	in, err := os.Open(src)
	if os.IsNotExist(err) {
		return nil // rien à sauvegarder
	}
	if err != nil {
		return err
	}
	defer in.Close()

	dir, err := backupDir()
	if err != nil {
		return err
	}

	stamp := time.Now().Format("2006-01-02T150405")
	dst := filepath.Join(dir, "ssh_config."+stamp+".bak")

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		// Backup de la même seconde déjà présent — pas bloquant.
		if os.IsExist(err) {
			return nil
		}
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("backup ssh_config: %w", err)
	}

	return pruneBackups(dir)
}

// backupDir retourne ~/.config/tmssh/backups (créé si absent).
func backupDir() (string, error) {
	base, err := Dir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "backups")
	return dir, os.MkdirAll(dir, 0o755)
}

// pruneBackups supprime les backups les plus anciens au-delà de maxBackups.
func pruneBackups(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	var baks []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".bak") {
			baks = append(baks, filepath.Join(dir, e.Name()))
		}
	}

	sort.Strings(baks) // tri lexicographique = chronologique (ISO timestamp)
	for len(baks) > maxBackups {
		if err := os.Remove(baks[0]); err != nil && !os.IsNotExist(err) {
			return err
		}
		baks = baks[1:]
	}
	return nil
}

// ListBackups retourne les chemins des backups existants (du plus récent au plus ancien).
func ListBackups() ([]string, error) {
	dir, err := backupDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var baks []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".bak") {
			baks = append(baks, filepath.Join(dir, e.Name()))
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(baks)))
	return baks, nil
}
