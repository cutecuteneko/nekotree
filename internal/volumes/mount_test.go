package volumes

import (
	"os"
	"path/filepath"
	"testing"
)

// --- NewMountManager ---

func TestNewMountManager_NoAdditionalMounts(t *testing.T) {
	mm := NewMountManager("/tmp/worktree")
	if mm.WorktreeRoot != "/tmp/worktree" {
		t.Errorf("unexpected WorktreeRoot: %s", mm.WorktreeRoot)
	}
	if len(mm.AdditionalMounts) != 0 {
		t.Errorf("expected no additional mounts, got %d", len(mm.AdditionalMounts))
	}
}

func TestNewMountManager_WithAdditionalMounts(t *testing.T) {
	mm := NewMountManager("/tmp/worktree",
		Mount{HostPath: "/data", ContainerPath: "/mnt", ReadOnly: true},
	)
	if len(mm.AdditionalMounts) != 1 {
		t.Fatalf("expected 1 additional mount, got %d", len(mm.AdditionalMounts))
	}
	if mm.AdditionalMounts[0].HostPath != "/data" {
		t.Errorf("unexpected HostPath: %s", mm.AdditionalMounts[0].HostPath)
	}
}

// --- GetDockerFlags ---

func TestGetDockerFlags_WorktreeOnly(t *testing.T) {
	mm := NewMountManager("/tmp/worktree")
	flags := mm.GetDockerFlags()

	if len(flags) != 2 {
		t.Fatalf("expected 2 flags (one -v pair), got %d: %v", len(flags), flags)
	}
	if flags[0] != "-v" {
		t.Errorf("expected '-v', got %s", flags[0])
	}
	absPath, _ := filepath.Abs("/tmp/worktree")
	expected := absPath + ":/workspace:rw"
	if flags[1] != expected {
		t.Errorf("expected %s, got %s", expected, flags[1])
	}
}

func TestGetDockerFlags_WithReadOnlyMount(t *testing.T) {
	mm := NewMountManager("/tmp/worktree",
		Mount{HostPath: "/data", ContainerPath: "/mnt", ReadOnly: true},
	)
	flags := mm.GetDockerFlags()

	// 2 for worktree + 2 for additional mount
	if len(flags) != 4 {
		t.Fatalf("expected 4 flags, got %d: %v", len(flags), flags)
	}
	if flags[2] != "-v" {
		t.Errorf("expected second -v flag, got %s", flags[2])
	}
	if flags[3] != "/data:/mnt:ro" {
		t.Errorf("expected /data:/mnt:ro, got %s", flags[3])
	}
}

func TestGetDockerFlags_WithReadWriteMount(t *testing.T) {
	mm := NewMountManager("/tmp/worktree",
		Mount{HostPath: "/src", ContainerPath: "/app", ReadOnly: false},
	)
	flags := mm.GetDockerFlags()

	if flags[3] != "/src:/app" {
		t.Errorf("expected /src:/app (no :ro), got %s", flags[3])
	}
}

// --- Validate ---

func TestValidate_ValidPath(t *testing.T) {
	dir := t.TempDir()
	mm := NewMountManager("/tmp/worktree",
		Mount{HostPath: dir, ContainerPath: "/mnt"},
	)
	if err := mm.Validate(); err != nil {
		t.Errorf("expected no error for valid path, got: %v", err)
	}
}

func TestValidate_NonexistentPath(t *testing.T) {
	mm := NewMountManager("/tmp/worktree",
		Mount{HostPath: "/does/not/exist", ContainerPath: "/mnt"},
	)
	if err := mm.Validate(); err == nil {
		t.Error("expected error for nonexistent host path")
	}
}

func TestValidate_NoMounts(t *testing.T) {
	mm := NewMountManager("/tmp/worktree")
	if err := mm.Validate(); err != nil {
		t.Errorf("expected no error with no additional mounts, got: %v", err)
	}
}

// --- LoadFromEnv ---

func TestLoadFromEnv_EnvNotSet(t *testing.T) {
	os.Unsetenv("DEVENV_MOUNTS")
	mm := NewMountManager("/tmp/worktree")
	if err := mm.LoadFromEnv(); err != nil {
		t.Errorf("expected no error when env not set, got: %v", err)
	}
	if len(mm.AdditionalMounts) != 0 {
		t.Errorf("expected no mounts, got: %v", mm.AdditionalMounts)
	}
}

func TestLoadFromEnv_SingleMount(t *testing.T) {
	t.Setenv("DEVENV_MOUNTS", "/src:/workspace")
	mm := NewMountManager("/tmp/worktree")
	if err := mm.LoadFromEnv(); err != nil {
		t.Fatalf("LoadFromEnv failed: %v", err)
	}
	if len(mm.AdditionalMounts) != 1 {
		t.Fatalf("expected 1 mount, got %d", len(mm.AdditionalMounts))
	}
	if mm.AdditionalMounts[0].HostPath != filepath.Clean("/src") {
		t.Errorf("unexpected HostPath: %s", mm.AdditionalMounts[0].HostPath)
	}
	if mm.AdditionalMounts[0].ContainerPath != "/workspace" {
		t.Errorf("unexpected ContainerPath: %s", mm.AdditionalMounts[0].ContainerPath)
	}
	if mm.AdditionalMounts[0].ReadOnly {
		t.Error("expected ReadOnly=false for host:container spec")
	}
}

func TestLoadFromEnv_MultipleMounts(t *testing.T) {
	t.Setenv("DEVENV_MOUNTS", "/src:/workspace,/data:/data:ro")
	mm := NewMountManager("/tmp/worktree")
	if err := mm.LoadFromEnv(); err != nil {
		t.Fatalf("LoadFromEnv failed: %v", err)
	}
	if len(mm.AdditionalMounts) != 2 {
		t.Fatalf("expected 2 mounts, got %d", len(mm.AdditionalMounts))
	}
	if !mm.AdditionalMounts[1].ReadOnly {
		t.Error("expected second mount to be read-only")
	}
}

func TestLoadFromEnv_AppendsToExisting(t *testing.T) {
	t.Setenv("DEVENV_MOUNTS", "/extra:/extra")
	mm := NewMountManager("/tmp/worktree",
		Mount{HostPath: "/pre", ContainerPath: "/pre"},
	)
	if err := mm.LoadFromEnv(); err != nil {
		t.Fatalf("LoadFromEnv failed: %v", err)
	}
	if len(mm.AdditionalMounts) != 2 {
		t.Fatalf("expected 2 mounts after append, got %d", len(mm.AdditionalMounts))
	}
}

// --- parseMountString ---

func TestParseMountString_HostContainer(t *testing.T) {
	mounts := parseMountString("/src:/workspace")
	if len(mounts) != 1 {
		t.Fatalf("expected 1 mount, got %d", len(mounts))
	}
	if mounts[0].HostPath != filepath.Clean("/src") || mounts[0].ContainerPath != "/workspace" {
		t.Errorf("unexpected mount: %+v", mounts[0])
	}
	if mounts[0].ReadOnly {
		t.Error("expected ReadOnly=false")
	}
}

func TestParseMountString_ReadOnly(t *testing.T) {
	mounts := parseMountString("/data:/data:ro")
	if len(mounts) != 1 {
		t.Fatalf("expected 1 mount, got %d", len(mounts))
	}
	if !mounts[0].ReadOnly {
		t.Error("expected ReadOnly=true for :ro spec")
	}
}

func TestParseMountString_CommaSeparated(t *testing.T) {
	mounts := parseMountString("/a:/a,/b:/b:ro,/c:/c")
	if len(mounts) != 3 {
		t.Fatalf("expected 3 mounts, got %d: %v", len(mounts), mounts)
	}
	if !mounts[1].ReadOnly {
		t.Error("expected second mount to be read-only")
	}
}

func TestParseMountString_EmptyString(t *testing.T) {
	mounts := parseMountString("")
	if len(mounts) != 0 {
		t.Errorf("expected no mounts for empty string, got: %v", mounts)
	}
}

func TestParseMountString_MalformedEntry(t *testing.T) {
	// An entry with no colon should be skipped; valid entry should still parse
	mounts := parseMountString("nocolon,/valid:/valid")
	if len(mounts) != 1 {
		t.Errorf("expected 1 valid mount after skipping malformed entry, got %d: %v", len(mounts), mounts)
	}
}

func TestParseMountString_WhitespaceAround(t *testing.T) {
	mounts := parseMountString("  /src:/workspace  ,  /data:/data  ")
	if len(mounts) != 2 {
		t.Fatalf("expected 2 mounts with trimmed whitespace, got %d", len(mounts))
	}
}
