package gitworktree

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/cutecuteneko/nekotree/internal/testutil"
)

// setupTestRepo creates a real, temporary Git repository for the tests to use.
// This is necessary because 'git worktree' commands require a valid .git context.
func setupTestRepo(t *testing.T) string {
	t.Helper()
	tempDir := t.TempDir()

	runGit(t, tempDir, "init")
	runGit(t, tempDir, "config", "user.email", "test@example.com")
	runGit(t, tempDir, "config", "user.name", "Test User")

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
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\nOutput: %s", args, err, string(out))
	}
}

// mockGitRunner is a package-local alias for the shared mock implementation.
type mockGitRunner = testutil.MockRunner

// --- CreateWorktree (real git) ---

func TestCreateWorktree(t *testing.T) {
	repoDir := setupTestRepo(t)
	wm := NewWorktreeManager(repoDir, nil)

	branch := "test-feature"
	if err := wm.CreateWorktree(branch); err != nil {
		t.Fatalf("CreateWorktree failed: %v", err)
	}

	repoName := filepath.Base(repoDir)
	expectedPath := filepath.Join(repoDir, fmt.Sprintf("nekotree-%s-%s", repoName, branch))
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Expected worktree directory at %s, but it was not found", expectedPath)
	}
}

func TestCreateWorktree_AlreadyExists(t *testing.T) {
	repoDir := setupTestRepo(t)
	wm := NewWorktreeManager(repoDir, nil)

	branch := "already-exists"
	// Create it once
	if err := wm.CreateWorktree(branch); err != nil {
		t.Fatalf("First create failed: %v", err)
	}
	// Create again — should be a no-op, not an error
	if err := wm.CreateWorktree(branch); err != nil {
		t.Errorf("Second create should be a no-op, got error: %v", err)
	}
}

func TestCreateWorktree_InvalidBranchName(t *testing.T) {
	repoDir := setupTestRepo(t)
	wm := NewWorktreeManager(repoDir, nil)

	err := wm.CreateWorktree("bad branch; rm -rf /")
	if err == nil {
		t.Error("expected error for invalid branch name")
	}
}

func TestCreateWorktree_BranchExistsInGit(t *testing.T) {
	// When the git branch already exists (but worktree doesn't), git returns
	// "already exists" and CreateWorktree falls back to linking the existing branch.
	repoDir := setupTestRepo(t)

	// Pre-create the branch in git without a worktree
	runGit(t, repoDir, "branch", "existing-branch")

	wm := NewWorktreeManager(repoDir, nil)
	err := wm.CreateWorktree("existing-branch")
	if err != nil {
		t.Errorf("expected no error when linking existing branch, got: %v", err)
	}
}

// --- RemoveWorktree (real git) ---

func TestRemoveWorktree(t *testing.T) {
	repoDir := setupTestRepo(t)
	wm := NewWorktreeManager(repoDir, nil)

	branch := "remove-me"
	repoName := filepath.Base(repoDir)
	targetPath := filepath.Join(repoDir, fmt.Sprintf("nekotree-%s-%s", repoName, branch))

	if err := wm.CreateWorktree(branch); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	if err := wm.RemoveWorktree(targetPath); err != nil {
		t.Fatalf("RemoveWorktree failed: %v", err)
	}

	if _, err := os.Stat(targetPath); !os.IsNotExist(err) {
		t.Errorf("Worktree directory still exists at %s after removal", targetPath)
	}
}

func TestRemoveWorktree_InvalidPath(t *testing.T) {
	repoDir := setupTestRepo(t)
	wm := NewWorktreeManager(repoDir, nil)

	err := wm.RemoveWorktree("../../etc/passwd")
	if err == nil {
		t.Error("expected error for directory traversal path")
	}
}

func TestRemoveWorktree_NotAWorkingTree(t *testing.T) {
	// When git says "not a working tree", RemoveWorktree falls back to os.RemoveAll.
	// Use a mock to simulate that git error and a real temp dir to remove.
	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "nekotree-repo-branch")
	if err := os.MkdirAll(subDir, 0750); err != nil {
		t.Fatalf("failed to create test dir: %v", err)
	}

	mock := &mockGitRunner{
		Output: []byte("not a working tree"),
		Err:    fmt.Errorf("exit status 128"),
	}
	wm := &WorktreeManager{repoRoot: tempDir, runner: mock}

	err := wm.RemoveWorktree(subDir)
	// os.RemoveAll succeeds; function should return nil
	if err != nil {
		t.Errorf("expected fallback RemoveAll to succeed, got: %v", err)
	}
	if _, statErr := os.Stat(subDir); !os.IsNotExist(statErr) {
		t.Error("expected directory to be removed by os.RemoveAll fallback")
	}
}

func TestRemoveWorktree_RunnerError(t *testing.T) {
	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "nekotree-repo-branch")
	if err := os.MkdirAll(subDir, 0750); err != nil {
		t.Fatalf("failed to create test dir: %v", err)
	}

	mock := &mockGitRunner{
		Output: []byte("some other git error"),
		Err:    fmt.Errorf("exit status 1"),
	}
	wm := &WorktreeManager{repoRoot: tempDir, runner: mock}

	err := wm.RemoveWorktree(subDir)
	if err == nil {
		t.Error("expected error when git worktree remove fails with non-sentinel output")
	}
}

// --- Exists ---

func TestExists_WorktreePresent(t *testing.T) {
	repoDir := setupTestRepo(t)
	wm := NewWorktreeManager(repoDir, nil)

	branch := "check-exists"
	if err := wm.CreateWorktree(branch); err != nil {
		t.Fatalf("CreateWorktree failed: %v", err)
	}

	if !wm.Exists(branch) {
		t.Error("expected Exists() to return true after creating worktree")
	}
}

func TestExists_WorktreeAbsent(t *testing.T) {
	repoDir := setupTestRepo(t)
	wm := NewWorktreeManager(repoDir, nil)

	if wm.Exists("no-such-branch") {
		t.Error("expected Exists() to return false for non-existent worktree")
	}
}

func TestExists_AfterRemoval(t *testing.T) {
	repoDir := setupTestRepo(t)
	wm := NewWorktreeManager(repoDir, nil)

	branch := "remove-check"
	repoName := filepath.Base(repoDir)
	targetPath := filepath.Join(repoDir, fmt.Sprintf("nekotree-%s-%s", repoName, branch))

	if err := wm.CreateWorktree(branch); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	if err := wm.RemoveWorktree(targetPath); err != nil {
		t.Fatalf("RemoveWorktree failed: %v", err)
	}

	if wm.Exists(branch) {
		t.Error("expected Exists() to return false after removal")
	}
}
