package volumes

import (
    "fmt"
    "os"
    "path/filepath"
	"strings"
)

// Mount represents a Docker volume mount configuration
type Mount struct {
    HostPath      string
    ContainerPath string
    ReadOnly      bool
}

// MountManager handles volume mounting operations
type MountManager struct {
    BaseImage       string
    WorktreeRoot    string
    AdditionalMounts []Mount
}

// NewMountManager creates a new mount manager instance
func NewMountManager(worktreeRoot string, additionalMounts ...Mount) *MountManager {
    return &MountManager{
        WorktreeRoot:     worktreeRoot,
        AdditionalMounts: additionalMounts,
    }
}

// LoadFromConfig loads mounts from environment variables or config file
func (m *MountManager) LoadFromEnv() error {
    // Support DEVENV_MOUNTS env var: "host1:container1:host2:container2"
    if mountsStr := os.Getenv("DEVENV_MOUNTS"); mountsStr != "" {
        m.AdditionalMounts = parseMountString(mountsStr)
    }

    return nil
}

// LoadFromFile loads mounts from a config file (YAML/TOML/JSON)
func (m *MountManager) LoadFromFile(filename string) error {
    _, err := os.ReadFile(filepath.Join(os.Getenv("HOME"), filename))
    if err != nil && !os.IsNotExist(err) {
        return fmt.Errorf("failed to read config file: %w", err)
    }

    switch filepath.Ext(filename) {
    case ".yaml", ".yml":
        // TODO: Implement YAML parsing
        return nil
    case ".json":
        // TODO: Implement JSON parsing
        return nil
    default:
        return fmt.Errorf("unsupported config file format: %s", filename)
    }
}

// parseMountString parses a string like "host1:container1:host2:container2"
func parseMountString(s string) []Mount {
    var mounts []Mount
    parts := strings.Split(s, ":")
    i := 0
    for i < len(parts)-1 {
        hostPath := filepath.Clean(parts[i])
        containerPath := parts[i+1]
        readOnly := false
        if i+2 < len(parts) && (parts[i+2] == "ro" || strings.Contains(parts[i+2], "ro")) {
            readOnly = true
        }

        mounts = append(mounts, Mount{
            HostPath:      hostPath,
            ContainerPath: containerPath,
            ReadOnly:      readOnly,
        })
        i += 2 + boolToInt(readOnly) // Skip 'ro' if present
    }
    return mounts
}

func boolToInt(b bool) int {
    if b {
        return 1
    }
    return 0
}

// GetDockerFlags returns Docker run flags for all configured mounts
func (m *MountManager) GetDockerFlags() []string {
    var flags []string

    // Mount worktree by default
    absWorktreePath, _ := filepath.Abs(m.WorktreeRoot)
    flags = append(flags, fmt.Sprintf("-v %s:/workspace:ro", absWorktreePath))

    // Mount additional volumes
    for _, mount := range m.AdditionalMounts {
        flag := "-v"
        if mount.ReadOnly {
            flag += ":ro"
        }
        flags = append(flags, fmt.Sprintf("%s %s:%s", flag, mount.HostPath, mount.ContainerPath))
    }

    return flags
}

// Validate validates all mount paths exist (for non-worktree mounts)
func (m *MountManager) Validate() error {
    for _, mount := range m.AdditionalMounts {
        if !strings.HasPrefix(mount.HostPath, "/") && !strings.HasPrefix(mount.HostPath, "~") {
            return fmt.Errorf("invalid host path: %s", mount.HostPath)
        }

        // Expand ~ to home directory
        expandedPath := strings.ReplaceAll(mount.HostPath, "~/", os.Getenv("HOME")+"/")
        if _, err := os.Stat(expandedPath); err != nil {
            return fmt.Errorf("host path does not exist: %s", mount.HostPath)
        }
    }

    return nil
}

// AddMount adds a new volume mount to the manager
func (m *MountManager) AddMount(hostPath, containerPath string, readOnly bool) error {
    // Validate host path exists
    expandedPath := strings.ReplaceAll(hostPath, "~/", os.Getenv("HOME")+"/")
    if _, err := os.Stat(expandedPath); err != nil {
        return fmt.Errorf("host path does not exist: %s", hostPath)
    }

    m.AdditionalMounts = append(m.AdditionalMounts, Mount{
        HostPath:      expandedPath,
        ContainerPath: containerPath,
        ReadOnly:      readOnly,
    })

    return nil
}

