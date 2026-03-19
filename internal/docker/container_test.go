package docker

import (
	"testing"
)

// MockCommander for testing without calling real Docker
type MockCommander struct {
	LastCmd  string
	LastArgs []string
}

func (m *MockCommander) Run(name string, arg ...string) error {
	m.LastCmd = name
	m.LastArgs = arg
	return nil
}

func TestNewContainerManager(t *testing.T) {
	name := "nekotree-test"
	image := "alpine"
	compose := "" // New third argument

	mgr := NewContainerManager(name, image, compose)

	if mgr.Name != name {
		t.Errorf("Expected name %s, got %s", name, mgr.Name)
	}
}

func TestContainerStartFlags(t *testing.T) {
	mock := &MockCommander{}
	mgr := NewContainerManager("nekotree-feat", "alpine", "")
	mgr.Exec = mock

	err := mgr.Start("/tmp/worktree")
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Verify it uses detached mode as requested
	foundDetached := false
	for _, arg := range mock.LastArgs {
		if arg == "-d" {
			foundDetached = true
			break
		}
	}

	if !foundDetached {
		t.Error("Expected -d flag in docker run command")
	}
}
