package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "nekotree-config.json")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}
	return path
}

// --- Load: valid configs ---

func TestLoad_ValidConfig(t *testing.T) {
	path := writeTempConfig(t, `{"compose_file": "docker-compose.yaml"}`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.ComposeFile != "docker-compose.yaml" {
		t.Errorf("expected docker-compose.yaml, got %s", cfg.ComposeFile)
	}
}

func TestLoad_EmptyComposeFile(t *testing.T) {
	// compose_file is optional; empty JSON should load fine
	path := writeTempConfig(t, `{}`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.ComposeFile != "" {
		t.Errorf("expected empty compose_file, got %s", cfg.ComposeFile)
	}
}

func TestLoad_EmptyJSON(t *testing.T) {
	path := writeTempConfig(t, `{}`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed for empty JSON: %v", err)
	}
	if cfg.ComposeFile != "" {
		t.Errorf("expected zero-value config, got %+v", cfg)
	}
}

// --- Load: error cases ---

func TestLoad_MissingFile(t *testing.T) {
	cfg, err := Load("/tmp/definitely-does-not-exist.json")
	if err != nil {
		t.Errorf("expected nil error for missing file, got: %v", err)
	}
	if cfg != nil {
		t.Errorf("expected nil config for missing file, got: %+v", cfg)
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	path := writeTempConfig(t, `{not valid json}`)

	_, err := Load(path)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestLoad_DirectoryTraversalPath(t *testing.T) {
	_, err := Load("../../etc/passwd")
	if err == nil {
		t.Error("expected security error for directory traversal path")
	}
}

func TestLoad_ComposeFileWithTraversal(t *testing.T) {
	// compose_file in the JSON itself contains a traversal path
	path := writeTempConfig(t, `{"compose_file": "../../etc/docker-compose.yaml"}`)

	_, err := Load(path)
	if err == nil {
		t.Error("expected security error for traversal in compose_file field")
	}
}
