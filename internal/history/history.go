// Package history journalise les connexions dans history.json.
package history

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/blackpant/tmssh/internal/config"
)

// Entry est une statistique de connexion par host.
type Entry struct {
	Host  string    `json:"host"`
	Count int       `json:"count"`
	Last  time.Time `json:"last"`
}

func path() (string, error) {
	dir, err := config.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "history.json"), nil
}

func load() (map[string]*Entry, error) {
	p, err := path()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return map[string]*Entry{}, nil
	}
	if err != nil {
		return nil, err
	}
	var entries []*Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	m := make(map[string]*Entry, len(entries))
	for _, e := range entries {
		m[e.Host] = e
	}
	return m, nil
}

func save(m map[string]*Entry) error {
	entries := make([]*Entry, 0, len(m))
	for _, e := range m {
		entries = append(entries, e)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Last.After(entries[j].Last) })

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	p, err := path()
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(p), ".history-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), p)
}

// Record incrémente le compteur des hosts au moment de la connexion.
func Record(hosts []string) error {
	m, err := load()
	if err != nil {
		return err
	}
	now := time.Now()
	for _, h := range hosts {
		if e, ok := m[h]; ok {
			e.Count++
			e.Last = now
		} else {
			m[h] = &Entry{Host: h, Count: 1, Last: now}
		}
	}
	return save(m)
}

// ForHost retourne la slice des entrees pour un host donne (0 ou 1 element).
// Utilise par le preview enrichi.
func ForHost(name string) ([]*Entry, error) {
	m, err := load()
	if err != nil {
		return nil, err
	}
	if e, ok := m[name]; ok {
		return []*Entry{e}, nil
	}
	return nil, nil
}

// Print affiche l'historique trié par récence (consommé par le preview fzf).
func Print(w io.Writer) error {
	m, err := load()
	if err != nil {
		return err
	}
	entries := make([]*Entry, 0, len(m))
	for _, e := range m {
		entries = append(entries, e)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Last.After(entries[j].Last) })

	for _, e := range entries {
		fmt.Fprintf(w, "%-20s ×%-4d %s\n", e.Host, e.Count, e.Last.Format("2006-01-02 15:04"))
	}
	return nil
}
