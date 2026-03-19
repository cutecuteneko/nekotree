package volumes

import (
	"testing"
)

func TestGetDockerFlags(t *testing.T) {
	mm := NewMountManager("/tmp/worktree", Mount{HostPath: "/data", ContainerPath: "/mnt", ReadOnly: true})
	flags := mm.GetDockerFlags()

	// Expected: ["-v", "/tmp/worktree:/workspace:rw", "-v", "/data:/mnt:ro"]
	expectedCount := 4
	if len(flags) != expectedCount {
		t.Fatalf("Expected %d flags, got %d: %v", expectedCount, len(flags), flags)
	}

	if flags[0] != "-v" || flags[2] != "-v" {
		t.Errorf("Flags are not correctly separated: %v", flags)
	}
}
