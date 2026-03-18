// internal/gitworktree/worktree_test.go
package gitworktree

import (
    "os"
    "os/exec"
    "path/filepath"
    "testing"
)

func TestCreateWorktree(t *testing.T) {
    // Create a temporary directory and initialize it as a Git repo
    tmpDir, err := os.MkdirTemp("", "gitworktree_*")
    if err != nil {
        t.Fatalf("Failed to create temp dir: %v", err)
    }
    defer os.RemoveAll(tmpDir)

    // Initialize git repo
    cmd := exec.Command("git", "init")
    cmd.Dir = tmpDir
    if err := cmd.Run(); err != nil {
        t.Fatalf("Failed to init git repo: %v", err)
    }

    // Set up git config
    cmd = exec.Command("git", "config", "user.name", "test")
    cmd.Dir = tmpDir
    if err := cmd.Run(); err != nil {
        t.Fatalf("Failed to set git user name: %v", err)
    }

    cmd = exec.Command("git", "config", "user.email", "test@example.com")
    cmd.Dir = tmpDir
    if err := cmd.Run(); err != nil {
        t.Fatalf("Failed to set git user email: %v", err)
    }

    // Create a base commit so worktrees can be created
    testFile := filepath.Join(tmpDir, "README.md")
    if err := os.WriteFile(testFile, []byte("# Test"), 0644); err != nil {
        t.Fatalf("Failed to create test file: %v", err)
    }

    cmd = exec.Command("git", "add", "README.md")
    cmd.Dir = tmpDir
    if err := cmd.Run(); err != nil {
        t.Fatalf("Failed to add file to git: %v", err)
    }

    cmd = exec.Command("git", "commit", "-m", "Initial commit")
    cmd.Dir = tmpDir
    if err := cmd.Run(); err != nil {
        t.Fatalf("Failed to commit: %v", err)
    }

    // Now test the worktree creation
    wm := NewWorktreeManager(tmpDir)
    err = wm.CreateWorktree("test-branch")
    if err != nil {
        t.Fatalf("CreateWorktree failed: %v", err)
    }

    // Verify the worktree was created
    worktrees, err := wm.ListWorktrees()
    if err != nil {
        t.Fatalf("ListWorktrees failed: %v", err)
    }

    if len(worktrees) == 0 {
        t.Error("expected at least one worktree")
    }
}

func TestListWorktrees(t *testing.T) {
    // Create a temporary directory and initialize it as a Git repo
    tmpDir, err := os.MkdirTemp("", "gitworktree_*")
    if err != nil {
        t.Fatalf("Failed to create temp dir: %v", err)
    }
    defer os.RemoveAll(tmpDir)

    // Initialize git repo
    cmd := exec.Command("git", "init")
    cmd.Dir = tmpDir
    if err := cmd.Run(); err != nil {
        t.Fatalf("Failed to init git repo: %v", err)
    }

    // Set up git config
    cmd = exec.Command("git", "config", "user.name", "test")
    cmd.Dir = tmpDir
    if err := cmd.Run(); err != nil {
        t.Fatalf("Failed to set git user name: %v", err)
    }

    cmd = exec.Command("git", "config", "user.email", "test@example.com")
    cmd.Dir = tmpDir
    if err := cmd.Run(); err != nil {
        t.Fatalf("Failed to set git user email: %v", err)
    }

    // Create a base commit so worktrees can be created
    testFile := filepath.Join(tmpDir, "README.md")
    if err := os.WriteFile(testFile, []byte("# Test"), 0644); err != nil {
        t.Fatalf("Failed to create test file: %v", err)
    }

    cmd = exec.Command("git", "add", "README.md")
    cmd.Dir = tmpDir
    if err := cmd.Run(); err != nil {
        t.Fatalf("Failed to add file to git: %v", err)
    }

    cmd = exec.Command("git", "commit", "-m", "Initial commit")
    cmd.Dir = tmpDir
    if err := cmd.Run(); err != nil {
        t.Fatalf("Failed to commit: %v", err)
    }

    // Test listing worktrees (should return just the main repo)
    wm := NewWorktreeManager(tmpDir)
    worktrees, err := wm.ListWorktrees()
    if err != nil {
        t.Fatalf("ListWorktrees failed: %v", err)
    }

    if len(worktrees) == 0 {
        t.Error("expected at least one worktree")
    }
}

