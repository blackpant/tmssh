// Package ui — picker et sauvegarde des groupes de connexion.
package ui

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"unicode/utf8"

	"github.com/blackpant/tmssh/internal/config"
)

// PrintGroupList écrit la liste des groupes au format consommé par fzf :
//
//	<nom>\t@<nom>  <N> hosts  [tags colorés]
//
// Même convention que PrintList : colonne 1 = clé cachée, padding calculé
// avant la colorisation.
func PrintGroupList(cfg *config.Config, w io.Writer) error {
	type row struct {
		name  string
		count int
		tags  []string
	}
	var rows []row
	wName := 0

	for _, name := range cfg.GroupNames() {
		n := 0
		if hosts, err := cfg.ResolveGroup(name); err == nil {
			n = len(hosts)
		}
		rows = append(rows, row{name: name, count: n, tags: cfg.Groups[name].Tags})
		wName = max(wName, utf8.RuneCountInString("@"+name))
	}

	for _, r := range rows {
		fmt.Fprintf(w, "%s\t%s%-*s%s  %s%d hosts%s  %s\n",
			r.name,
			cName, wName, "@"+r.name, cReset,
			cUser, r.count, cReset,
			colorTags(r.tags),
		)
	}
	return nil
}

// GroupPick ouvre le picker de groupes (binding ctrl-e de l'UI principale,
// via become : il remplace le fzf principal).
//
// Bindings :
//   - Tab    : multi-sélection (connexion à l'union des groupes)
//   - Enter  : tmssh connect @nom...
//   - ctrl-d : supprime le groupe sous le curseur, puis reload
//   - Esc    : quitte sans rien faire
func GroupPick(cfg *config.Config) error {
	if len(cfg.Groups) == 0 {
		fmt.Fprintln(os.Stderr, "aucun groupe défini (ctrl-s dans l'UI pour en créer un)")
		return nil
	}

	self, err := os.Executable()
	if err != nil {
		return err
	}

	cmd := exec.Command("fzf",
		"--multi",
		"--ansi",
		"--delimiter=\t",
		"--with-nth=2..",
		"--prompt=groupe ❯ ",
		"--pointer=▶", "--marker=▌",
		"--header=Tab: sélection ▪ Enter: connect ▪ C-d: supprimer ▪ Esc: quitter",
		fzfTheme,
		fmt.Sprintf("--bind=ctrl-d:execute-silent(%s group rm {1})+reload(%s group list)", self, self),
	)

	var in strings.Builder
	if err := PrintGroupList(cfg, &in); err != nil {
		return err
	}
	cmd.Stdin = strings.NewReader(in.String())
	cmd.Stderr = os.Stderr

	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok && (ee.ExitCode() == 130 || ee.ExitCode() == 1) {
			return nil // annulé ou liste vide après suppressions
		}
		return err
	}

	var names []string
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	for sc.Scan() {
		if name, _, ok := strings.Cut(sc.Text(), "\t"); ok {
			names = append(names, "@"+name)
		}
	}
	if len(names) == 0 {
		return nil
	}

	connect := exec.Command(self, append([]string{"connect"}, names...)...)
	connect.Stdout, connect.Stderr = os.Stdout, os.Stderr
	return connect.Run()
}

// GroupSave demande un nom (fzf en saisie libre + groupes existants en
// suggestion, l'écrasement conservant les tags) et sauvegarde le groupe.
func GroupSave(cfg *config.Config, hosts []string) error {
	name, err := pickTag("Nom du groupe ❯ ", cfg.GroupNames())
	if err != nil || name == "" {
		return err
	}
	return cfg.SaveGroup(name, hosts)
}
