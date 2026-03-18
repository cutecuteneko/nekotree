// integration/worktree_integration_test.go
//go:build integration
// +build integration

package integration

import (
    "os"
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

