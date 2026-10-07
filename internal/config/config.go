// Package config loads ~/.config/replayer/config.json.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"replayer/internal/rank"
)

type Config struct {
	// DemosDir is the Rocket League replay folder.
	DemosDir string `json:"demos_dir"`
	// Players are the in-game names to coach (you and your mates).
	// The first one is "you". Empty means: the player who recorded the replay.
	Players []string `json:"players"`
	// Workdir holds the analysis cache, the benchmarks, the coaching
	// sessions and the coach memory.
	Workdir string `json:"workdir"`
	// Rank is the default rank of the coached players (not in the replays),
	// Ranks overrides it per playlist (ballchasing names, e.g. "ranked-duels").
	Rank  string            `json:"rank"`
	Ranks map[string]string `json:"ranks"`
	// Aliases maps other in-game names of a player to their main name
	// (e.g. a second account), applied to every replay.
	Aliases map[string]string `json:"aliases"`
	// BallchasingToken enables the ballchasing.com benchmark. The
	// BALLCHASING_TOKEN environment variable takes precedence.
	BallchasingToken string `json:"ballchasing_token"`
}

func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "replayer", "config.json"), nil
}

// Load returns the defaults overridden by the config file, if it exists.
func Load() (Config, error) {
	home, _ := os.UserHomeDir()
	c := Config{
		DemosDir: filepath.Join(home, ".var/app/com.valvesoftware.Steam/.local/share/Steam/steamapps/compatdata/252950/pfx/drive_c/users/steamuser/Documents/My Games/Rocket League/TAGame/Demos"),
		Workdir:  "~/Documents/replayer-data",
	}
	p, err := Path()
	if err != nil {
		return c, err
	}
	b, err := os.ReadFile(p)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return c, err
	}
	if err == nil {
		if err := json.Unmarshal(b, &c); err != nil {
			return c, fmt.Errorf("%s: %w", p, err)
		}
	}
	if t := os.Getenv("BALLCHASING_TOKEN"); t != "" {
		c.BallchasingToken = t
	}
	if rest, ok := strings.CutPrefix(c.Workdir, "~/"); ok {
		c.Workdir = filepath.Join(home, rest)
	}
	if c.Rank != "" {
		if c.Rank, err = rank.Normalize(c.Rank); err != nil {
			return c, fmt.Errorf("%s: rank: %w", p, err)
		}
	}
	for pl, r := range c.Ranks {
		if c.Ranks[pl], err = rank.Normalize(r); err != nil {
			return c, fmt.Errorf("%s: ranks.%s: %w", p, pl, err)
		}
	}
	return c, nil
}

// RankFor returns the configured rank for a playlist (ballchasing name),
// empty if none.
func (c Config) RankFor(playlist string) string {
	if r, ok := c.Ranks[playlist]; ok {
		return r
	}
	return c.Rank
}
