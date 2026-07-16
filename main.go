// tmssh — sélecteur de connexions SSH pour tmux (fzf + tags + historique)
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/blackpant/tmssh/internal/config"
	"github.com/blackpant/tmssh/internal/history"
	"github.com/blackpant/tmssh/internal/tmuxctl"
	"github.com/blackpant/tmssh/internal/ui"
)

const usage = `tmssh — SSH connection picker for tmux

Usage:
  tmssh                lance l'UI fzf (dans un popup tmux)
  tmssh list           liste formatée des hosts (consommée par fzf)
  tmssh connect H...   ouvre des panes/windows tmux (H = host ou @groupe)
  tmssh tag add  H...  ajoute un tag aux hosts sélectionnés (prompt fzf)
  tmssh tag rm   H...  retire un tag des hosts sélectionnés (prompt fzf)
  tmssh tags           liste tous les tags connus
  tmssh tagfilter pick   ouvre le tag picker (filtre par tags, sémantique ET)
  tmssh tagfilter clear  efface le filtre par tags
  tmssh groups         liste les groupes de connexion
  tmssh group save H...  sauvegarde un groupe avec ces hosts (prompt du nom)
  tmssh group rm <nom>   supprime un groupe
  tmssh group pick       picker de groupes (Enter = connexion)
  tmssh group list       liste formatée des groupes (consommée par fzf)
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
		if len(args) < 2 {
			return fmt.Errorf("connect: aucun host fourni")
		}
		// Les @groupes sont étendus en hosts individuels : l'historique
		// n'enregistre que des hosts, jamais des groupes.
		hosts, err := cfg.ExpandGroups(args[1:])
		if err != nil {
			return err
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

	case "group":
		if len(args) < 2 {
			return fmt.Errorf("group: usage tmssh group save|rm|pick|list")
		}
		switch args[1] {
		case "save":
			if len(args) < 3 {
				return fmt.Errorf("group save: aucun host fourni")
			}
			return ui.GroupSave(cfg, args[2:])
		case "rm":
			if len(args) < 3 {
				return fmt.Errorf("group rm: usage tmssh group rm <nom>")
			}
			return cfg.RemoveGroup(args[2])
		case "pick":
			return ui.GroupPick(cfg)
		case "list":
			return ui.PrintGroupList(cfg, os.Stdout)
		default:
			return fmt.Errorf("group: action inconnue %q", args[1])
		}

	case "groups":
		for _, name := range cfg.GroupNames() {
			n := 0
			if hosts, err := cfg.ResolveGroup(name); err == nil {
				n = len(hosts)
			}
			line := fmt.Sprintf("%-20s %d hosts", name, n)
			if g := cfg.Groups[name]; len(g.Tags) > 0 {
				line += "  tags: " + strings.Join(g.Tags, ",")
			}
			fmt.Println(line)
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
