// Command replayer browses Rocket League replays in a TUI and launches a
// Claude coaching session on the selected one.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"replayer/internal/config"
	"replayer/internal/replay"
	"replayer/internal/tui"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "config:", err)
		os.Exit(1)
	}
	dir := flag.String("dir", cfg.DemosDir, "dossier des replays")
	players := flag.String("players", strings.Join(cfg.Players, ","), "pseudos à coacher, séparés par des virgules (le premier = toi)")
	flag.Parse()

	cfg.DemosDir = *dir
	cfg.Players = nil
	for _, p := range strings.Split(*players, ",") {
		if p = strings.TrimSpace(p); p != "" {
			cfg.Players = append(cfg.Players, p)
		}
	}

	replays, errs := replay.Scan(cfg.DemosDir)
	if len(replays) == 0 && len(errs) > 0 {
		fmt.Fprintln(os.Stderr, "lecture des replays:", errs[0])
		os.Exit(1)
	}
	for _, r := range replays {
		r.ApplyAliases(cfg.Aliases)
	}

	if _, err := tea.NewProgram(tui.New(replays, cfg, errs), tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
