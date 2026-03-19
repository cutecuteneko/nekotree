package gitworktree

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// setupTestRepo creates a real, temporary Git repository for the tests to use.
// This is necessary because 'git worktree' commands require a valid .git context.
func setupTestRepo(t *testing.T) string {
	tempDir := t.TempDir()

	// Initialize the repo
	runGit(t, tempDir, "init")

	// Set local config so it doesn't fail on RHEL environments without global git config
	runGit(t, tempDir, "config", "user.email", "test@example.com")
	runGit(t, tempDir, "config", "user.name", "Test User")

	// Create an initial commit (worktrees require at least one commit to exist)
	dummyFile := filepath.Join(tempDir, "README.md")
	if err := os.WriteFile(dummyFile, []byte("# Test Repo"), 0644); err != nil {
		t.Fatalf("Failed to write dummy file: %v", err)
	}

	runGit(t, tempDir, "add", ".")
	runGit(t, tempDir, "commit", "-m", "initial commit")

	return tempDir
}

// runGit is a helper to execute git commands during test setup.
func runGit(t *testing.T, dir string, args ...string) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\nOutput: %s", args, err, string(out))
	}
}

func TestCreateWorktree(t *testing.T) {
	// 1. Setup a real git repo in a temp folder
	repoDir := setupTestRepo(t)

	// 2. Initialize manager with 'nil' for the production runner
	wm := NewWorktreeManager(repoDir, nil)

	// 3. Test a normal branch creation
	branch := "test-feature"
	err := wm.CreateWorktree(branch)
	if err != nil {
		t.Fatalf("CreateWorktree failed: %v", err)
	}

	// 4. Verify directory was created
	repoName := filepath.Base(repoDir)
	expectedPath := filepath.Join(repoDir, fmt.Sprintf("nekotree-%s-%s", repoName, branch))
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Expected worktree directory at %s, but it was not found", expectedPath)
	}
}

func TestRemoveWorktree(t *testing.T) {
	repoDir := setupTestRepo(t)
	wm := NewWorktreeManager(repoDir, nil)

	branch := "remove-me"
	repoName := filepath.Base(repoDir)
	targetPath := filepath.Join(repoDir, fmt.Sprintf("nekotree-%s-%s", repoName, branch))

	// 1. Create it first
	if err := wm.CreateWorktree(branch); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// 2. Remove it
	err := wm.RemoveWorktree(targetPath)
	if err != nil {
		t.Fatalf("RemoveWorktree failed: %v", err)
	}

	// 3. Verify it's gone
	if _, err := os.Stat(targetPath); !os.IsNotExist(err) {
		t.Errorf("Worktree directory still exists at %s after removal", targetPath)
	}
}
