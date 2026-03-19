package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Setup a temporary config file
	home, _ := os.UserHomeDir()
	configDir := filepath.Join(home, ".config", "nekotree")
	os.MkdirAll(configDir, 0755)

	configPath := filepath.Join(configDir, "config.yaml")
	content := []byte("worktree_root: /tmp/trees\ndefault_image: golang:alpine\n")
	os.WriteFile(configPath, content, 0644)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.DefaultImage != "golang:alpine" {
		t.Errorf("Expected DefaultImage 'golang:alpine', got %s", cfg.DefaultImage)
	}
}

func TestConfigDefaults(t *testing.T) {
	// Test the fallback logic when file is missing
	os.Remove(filepath.Join(os.Getenv("HOME"), ".config/nekotree/config.yaml"))

	cfg, _ := Load()
	if cfg.DefaultImage == "" {
		t.Error("Config should have a default image even if file is missing")
	}
}
