package volumes

import (
    "os"
    "testing"
)

func TestParseMountString(t *testing.T) {
    tests := []struct {
        input    string
        expected int
        readOnly bool
    }{
        {"host1:container1", 2, false},
        {"host1:container1:host2:container2", 4, false},
        {"host1:container1:ro", 3, true},
    }

    for _, tt := range tests {
        t.Run(tt.input, func(t *testing.T) {
            mounts := parseMountString(tt.input)
            if len(mounts) != tt.expected/2 {
                t.Errorf("expected %d mounts, got %d", tt.expected/2, len(mounts))
            }
        })
    }
}

func TestGetDockerFlags(t *testing.T) {
    m := &MountManager{
        WorktreeRoot:     "/test/worktree",
        AdditionalMounts: []Mount{{HostPath: "/data", ContainerPath: "/data"}},
    }
    flags := m.GetDockerFlags()

    if len(flags) != 2 {
        t.Errorf("expected 2 flags, got %d", len(flags))
    }
}

func TestValidate(t *testing.T) {
    m := &MountManager{AdditionalMounts: []Mount{{HostPath: "invalid/path"}}}
    err := m.Validate()
    if err == nil {
        t.Error("expected validation error for non-existent path")
    }
}

func TestAddMount(t *testing.T) {
    tmpDir, _ := os.MkdirTemp("", "test_mount_*")
    defer os.RemoveAll(tmpDir)

    m := &MountManager{}
    err := m.AddMount(tmpDir, "/container/path", false)
    if err != nil {
        t.Fatalf("AddMount failed: %v", err)
    }
}

