package gitworktree

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// setupTestRepo creates a temporary directory, initializes a git repo,
// and adds an initial commit so worktrees have a HEAD to branch from.
func setupTestRepo(t *testing.T) string {
	tempDir, err := os.MkdirTemp("", "nekotree-git-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Initialize Git
	runGit(t, tempDir, "init")
	runGit(t, tempDir, "config", "user.email", "test@example.com")
	runGit(t, tempDir, "config", "user.name", "Test User")

	// Create initial commit
	dummyFile := filepath.Join(tempDir, "README.md")
	if err := os.WriteFile(dummyFile, []byte("# Test Repo"), 0644); err != nil {
		t.Fatalf("Failed to write dummy file: %v", err)
	}

	runGit(t, tempDir, "add", ".")
	runGit(t, tempDir, "commit", "-m", "initial commit")

	return tempDir
}

// runGit is a helper to execute git commands within a specific directory
func runGit(t *testing.T, dir string, args ...string) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\nOutput: %s", args, err, string(out))
	}
}

func TestCreateWorktree(t *testing.T) {
	repoDir := setupTestRepo(t)
	defer os.RemoveAll(repoDir)

	wm := NewWorktreeManager(repoDir)
	branch := "test-feature"

	// 1. Test Initial Creation
	err := wm.CreateWorktree(branch)
	if err != nil {
		t.Fatalf("CreateWorktree failed: %v", err)
	}

	// The folder should be INSIDE the repoDir (repoDir/nekotree-test-feature)
	expectedPath := filepath.Join(repoDir, "nekotree-"+branch)
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Worktree directory was not created at %s", expectedPath)
	}

	// 2. Test Idempotency (Running it again should not fail)
	err = wm.CreateWorktree(branch)
	if err != nil {
		t.Errorf("CreateWorktree failed on second run (idempotency check): %v", err)
	}
}

func TestRemoveWorktree(t *testing.T) {
	repoDir := setupTestRepo(t)
	defer os.RemoveAll(repoDir)

	wm := NewWorktreeManager(repoDir)
	branch := "remove-me"
	targetPath := filepath.Join(repoDir, "nekotree-"+branch)

	// Setup: Create the worktree first
	if err := wm.CreateWorktree(branch); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// Test Removal
	err := wm.RemoveWorktree(targetPath)
	if err != nil {
		t.Fatalf("RemoveWorktree failed: %v", err)
	}

	// Verify folder is gone
	if _, err := os.Stat(targetPath); !os.IsNotExist(err) {
		t.Errorf("Worktree directory still exists at %s after removal", targetPath)
	}
}
