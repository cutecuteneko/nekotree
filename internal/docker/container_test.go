package docker

import (
	"fmt"
	"strings"
	"testing"

	"cubicheart.com/munchtoast/nekotree/internal/config"
)

// MockRunner now records a history of calls
type MockRunner struct {
	Calls []string
}

func (m *MockRunner) Run(name string, arg ...string) error {
	m.Calls = append(m.Calls, fmt.Sprintf("%s %s", name, strings.Join(arg, " ")))
	return nil
}

func (m *MockRunner) CombinedOutput(name string, arg ...string) ([]byte, error) {
	m.Calls = append(m.Calls, fmt.Sprintf("%s %s", name, strings.Join(arg, " ")))
	// Return a "No such container" style message if needed for logic testing
	return []byte("mock output"), nil
}

func TestContainerStop(t *testing.T) {
	mock := &MockRunner{}
	cfg := &config.Config{}
	cm := NewContainerManager("test-env", cfg, mock)

	err := cm.Stop()
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	// We expect at least 'docker stop' and 'docker rm'
	foundStop := false
	foundRm := false

	for _, cmd := range mock.Calls {
		if strings.Contains(cmd, "docker stop test-env") {
			foundStop = true
		}
		if strings.Contains(cmd, "docker rm -v test-env") {
			foundRm = true
		}
	}

	if !foundStop {
		t.Errorf("Expected 'docker stop' to be called, but wasn't. Calls: %v", mock.Calls)
	}
	if !foundRm {
		t.Errorf("Expected 'docker rm -v' to be called, but wasn't. Calls: %v", mock.Calls)
	}
}
