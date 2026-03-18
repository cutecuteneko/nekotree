package docker

import (
	"strings"
	"testing"
	"cubicheart.com/munchtoast/nekotree/internal/volumes"
)

// MockCommander records commands instead of running them
type MockCommander struct {
	LastCommand string
	LastArgs    []string
	OutBytes    []byte
	Err         error
}

func (m *MockCommander) Run(name string, arg ...string) error {
	m.LastCommand = name
	m.LastArgs = arg
	return m.Err
}

func (m *MockCommander) Output(name string, arg ...string) ([]byte, error) {
	m.LastCommand = name
	m.LastArgs = arg
	return m.OutBytes, m.Err
}

func TestStartWithVolumes(t *testing.T) {
	mock := &MockCommander{}
	
	// Define a custom mount to use the volumes package
	customMount := volumes.Mount{
		HostPath:      "/tmp", 
		ContainerPath: "/mnt/data", 
		ReadOnly:      true,
	}
	
	mgr := NewContainerManager("test-vol-app", "alpine:latest", customMount)
	mgr.Exec = mock

	// Start with a dummy worktree root
	err := mgr.Start("/tmp") 
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	argString := strings.Join(mock.LastArgs, " ")

	// 1. Check if the default worktree mount is present
	if !strings.Contains(argString, "/tmp:/workspace:ro") {
		t.Error("Missing default worktree mount in docker args")
	}

	// 2. Check if the custom volume mount is present
	// The volumes package generates "-v:ro /host:/container" or similar based on your GetDockerFlags logic
	if !strings.Contains(argString, "/tmp:/mnt/data") {
		t.Errorf("Custom mount not found in args. Got: %s", argString)
	}
}

func TestStatusParsing(t *testing.T) {
	mock := &MockCommander{
		OutBytes: []byte("running\n"),
	}
	mgr := NewContainerManager("status-check", "alpine")
	mgr.Exec = mock

	err := mgr.Status()
	if err != nil {
		t.Errorf("Status() failed unexpectedly: %v", err)
	}

	if mock.LastArgs[0] != "inspect" {
		t.Errorf("Expected 'inspect' arg, got %s", mock.LastArgs[0])
	}
}
