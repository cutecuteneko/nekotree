package gitworktree

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type WorktreeManager struct {
	RepoRoot string
}

func NewWorktreeManager(repoRoot string) *WorktreeManager {
	absRoot, _ := filepath.Abs(repoRoot)
	return &WorktreeManager{RepoRoot: absRoot}
}

func (w *WorktreeManager) CreateWorktree(branch string) error {
	// FIX: Ensure targetPath is explicitly inside the RepoRoot
	targetPath := filepath.Join(w.RepoRoot, "nekotree-"+branch)

	// IDEMPOTENCY CHECK: If the directory already exists, skip git worktree add
	if _, err := os.Stat(targetPath); err == nil {
		fmt.Printf("ℹ️  Worktree directory already exists at %s, skipping creation.\n", targetPath)
		return nil
	}

	// git worktree add <path> -b <branch>
	cmd := exec.Command("git", "worktree", "add", targetPath, "-b", branch)
	cmd.Dir = w.RepoRoot

	if out, err := cmd.CombinedOutput(); err != nil {
		output := string(out)
		// If the branch already exists, try adding the worktree without the -b flag
		if strings.Contains(output, "already exists") || strings.Contains(output, "already checked out") {
			fmt.Printf("ℹ️  Branch '%s' already exists, linking to existing branch.\n", branch)
			cmd = exec.Command("git", "worktree", "add", targetPath, branch)
			cmd.Dir = w.RepoRoot
			if out2, err2 := cmd.CombinedOutput(); err2 != nil {
				return fmt.Errorf("failed to link existing branch: %v, output: %s", err2, string(out2))
			}
			return nil
		}
		return fmt.Errorf("git worktree add failed: %v, output: %s", err, output)
	}

	return nil
}

func (w *WorktreeManager) ListWorktree

func (w *WorktreeManager) RemoveWorktree(targetPath string) error {
	// Ensure metadata is clean
	exec.Command("git", "-C", w.RepoRoot, "worktree", "prune").Run()

	cmd := exec.Command("git", "-C", w.RepoRoot, "worktree", "remove", targetPath, "--force")
	if out, err := cmd.CombinedOutput(); err != nil {
		if strings.Contains(string(out), "not a working tree") {
			return os.RemoveAll(targetPath) // Manual fallback
		}
		return err
	}
	return nil
}
