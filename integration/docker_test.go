//go:build integration
// +build integration

// integration/docker_test.go
package integration

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"cubicheart.com/munchtoast/nekotree/internal/volumes"
)

func TestContainerLifecycle(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Skipping: Docker not available")
	}

	name := "nekotree-test-" + generateRandomName(8)
	defer cleanup(name)

	m := &volumes.MountManager{WorktreeRoot: "/workspace"}
	flags := m.GetDockerFlags()

	// Use a real image for testing
	cmdStr := `docker run --name %s -d --rm alpine sleep 30`
	_, err := execCommand(cmdStr, name)
	if err != nil {
		t.Fatalf("failed to start container: %v", err)
	}

	time.Sleep(2 * time.Second)

	cmdStr = `docker inspect --format '{{.State.Running}}' %s`
	out, err := execCommand(cmdStr, name)
	if err != nil || !contains(out, "true") {
		t.Fatalf("container is not running: %v", err)
	}

	_, stopErr := execCommand(`docker stop %s`, name)
	if stopErr != nil {
		t.Logf("stop error (expected): %v", stopErr)
	}
}

func TestContainerExec(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Skipping: Docker not available")
	}

	name := "nekotree-exec-" + generateRandomName(8)
	defer cleanup(name)

	_, _ = execCommand(`docker run -d --name %s alpine sleep 30`, name)
	time.Sleep(2 * time.Second)

	out, err := execCommand(`docker exec %s echo "hello"`, name)
	if err != nil {
		t.Fatalf("exec failed: %v", err)
	}
	if !contains(out, "hello") {
		t.Error("unexpected output from container exec")
	}
}

func TestContainerStatus(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Skipping: Docker not available")
	}

	name := "nekotree-status-" + generateRandomName(8)
	defer cleanup(name)

	_, _ = execCommand(`docker run -d --name %s alpine sleep 30`, name)
	time.Sleep(2 * time.Second)

	out, err := execCommand(`docker inspect --format '{{.State.Status}}' %s`, name)
	if err != nil {
		t.Fatalf("inspect failed: %v", err)
	}
	if !contains(out, "running") {
		t.Errorf("expected container running, got status: %s", out)
	}
}

func isDockerAvailable() bool {
	cmd := exec.Command("docker", "info")
	err := cmd.Run()
	return err == nil
}

func cleanup(name string) {
	_, err = execCommand(`docker rm -f %s`, name)
	time.Sleep(500 * time.Millisecond) // Allow Docker to clean up
}

func generateRandomName(n int) string {
	chars := "abcdefghijklmnopqrstuvwxyz"
	result := make([]byte, n)
	for i := range result {
		result[i] = chars[os.Getpid()%len(chars)]
	}
	return string(result)
}

func execCommand(cmdStr string, args ...string) (string, error) {
	// Properly construct the command
	cmd := exec.Command("sh", "-c", "docker "+cmdStr)
	for _, arg := range args {
		cmdStr = strings.Replace(cmdStr, "%s", arg, 1)
	}

	output, err := cmd.Output()
	return string(output), err
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
