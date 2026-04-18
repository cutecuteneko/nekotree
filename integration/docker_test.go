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

	t.Run("ComposeWorkflow", func(t *testing.T) {
		tmpDir := t.TempDir()
		composePath := filepath.Join(tmpDir, "docker-compose.yaml")
		name := "nekotree-compose-" + randomID(5)

		composeContent := fmt.Sprintf(`
services:
  test-app:
    image: alpine
    container_name: %s
    command: ["/bin/sh", "-c", "sleep 3000"]
`, name)

		_ = os.WriteFile(composePath, []byte(composeContent), 0644)

		cfg := &config.Config{ComposeFile: composePath}
		cm := docker.NewContainerManager(name, cfg, nil)

		// FIX: Pass 4 arguments. For Compose, imageName, flags, and command are nil/empty.
		if err := cm.Start(tmpDir, "", nil, nil); err != nil {
			t.Fatalf("failed to start compose: %v", err)
		}
		defer cm.Stop()

		verifyRunning(t, name)
	})

	t.Run("StandaloneImageWorkflow", func(t *testing.T) {
		tmpDir := t.TempDir()
		name := "nekotree-standalone-" + randomID(5)
		image := "alpine:latest"

		flags := []string{"-v", "/tmp:/tmp"}
		command := []string{"sleep", "3000"}

		cfg := &config.Config{}
		cm := docker.NewContainerManager(name, cfg, nil)

		if err := cm.Start(tmpDir, image, flags, command); err != nil {
			t.Fatalf("failed to start standalone image: %v", err)
		}
		defer cm.Stop()

		verifyRunning(t, name)
	})
}

// verifyRunning is a helper to check if a container is actually up
func verifyRunning(t *testing.T, containerName string) {
	t.Helper()

	// Give the engine a moment to transition
	time.Sleep(2 * time.Second)

	out, err := exec.Command("docker", "inspect", "--format", "{{.State.Running}}", containerName).CombinedOutput()
	if err != nil {
		t.Fatalf("docker inspect failed for %s: %v (output: %s)", containerName, err, string(out))
	}

	if !strings.Contains(string(out), "true") {
		t.Fatalf("container %s is not running (output: %s)", containerName, string(out))
	}
}

// Helpers
func isDockerAvailable() bool {
	return exec.Command("docker", "info").Run() == nil
}

func randomID(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}
