//go:build integration
// +build integration

package integration

import (
	"crypto/rand"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cubicheart.com/munchtoast/nekotree/internal/config"
	"cubicheart.com/munchtoast/nekotree/internal/docker"
)

func TestContainerLifecycle(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Skipping: Docker not available")
	}

	tmpDir := t.TempDir()
	composePath := filepath.Join(tmpDir, "docker-compose.yml")

	// FIX: Explicitly set the container_name so 'docker inspect' knows exactly what to look for
	name := "nekotree-test-" + randomID(5)
	composeContent := fmt.Sprintf(`
services:
  test-app:
    image: alpine
    container_name: %s
    command: ["/bin/sh", "-c", "sleep 3000"]
`, name)

	if err := os.WriteFile(composePath, []byte(composeContent), 0644); err != nil {
		t.Fatalf("failed to write tmp compose file: %v", err)
	}

	cfg := &config.Config{ComposeFile: composePath}
	cm := docker.NewContainerManager(name, cfg, nil)

	t.Logf("Starting container: %s", name)
	if err := cm.Start(tmpDir); err != nil {
		t.Fatalf("failed to start container: %v", err)
	}

	// Ensure we clean up even if the test fails
	defer cm.Stop()

	// Give the engine a moment to transition to 'running'
	time.Sleep(2 * time.Second)

	// Check status
	out, err := exec.Command("docker", "inspect", "--format", "{{.State.Running}}", name).CombinedOutput()
	if err != nil {
		t.Fatalf("docker inspect failed: %v (output: %s)", err, string(out))
	}

	if !strings.Contains(string(out), "true") {
		t.Fatalf("container is not running (output: %s)", string(out))
	}
}

// Helpers
func isDockerAvailable() bool {
	return exec.Command("docker", "info").Run() == nil
}

func randomID(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}
