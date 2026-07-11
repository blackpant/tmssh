// Package ui pilote fzf : liste des hosts, multi-select, tags, tag picker.
package ui

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"unicode/utf8"

	"github.com/you/tmssh/internal/config"
)

// ---------------------------------------------------------------------------
// Couleurs
// ---------------------------------------------------------------------------

const (
	cReset = "\x1b[0m"
	cName  = "\x1b[1;37m"  // nom du host : blanc gras
	cUser  = "\x1b[38;5;245m" // user@hostname : gris
	cIcon  = "\x1b[38;5;110m" // icône : bleu doux
)

// Couleurs fixes pour les tags usuels ; les autres reçoivent une couleur
// stable dérivée d'un hash du nom (même tag = même couleur à chaque fois).
var tagColors = map[string]string{
	"prod":    "\x1b[38;5;203m", // rouge
	"staging": "\x1b[38;5;215m", // orange
	"dev":     "\x1b[38;5;114m", // vert
	"web":     "\x1b[38;5;117m", // cyan
	"db":      "\x1b[38;5;141m", // violet
}

var fallbackPalette = []string{
	"\x1b[38;5;110m", "\x1b[38;5;150m", "\x1b[38;5;180m",
	"\x1b[38;5;175m", "\x1b[38;5;109m", "\x1b[38;5;144m",
}

func tagColor(tag string) string {
	if c, ok := tagColors[tag]; ok {
		return c
	}
	var h uint32
	for _, r := range tag { // hash FNV-ish minimal
		h = h*31 + uint32(r)
	}
	return fallbackPalette[h%uint32(len(fallbackPalette))]
}

func colorTags(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	parts := make([]string, len(tags))
	for i, t := range tags {
		parts[i] = tagColor(t) + t + cReset
	}
	return cUser + "[" + cReset + strings.Join(parts, cUser+","+cReset) + cUser + "]" + cReset
}

// Thème fzf (palette sombre douce, style tokyonight). Adapter librement.
var fzfTheme = "--color=" + strings.Join([]string{
	"bg+:#2d3149", "fg:#c0caf5", "fg+:#c0caf5", "hl:#7aa2f7", "hl+:#7dcfff",
	"info:#7aa2f7", "prompt:#7dcfff", "pointer:#f7768e", "marker:#9ece6a",
	"spinner:#9ece6a", "header:#565f89", "border:#3b4261", "query:#c0caf5",
}, ",")

// ---------------------------------------------------------------------------
// Liste
// ---------------------------------------------------------------------------

// PrintList écrit la liste des hosts au format consommé par fzf :
//
//	<name>\t<icône> <name>  <user>@<hostname>  [tags colorés]
//
// La 1re colonne (cachée par --with-nth=2..) sert de clé stable pour {+1}.
// Les largeurs de colonnes sont calculées dynamiquement pour un alignement
// propre quel que soit le contenu, et le padding est fait AVANT la
// colorisation (les codes ANSI ne comptent donc pas dans la largeur).
func PrintList(cfg *config.Config, w io.Writer) error {
	filter, err := config.LoadTagFilter()
	if err != nil {
		return err
	}

	type row struct {
		name, target string
		tags         []string
		icon         string
	}
	var rows []row
	wName, wTarget := 0, 0

	for _, h := range cfg.Hosts {
		if !config.MatchesFilter(h, filter) {
			continue
		}
		target := h.HostName
		if h.User != "" {
			target = h.User + "@" + target
		}
		if h.Port != "" && h.Port != "22" {
			target += ":" + h.Port
		}
		r := row{name: h.Name, target: target, tags: h.Tags, icon: cfg.IconFor(h)}
		rows = append(rows, r)
		wName = max(wName, utf8.RuneCountInString(r.name))
		wTarget = max(wTarget, utf8.RuneCountInString(r.target))
	}

	for _, r := range rows {
		fmt.Fprintf(w, "%s\t%s%s%s  %s%-*s%s  %s%-*s%s  %s\n",
			r.name,
			cIcon, r.icon, cReset,
			cName, wName, r.name, cReset,
			cUser, wTarget, r.target, cReset,
			colorTags(r.tags),
		)
	}
	return nil
}

// ---------------------------------------------------------------------------
// UI principale
// ---------------------------------------------------------------------------

// Run lance l'UI fzf principale.
//
// Bindings :
//   - Tab      : multi-sélection
//   - ctrl-t   : sélectionne TOUS les hosts affichés (résultat du filtre/recherche)
//   - Enter    : tmssh connect <hosts>
//   - ctrl-a   : tmssh tag add <hosts>       puis reload
//   - ctrl-d   : tmssh tag rm  <hosts>       puis reload
//   - ctrl-f   : tag picker (filtre par tags) puis reload
//   - ctrl-g   : efface le filtre par tags    puis reload
//   - ctrl-h   : affiche l'historique en preview
func Run(cfg *config.Config) error {
	// Chaque lancement repart sans filtre : le filtre ne persiste
	// qu'entre les reloads d'une même session fzf.
	if err := config.ClearTagFilter(); err != nil {
		return err
	}

	self, err := os.Executable()
	if err != nil {
		return err
	}

	args := []string{
		"--multi",
		"--ansi",
		"--delimiter=\t",
		"--with-nth=2..", // cache la colonne clé
		"--prompt=ssh ❯ ",
		"--pointer=▶", "--marker=▌",
		"--header=Tab: sélection ▪ C-t: tout ▪ Enter: connect ▪ C-a/C-d: ±tag ▪ C-f: filtre tags ▪ C-g: reset ▪ C-h: historique",
		fzfTheme,
		"--preview=" + self + " info {1}",
		"--preview-window=right,45%,wrap",
		"--bind=ctrl-h:toggle-preview",
		"--bind=ctrl-t:select-all", // select-all n'agit que sur les lignes matchées
		fmt.Sprintf("--bind=ctrl-a:execute(%s tag add {+1})+reload(%s list)", self, self),
		fmt.Sprintf("--bind=ctrl-d:execute(%s tag rm {+1})+reload(%s list)", self, self),
		fmt.Sprintf("--bind=ctrl-f:execute(%s tagfilter pick)+reload(%s list)+first", self, self),
		fmt.Sprintf("--bind=ctrl-g:execute-silent(%s tagfilter clear)+reload(%s list)+first", self, self),
	}

	cmd := exec.Command("fzf", args...)
	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	var out strings.Builder
	cmd.Stdout = &out

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("fzf introuvable ? %w", err)
	}
	if err := PrintList(cfg, stdin); err != nil {
		return err
	}
	stdin.Close()

	if err := cmd.Wait(); err != nil {
		// Code 130 = annulation (Esc / ctrl-c) : pas une erreur.
		if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode() == 130 {
			return nil
		}
		return err
	}

	// Récupère la clé (colonne 1) de chaque ligne sélectionnée.
	var names []string
	sc := bufio.NewScanner(strings.NewReader(out.String()))
	for sc.Scan() {
		if name, _, ok := strings.Cut(sc.Text(), "\t"); ok {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return nil
	}

	connect := exec.Command(self, append([]string{"connect"}, names...)...)
	connect.Stdout, connect.Stderr = os.Stdout, os.Stderr
	return connect.Run()
}

// ---------------------------------------------------------------------------
// Tag picker (filtre)
// ---------------------------------------------------------------------------

// TagFilterPick ouvre un fzf multi-select sur l'ensemble des tags connus
// et enregistre la sélection comme filtre actif (sémantique ET).
// Les tags du filtre courant sont présélectionnés... via re-proposition simple.
func TagFilterPick(cfg *config.Config) error {
	all := cfg.AllTags()
	if len(all) == 0 {
		fmt.Fprintln(os.Stderr, "aucun tag défini")
		return nil
	}

	var colored []string
	for _, t := range all {
		colored = append(colored, t+"\t"+tagColor(t)+t+cReset)
	}

	cmd := exec.Command("fzf",
		"--multi",
		"--ansi",
		"--delimiter=\t",
		"--with-nth=2",
		"--prompt=filtre tags ❯ ",
		"--header=Tab: sélection ▪ Enter: appliquer ▪ Esc: annuler",
		"--height=~50%",
		fzfTheme,
	)
	cmd.Stdin = strings.NewReader(strings.Join(colored, "\n"))
	cmd.Stderr = os.Stderr

	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok && (ee.ExitCode() == 130 || ee.ExitCode() == 1) {
			return nil // annulé : on garde le filtre existant
		}
		return err
	}

	var tags []string
	for _, l := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
		if name, _, ok := strings.Cut(l, "\t"); ok {
			tags = append(tags, name)
		}
	}
	return config.SaveTagFilter(tags)
}

// ---------------------------------------------------------------------------
// Prompts de tags (add / rm)
// ---------------------------------------------------------------------------

// TagAdd demande un tag (fzf en mode saisie libre + suggestions) et
// l'applique aux hosts donnés.
func TagAdd(cfg *config.Config, hosts []string) error {
	tag, err := pickTag("Ajouter un tag ❯ ", cfg.AllTags())
	if err != nil || tag == "" {
		return err
	}
	return cfg.AddTag(tag, hosts)
}

// TagRemove propose les tags communs aux hosts sélectionnés et retire
// celui choisi.
func TagRemove(cfg *config.Config, hosts []string) error {
	common := cfg.CommonTags(hosts)
	if len(common) == 0 {
		fmt.Fprintln(os.Stderr, "aucun tag commun à retirer")
		return nil
	}
	tag, err := pickTag("Retirer un tag ❯ ", common)
	if err != nil || tag == "" {
		return err
	}
	return cfg.RemoveTag(tag, hosts)
}

// pickTag ouvre un fzf secondaire : suggestions + saisie libre (--print-query).
func pickTag(prompt string, suggestions []string) (string, error) {
	cmd := exec.Command("fzf",
		"--prompt="+prompt,
		"--print-query", // permet de créer un tag inexistant
		"--height=~40%",
		fzfTheme,
	)
	cmd.Stdin = strings.NewReader(strings.Join(suggestions, "\n"))
	cmd.Stderr = os.Stderr

	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok && (ee.ExitCode() == 130 || ee.ExitCode() == 1) {
			lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
			if ee.ExitCode() == 1 && len(lines) > 0 && lines[0] != "" {
				return strings.TrimSpace(lines[0]), nil
			}
			return "", nil
		}
		return "", err
	}

	// Sortie : ligne 1 = query, ligne 2 = sélection (si match).
	lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	if len(lines) >= 2 && lines[1] != "" {
		return strings.TrimSpace(lines[1]), nil
	}
	return strings.TrimSpace(lines[0]), nil
}
