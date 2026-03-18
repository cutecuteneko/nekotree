package config

import (
    "os"
    "testing"
)

func TestLoad(t *testing.T) {
    os.Setenv("DEVENV_BASE_IMAGE", "test-image")
    os.Setenv("DEVENV_WORKTREE_ROOT", "/tmp/worktree")
    defer func() {
        os.Unsetenv("DEVENV_BASE_IMAGE")
        os.Unsetenv("DEVENV_WORKTREE_ROOT")
    }()

    cfg, err := Load()
    if err != nil {
        t.Fatalf("Load failed: %v", err)
    }
    if cfg.BaseImage != "test-image" {
        t.Errorf("expected base image 'test-image', got '%s'", cfg.BaseImage)
    }
}

func TestValidate(t *testing.T) {
    cfg := &Config{BaseImage: "", WorktreeRoot: "", FeatureBranch: ""}
    err := cfg.Validate()
    if err == nil {
        t.Error("expected validation error for missing fields")
    }
}

