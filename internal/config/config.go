package config

import (
    "fmt"
    "os"
)

type Config struct {
    BaseImage       string
    WorktreeRoot    string
    FeatureBranch   string
    ContainerName   string
}

func Load() (*Config, error) {
    return &Config{
        BaseImage:     os.Getenv("DEVENV_BASE_IMAGE"),
        WorktreeRoot:  os.Getenv("DEVENV_WORKTREE_ROOT"),
        FeatureBranch: os.Getenv("DEVENV_FEATURE_BRANCH"),
        ContainerName: os.Getenv("DEVENV_CONTAINER_NAME"),
    }, nil
}

func (c *Config) Validate() error {
    if c.BaseImage == "" || c.WorktreeRoot == "" || c.FeatureBranch == "" {
        return fmt.Errorf("missing required config")
    }
    return nil
}

