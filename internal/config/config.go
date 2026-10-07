// Package config loads ~/.config/replayer/config.json.
package config

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

type Config struct {
	// DemosDir is the Rocket League replay folder.
	DemosDir string `json:"demos_dir"`
	// Players are the in-game names to coach (you and your mates).
	// The first one is "you". Empty means: the player who recorded the replay.
	Players []string `json:"players"`
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
	}
	p, err := Path()
	if err != nil {
		return c, err
	}
	b, err := os.ReadFile(p)
	if errors.Is(err, fs.ErrNotExist) {
		return c, nil
	}
	if err != nil {
		return c, err
	}
	err = json.Unmarshal(b, &c)
	return c, err
}
