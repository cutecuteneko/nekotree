package volumes

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Mount struct {
	HostPath      string
	ContainerPath string
	ReadOnly      bool
}

type MountManager struct {
	WorktreeRoot     string
	AdditionalMounts []Mount
}

func NewMountManager(worktreeRoot string, additionalMounts ...Mount) *MountManager {
	return &MountManager{
		WorktreeRoot:     worktreeRoot,
		AdditionalMounts: additionalMounts,
	}
}

func (m *MountManager) LoadFromEnv() error {
	if mountsStr := os.Getenv("DEVENV_MOUNTS"); mountsStr != "" {
		m.AdditionalMounts = append(m.AdditionalMounts, parseMountString(mountsStr)...)
	}
	return nil
}

func (m *MountManager) GetDockerFlags() []string {
	var flags []string
	absWorktreePath, _ := filepath.Abs(m.WorktreeRoot)

	// Map the worktree to /workspace by default
	flags = append(flags, "-v", fmt.Sprintf("%s:/workspace:rw", absWorktreePath))

	for _, mount := range m.AdditionalMounts {
		mapping := fmt.Sprintf("%s:%s", mount.HostPath, mount.ContainerPath)
		if mount.ReadOnly {
			mapping += ":ro"
		}
		flags = append(flags, "-v", mapping)
	}
	return flags
}

func (m *MountManager) Validate() error {
	for _, mount := range m.AdditionalMounts {
		if _, err := os.Stat(mount.HostPath); err != nil {
			return fmt.Errorf("invalid host path: %s", mount.HostPath)
		}
	}
	return nil
}

// parseMountString parses a comma-separated list of Docker volume specs.
// Each entry is host:container or host:container:ro.
// Example: DEVENV_MOUNTS=/src:/workspace,/data:/data:ro
func parseMountString(s string) []Mount {
	var mounts []Mount
	for _, entry := range strings.Split(s, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		parts := strings.SplitN(entry, ":", 3)
		if len(parts) < 2 {
			continue
		}
		m := Mount{
			HostPath:      filepath.Clean(parts[0]),
			ContainerPath: parts[1],
		}
		if len(parts) == 3 && parts[2] == "ro" {
			m.ReadOnly = true
		}
		mounts = append(mounts, m)
	}
	return mounts
}
