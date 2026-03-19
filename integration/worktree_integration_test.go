//go:build integration
// +build integration

// integration/worktree_integration_test.go
package integration

import (
	"os"
	"os/exec"
	"testing"

	"cubicheart.com/munchtoast/nekotree/internal/gitworktree"
)

func TestGitWorktreeIntegration(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Skipping: Docker not available")
	}

	tmpDir, _ := os.MkdirTemp("", "gitworktree_*")
	defer os.RemoveAll(tmpDir)

	wm := gitworktree.NewWorktreeManager(tmpDir)
	err := wm.CreateWorktree("integration-test-branch")
	if err != nil {
		t.Fatalf("CreateWorktree failed: %v", err)
	}

	worktrees, err := wm.ListWorktrees()
	if err != nil {
		t.Fatalf("ListWorktrees failed: %v", err)
	}

	if len(worktrees) == 0 {
		t.Error("expected at least one worktree")
	}
}

func isDockerAvailable() bool {
	cmd := exec.Command("docker", "info")
	err := cmd.Run()
	return err == nil
}
