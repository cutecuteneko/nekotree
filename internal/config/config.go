package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	WorktreeRoot string `yaml:"worktree_root"`
	DefaultImage string `yaml:"default_image"`
	ComposeFile  string `yaml:"compose_file,omitempty"`
}

func Load() (*Config, error) {
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".config", "nekotree", "config.yaml")

	data, err := os.ReadFile(configPath)
	if err != nil {
		// Return sensible defaults if no config exists
		return &Config{
			WorktreeRoot: filepath.Join(home, "Documents", "worktrees"),
			DefaultImage: "alpine",
		}, nil
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
