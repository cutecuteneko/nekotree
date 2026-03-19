package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	// Create a temporary config file
	tmpFile := "test-config.json"
	content := `{"compose_file": "docker-compose.yml", "service": "app"}`
	os.WriteFile(tmpFile, []byte(content), 0644)
	defer os.Remove(tmpFile)

	t.Run("Valid Config", func(t *testing.T) {
		cfg, err := Load(tmpFile)
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}
		if cfg.ComposeFile != "docker-compose.yml" {
			t.Errorf("Expected docker-compose.yml, got %s", cfg.ComposeFile)
		}
	})

	t.Run("Missing File", func(t *testing.T) {
		_, err := Load("non-existent.json")
		if err == nil {
			t.Error("Expected error for missing file, got nil")
		}
	})
}
