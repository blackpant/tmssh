// Package tmuxctl ouvre les connexions SSH dans tmux (panes ou windows).
package tmuxctl

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/blackpant/tmssh/internal/config"
)

// Connect ouvre une connexion SSH par host :
//   - 1 host  : nouvelle window
//   - N hosts : une window découpée en N panes (layout tiled)
//   - option TMSSH_SYNC=1 pour activer synchronize-panes.
func Connect(cfg *config.Config, names []string) error {
	if os.Getenv("TMUX") == "" {
		return fmt.Errorf("pas dans une session tmux")
	}

	if len(names) == 1 {
		return tmux("new-window", "-n", names[0], sshCmd(names[0]))
	}

	// Première connexion : nouvelle window. On récupère son window_id
	// (unique, contrairement au nom) pour cibler les splits suivants :
	// une window "ssh-multi" d'une session précédente ne doit pas
	// recevoir les nouveaux panes.
	out, err := exec.Command("tmux", "new-window", "-P", "-F", "#{window_id}",
		"-n", "ssh-multi", sshCmd(names[0])).Output()
	if err != nil {
		return err
	}
	win := strings.TrimSpace(string(out))

	// Les suivantes : split de la window fraîchement créée.
	for _, name := range names[1:] {
		if err := tmux("split-window", "-t", win, sshCmd(name)); err != nil {
			return err
		}
		if err := tmux("select-layout", "-t", win, "tiled"); err != nil {
			return err
		}
	}

	if os.Getenv("TMSSH_SYNC") == "1" {
		return tmux("set-window-option", "-t", win, "synchronize-panes", "on")
	}
	return nil
}

func sshCmd(host string) string {
	// ssh_config fait autorité pour user/port : on passe juste l'alias.
	return "ssh " + host
}

func tmux(args ...string) error {
	cmd := exec.Command("tmux", args...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return cmd.Run()
}
