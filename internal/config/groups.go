package config

import (
	"fmt"
	"slices"
	"sort"
	"strings"
)

// Group est un groupe de connexion nommé : l'union de hosts explicites
// et des hosts possédant tous les tags (sémantique ET, comme le filtre).
// Le champ tags s'édite directement dans hosts.json (comme les icônes) ;
// l'UI ne crée que des groupes à hosts explicites.
type Group struct {
	Hosts []string `json:"hosts,omitempty"`
	Tags  []string `json:"tags,omitempty"`
}

// ValidateGroupName rejette les noms inutilisables par `connect @nom`.
func ValidateGroupName(name string) error {
	if name == "" {
		return fmt.Errorf("nom de groupe vide")
	}
	if strings.ContainsAny(name, "@ \t") {
		return fmt.Errorf("nom de groupe %q invalide (ni @ ni espace)", name)
	}
	return nil
}

// GroupNames retourne les noms de groupes triés.
func (c *Config) GroupNames() []string {
	names := make([]string, 0, len(c.Groups))
	for n := range c.Groups {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// ResolveGroup retourne les hosts du groupe : d'abord les hosts explicites
// encore présents dans la config (les disparus sont ignorés), puis les
// hosts matchant tous les tags, dans l'ordre de hosts.json. Dédupliqué.
func (c *Config) ResolveGroup(name string) ([]string, error) {
	g, ok := c.Groups[name]
	if !ok {
		return nil, fmt.Errorf("groupe %q inconnu", name)
	}

	var out []string
	seen := map[string]bool{}
	add := func(n string) {
		if !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}

	for _, n := range g.Hosts {
		if c.Find(n) != nil {
			add(n)
		}
	}
	// Garde-fou : MatchesFilter avec un filtre vide matche tout.
	if len(g.Tags) > 0 {
		for _, h := range c.Hosts {
			if MatchesFilter(h, g.Tags) {
				add(h.Name)
			}
		}
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("groupe %q vide (hosts disparus ou aucun host avec ces tags)", name)
	}
	return out, nil
}

// SaveGroup crée ou remplace le groupe nommé avec ces hosts explicites.
// Les tags d'un groupe existant sont préservés : ils ne s'éditent que
// dans hosts.json et l'UI ne doit pas pouvoir les détruire.
func (c *Config) SaveGroup(name string, hosts []string) error {
	if err := ValidateGroupName(name); err != nil {
		return err
	}
	if len(hosts) == 0 {
		return fmt.Errorf("groupe %q : aucun host fourni", name)
	}
	if c.Groups == nil {
		c.Groups = map[string]Group{}
	}
	g := c.Groups[name]
	g.Hosts = slices.Clone(hosts)
	c.Groups[name] = g
	return c.Save()
}

// RemoveGroup supprime le groupe nommé.
func (c *Config) RemoveGroup(name string) error {
	if _, ok := c.Groups[name]; !ok {
		return fmt.Errorf("groupe %q inconnu", name)
	}
	delete(c.Groups, name)
	return c.Save()
}

// ExpandGroups remplace chaque @nom par les hosts résolus du groupe,
// laisse les autres noms tels quels, et déduplique le résultat.
func (c *Config) ExpandGroups(names []string) ([]string, error) {
	var out []string
	seen := map[string]bool{}
	add := func(n string) {
		if !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	for _, n := range names {
		if g, ok := strings.CutPrefix(n, "@"); ok {
			hosts, err := c.ResolveGroup(g)
			if err != nil {
				return nil, err
			}
			for _, h := range hosts {
				add(h)
			}
		} else {
			add(n)
		}
	}
	return out, nil
}
