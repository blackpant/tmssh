// tmssh — sélecteur de connexions SSH pour tmux (fzf + tags + historique)
package main

import (
	"fmt"
	"os"

	"github.com/blackpant/tmssh/internal/config"
	"github.com/blackpant/tmssh/internal/history"
	"github.com/blackpant/tmssh/internal/tmuxctl"
	"github.com/blackpant/tmssh/internal/ui"
)

const usage = `tmssh — SSH connection picker for tmux

Usage:
  tmssh                lance l'UI fzf (dans un popup tmux)
  tmssh list           liste formatée des hosts (consommée par fzf)
  tmssh connect H...   ouvre des panes/windows tmux vers les hosts H
  tmssh tag add  H...  ajoute un tag aux hosts sélectionnés (prompt fzf)
  tmssh tag rm   H...  retire un tag des hosts sélectionnés (prompt fzf)
  tmssh tags           liste tous les tags connus
  tmssh tagfilter pick   ouvre le tag picker (filtre par tags, sémantique ET)
  tmssh tagfilter clear  efface le filtre par tags
  tmssh sync           sync bidirectionnelle ~/.ssh/config <-> hosts.json
  tmssh info     H     preview enrichi d'un host (latence, tags, historique)
  tmssh history        affiche l'historique des connexions
  tmssh backups        liste les backups de ~/.ssh/config
`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "tmssh:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if len(args) == 0 {
		return ui.Run(cfg) // UI interactive par défaut
	}

	switch args[0] {
	case "list":
		return ui.PrintList(cfg, os.Stdout)

	case "connect":
		hosts := args[1:]
		if len(hosts) == 0 {
			return fmt.Errorf("connect: aucun host fourni")
		}
		// L'historique n'est enregistré que si la connexion a réussi.
		if err := tmuxctl.Connect(cfg, hosts); err != nil {
			return err
		}
		return history.Record(hosts)

	case "tag":
		if len(args) < 3 {
			return fmt.Errorf("tag: usage tmssh tag add|rm <host...>")
		}
		action, hosts := args[1], args[2:]
		switch action {
		case "add":
			return ui.TagAdd(cfg, hosts)
		case "rm":
			return ui.TagRemove(cfg, hosts)
		default:
			return fmt.Errorf("tag: action inconnue %q", action)
		}

	case "tagfilter":
		if len(args) < 2 {
			return fmt.Errorf("tagfilter: usage tmssh tagfilter pick|clear")
		}
		switch args[1] {
		case "pick":
			return ui.TagFilterPick(cfg)
		case "clear":
			return config.ClearTagFilter()
		default:
			return fmt.Errorf("tagfilter: action inconnue %q", args[1])
		}

	case "tags":
		for _, t := range cfg.AllTags() {
			fmt.Println(t)
		}
		return nil

	case "info":
		if len(args) < 2 {
			return fmt.Errorf("info: usage tmssh info <host>")
		}
		return ui.PrintInfo(cfg, args[1])

	case "backups":
		baks, err := config.ListBackups()
		if err != nil {
			return err
		}
		if len(baks) == 0 {
			fmt.Println("aucun backup disponible")
			return nil
		}
		for _, b := range baks {
			fmt.Println(b)
		}
		return nil

	case "sync":
		return config.Sync(cfg)

	case "history":
		return history.Print(os.Stdout)

	case "-h", "--help", "help":
		fmt.Print(usage)
		return nil

	default:
		return fmt.Errorf("commande inconnue %q\n\n%s", args[0], usage)
	}
}
