package config

import (
	"encoding/json"
	"fmt"
	"os"

	"cubicheart.com/munchtoast/nekotree/internal/utils"
)

// Config represents the nekotree-config.json structure
type Config struct {
	ComposeFile string `json:"compose_file"`
}

// Load reads and parses the configuration file safely
func Load(configPath string) (*Config, error) {
	// 1. Sanitize path to prevent G304 (Potential file inclusion)
	safePath, err := utils.SanitizePath(configPath)
	if err != nil {
		return nil, fmt.Errorf("security violation: %w", err)
	}

	// 2. Read file using validated path
	// #nosec G304 - Path is validated to prevent directory traversal in utils.SanitizePath
	data, err := os.ReadFile(safePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read config at %s: %w", safePath, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config JSON: %w", err)
	}

	// 3. Ensure internal paths in the config are also safe
	if cfg.ComposeFile != "" {
		if _, err := utils.SanitizePath(cfg.ComposeFile); err != nil {
			return nil, fmt.Errorf("invalid compose_file path in config: %w", err)
		}
	}

	return &cfg, nil
}
