package config

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Le filtre par tags actif est stocké dans ~/.config/tmssh/tagfilter
// (un tag par ligne). Il survit entre les reloads fzf mais est effacé
// à chaque nouveau lancement de l'UI, pour repartir d'une liste complète.

func filterPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "tagfilter"), nil
}

// LoadTagFilter retourne les tags du filtre actif (vide = pas de filtre).
func LoadTagFilter() ([]string, error) {
	p, err := filterPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var tags []string
	for _, l := range strings.Split(string(data), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			tags = append(tags, l)
		}
	}
	return tags, nil
}

// SaveTagFilter persiste le filtre (liste vide = suppression du filtre).
func SaveTagFilter(tags []string) error {
	p, err := filterPath()
	if err != nil {
		return err
	}
	if len(tags) == 0 {
		return ClearTagFilter()
	}
	return os.WriteFile(p, []byte(strings.Join(tags, "\n")+"\n"), 0o644)
}

// ClearTagFilter supprime le filtre actif.
func ClearTagFilter() error {
	p, err := filterPath()
	if err != nil {
		return err
	}
	err = os.Remove(p)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// MatchesFilter indique si le host possède TOUS les tags du filtre
// (sémantique ET : sélectionner prod+web ne garde que les hosts prod ET web).
func MatchesFilter(h Host, filter []string) bool {
	for _, t := range filter {
		if !slices.Contains(h.Tags, t) {
			return false
		}
	}
	return true
}
