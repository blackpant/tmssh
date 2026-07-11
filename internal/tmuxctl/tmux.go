// Package tmuxctl ouvre les connexions SSH dans tmux (panes ou windows).
package tmuxctl

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/you/tmssh/internal/config"
)

// Connect ouvre une connexion SSH par host :
//   - 1 host  : nouvelle window
//   - N hosts : une window découpée en N panes (layout tiled)
//     + option TMSSH_SYNC=1 pour activer synchronize-panes.
func Connect(cfg *config.Config, names []string) error {
	if os.Getenv("TMUX") == "" {
		return fmt.Errorf("pas dans une session tmux")
	}

	if len(names) == 1 {
		return tmux("new-window", "-n", names[0], sshCmd(names[0]))
	}

	// Première connexion : nouvelle window.
	if err := tmux("new-window", "-n", "ssh-multi", sshCmd(names[0])); err != nil {
		return err
	}
	// Les suivantes : split de la window courante.
	for _, name := range names[1:] {
		if err := tmux("split-window", "-t", "ssh-multi", sshCmd(name)); err != nil {
			return err
		}
		if err := tmux("select-layout", "-t", "ssh-multi", "tiled"); err != nil {
			return err
		}
	}

	if os.Getenv("TMSSH_SYNC") == "1" {
		return tmux("set-window-option", "-t", "ssh-multi", "synchronize-panes", "on")
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
