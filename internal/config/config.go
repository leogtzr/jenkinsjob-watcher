package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Interval      int    `toml:"interval"`
	NotifyCommand string `toml:"notify_command"`
}

func Load() (Config, error) {
	cfg := Config{
		Interval: 20,
	}

	candidates := []string{}

	// cwd:
	candidates = append(candidates, "watch.toml")
	// Users config dir:
	if configDir, err := os.UserConfigDir(); err == nil {
		candidates = append(candidates, filepath.Join(configDir, "jenkinsjob-watcher", "watch.toml"))
	}

	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue // keep looking...
			}
			return cfg, fmt.Errorf("error reading %s: %w", path, err)
		}

		if err := toml.Unmarshal(data, &cfg); err != nil {
			return cfg, fmt.Errorf("error parsing %s: %w", path, err)
		}

		// Found and loaded a valid one:
		_, _ = fmt.Fprintf(os.Stderr, "Using configuration: %s\n", path)
		return cfg, nil
	}

	return cfg, nil
}
