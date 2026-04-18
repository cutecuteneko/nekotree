package gitworktree

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cubicheart.com/munchtoast/nekotree/internal/docker" // Reusing the interface
	"cubicheart.com/munchtoast/nekotree/internal/utils"
)

type WorktreeManager struct {
	repoRoot string
	runner   docker.CommandRunner // Add the runner here
}

func NewWorktreeManager(repoRoot string, runner docker.CommandRunner) *WorktreeManager {
	absRoot, _ := filepath.Abs(repoRoot)
	if runner == nil {
		runner = &docker.RealRunner{}
	}
	return &WorktreeManager{
		repoRoot: absRoot,
		runner:   runner,
	}
}

func (w *WorktreeManager) CreateWorktree(branch string) error {
	safeBranch, err := utils.Sanitize(branch)
	if err != nil {
		return fmt.Errorf("invalid branch name: %w", err)
	}

	repoName := filepath.Base(w.repoRoot)
	targetPath := filepath.Join(w.repoRoot, fmt.Sprintf("nekotree-%s-%s", repoName, safeBranch))

	safePath, err := utils.SanitizePath(targetPath)
	if err != nil {
		return fmt.Errorf("invalid target path: %w", err)
	}

	if _, err := os.Stat(safePath); err == nil {
		fmt.Printf("ℹ️  Worktree already exists: %s\n", safePath)
		return nil
	}

	// Use w.runner instead of exec.Command
	out, err := w.runner.CombinedOutput("git", "-C", w.repoRoot, "worktree", "add", safePath, "-b", safeBranch)

	if err != nil {
		output := string(out)
		if strings.Contains(output, "already exists") {
			fmt.Printf("ℹ️  Branch '%s' exists, linking...\n", safeBranch)
			_, err2 := w.runner.CombinedOutput("git", "-C", w.repoRoot, "worktree", "add", safePath, safeBranch)
			return err2
		}
		return fmt.Errorf("git error: %v, output: %s", err, output)
	}
	return nil
}

func (w *WorktreeManager) RemoveWorktree(targetPath string) error {
	safePath, err := utils.SanitizePath(targetPath)
	if err != nil {
		return fmt.Errorf("invalid path for removal: %w", err)
	}

	// Use w.runner
	_ = w.runner.Run("git", "-C", w.repoRoot, "worktree", "prune")

	out, err := w.runner.CombinedOutput("git", "-C", w.repoRoot, "worktree", "remove", safePath, "--force")
	if err != nil {
		if strings.Contains(string(out), "not a working tree") {
			return os.RemoveAll(safePath)
		}
		return fmt.Errorf("failed to remove worktree: %s: %w", string(out), err)
	}
	return nil
}

// Exists checks if a worktree directory for this branch exists
func (w *WorktreeManager) Exists(branch string) bool {
	// Assuming target path logic follows your naming convention
	repoName := filepath.Base(w.repoRoot)
	target := filepath.Join(w.repoRoot, fmt.Sprintf("nekotree-%s-%s", repoName, branch))
	info, err := os.Stat(target)
	return err == nil && info.IsDir()
}
