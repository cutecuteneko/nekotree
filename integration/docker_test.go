//go:build integration
// +build integration

package integration

import (
	"crypto/rand"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	"cubicheart.com/munchtoast/nekotree/internal/config"
	"cubicheart.com/munchtoast/nekotree/internal/docker"
	"cubicheart.com/munchtoast/nekotree/internal/volumes"
)

func TestContainerLifecycle(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Skipping: Docker not available")
	}

	name := "nekotree-test-" + randomID(5)
	cfg := &config.Config{DefaultImage: "alpine"}
	mv := volumes.NewMountManager("/tmp")

	cm := docker.NewContainerManager(name, cfg, mv)

	t.Logf("Starting container: %s", name)
	if err := cm.Start("/tmp"); err != nil {
		t.Fatalf("failed to start container: %v", err)
	}
	defer cm.Stop()

	// Wait for Docker
	time.Sleep(500 * time.Millisecond)

	out, err := exec.Command("docker", "inspect", "--format", "{{.State.Running}}", name).Output()
	if err != nil || !strings.Contains(string(out), "true") {
		t.Fatalf("container is not running: %v (output: %s)", err, string(out))
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
