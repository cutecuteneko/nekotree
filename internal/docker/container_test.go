package docker

import (
	"testing"

	"cubicheart.com/munchtoast/nekotree/internal/config"
)

type MockRunner struct {
	LastCmd  string
	LastArgs []string
}

func (m *MockRunner) Run(name string, arg ...string) error {
	m.LastCmd = name
	m.LastArgs = arg
	return nil
}

func (m *MockRunner) CombinedOutput(name string, arg ...string) ([]byte, error) {
	m.LastCmd = name
	m.LastArgs = arg
	return []byte("mock output"), nil
}

func TestContainerStop(t *testing.T) {
	mock := &MockRunner{}
	cfg := &config.Config{}
	cm := NewContainerManager("test-env", cfg, mock)

	err := cm.Stop()
	if err != nil {
		t.Errorf("Stop failed: %v", err)
	}

	// Verify the last command called was 'docker rm'
	if mock.LastCmd != "docker" || mock.LastArgs[0] != "rm" {
		t.Errorf("Expected docker rm, got %s %v", mock.LastCmd, mock.LastArgs)
	}
}
