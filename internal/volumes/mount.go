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

// GetDockerFlags returns a flat slice of strings ready for exec.Command
func (m *MountManager) GetDockerFlags() []string {
	var flags []string

	// Primary Worktree Mount
	absWorktreePath, _ := filepath.Abs(m.WorktreeRoot)
	// We append two separate strings: the flag and the mapping
	flags = append(flags, "-v", fmt.Sprintf("%s:/workspace:rw", absWorktreePath))

	// Additional volumes
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
		expandedPath := strings.ReplaceAll(mount.HostPath, "~/", os.Getenv("HOME")+"/")
		if _, err := os.Stat(expandedPath); err != nil {
			return fmt.Errorf("host path does not exist: %s", mount.HostPath)
		}
	}
	return nil
}

func parseMountString(s string) []Mount {
	var mounts []Mount
	parts := strings.Split(s, ":")
	for i := 0; i < len(parts)-1; i += 2 {
		mounts = append(mounts, Mount{
			HostPath:      filepath.Clean(parts[i]),
			ContainerPath: parts[i+1],
			ReadOnly:      false,
		})
	}
	return mounts
}
