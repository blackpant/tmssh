// Package ui — preview enrichi par host (appelé par fzf via --preview).
package ui

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/you/tmssh/internal/config"
	"github.com/you/tmssh/internal/history"
)

const (
	cSection = "\x1b[1;34m"  // titres de section : bleu gras
	cKey     = "\x1b[38;5;245m" // clés : gris
	cVal     = "\x1b[1;37m"  // valeurs : blanc gras
	cOk      = "\x1b[38;5;114m" // vert : host joignable
	cWarn    = "\x1b[38;5;203m" // rouge : host injoignable
	cDim     = "\x1b[38;5;240m" // gris sombre : séparateur
)

func sep(w int) string {
	return cDim + strings.Repeat("─", w) + cReset
}

func kv(key, val string) string {
	return fmt.Sprintf("  %s%-12s%s %s%s%s", cKey, key, cReset, cVal, val, cReset)
}

// PrintHostInfo écrit dans stdout le bloc preview pour le host nommé name.
// Contenu : icône + infos de connexion, tags, latence (ping TCP), historique.
func PrintHostInfo(cfg *config.Config, name string) error {
	h := cfg.Find(name)
	if h == nil {
		fmt.Fprintf(os.Stdout, "%s host %q introuvable%s\n", cWarn, name, cReset)
		return nil
	}

	w := 44 // largeur du preview

	// ── En-tête ────────────────────────────────
	fmt.Printf("\n  %s%s  %s%s%s\n", cIcon, cfg.IconFor(*h), cName, h.Name, cReset)
	fmt.Println(sep(w))

	// ── Infos de connexion ─────────────────────
	fmt.Printf("%s  Connexion%s\n", cSection, cReset)
	if h.HostName != "" {
		fmt.Println(kv("hostname", h.HostName))
	}
	if h.User != "" {
		fmt.Println(kv("user", h.User))
	}
	port := h.Port
	if port == "" {
		port = "22"
	}
	fmt.Println(kv("port", port))

	// ── Tags ───────────────────────────────────
	if len(h.Tags) > 0 {
		fmt.Println()
		fmt.Printf("%s  Tags%s\n", cSection, cReset)
		fmt.Printf("  %s\n", colorTags(h.Tags))
	}

	// ── Latence (ping TCP non bloquant, timeout 1 s) ──
	fmt.Println()
	fmt.Printf("%s  Réseau%s\n", cSection, cReset)
	target := h.HostName
	if target == "" {
		target = h.Name
	}
	latency, reachable := tcpPing(target, port, 1*time.Second)
	if reachable {
		fmt.Println(kv("latence", fmt.Sprintf("%s%s%s", cOk, latency.Round(time.Millisecond).String(), cReset)))
	} else {
		fmt.Println(kv("latence", fmt.Sprintf("%sinjoignable%s", cWarn, cReset)))
	}

	// ── Historique ─────────────────────────────
	entries, err := history.ForHost(name)
	if err == nil && len(entries) > 0 {
		fmt.Println()
		fmt.Printf("%s  Historique%s\n", cSection, cReset)
		e := entries[0]
		fmt.Println(kv("dernière co.", e.Last.Format("2006-01-02 15:04")))
		fmt.Println(kv("connexions", fmt.Sprintf("%d", e.Count)))
	}

	// ── Backups disponibles ────────────────────
	baks, _ := config.ListBackups()
	if len(baks) > 0 {
		fmt.Println()
		fmt.Printf("%s  ssh_config backups%s\n", cSection, cReset)
		limit := 3
		if len(baks) < limit {
			limit = len(baks)
		}
		for _, b := range baks[:limit] {
			// N'affiche que le nom du fichier, pas le chemin complet.
			parts := strings.Split(b, "/")
			fmt.Printf("  %s%s%s\n", cKey, parts[len(parts)-1], cReset)
		}
		if len(baks) > 3 {
			fmt.Printf("  %s+ %d autres…%s\n", cDim, len(baks)-3, cReset)
		}
	}

	fmt.Println()
	return nil
}

// tcpPing mesure la latence d'une connexion TCP vers host:port.
func tcpPing(host, port string, timeout time.Duration) (time.Duration, bool) {
	addr := net.JoinHostPort(host, port)
	start := time.Now()
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return 0, false
	}
	conn.Close()
	return time.Since(start), true
}

// PrintInfo est le point d'entrée appelé par `tmssh info <host>`.
func PrintInfo(cfg *config.Config, name string) error {
	return PrintHostInfo(cfg, name)
}

// uplookupSSHConfigIdentity retourne la ligne IdentityFile du host
// telle que définie dans ssh_config (optionnel, usage futur).
func uplookupSSHConfigIdentity(host string) string {
	out, err := exec.Command("ssh", "-G", host).Output()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(strings.ToLower(line), "identityfile ") {
			parts := strings.SplitN(line, " ", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}
