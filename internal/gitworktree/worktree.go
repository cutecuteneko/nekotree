package gitworktree

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cutecuteneko/nekotree/internal/runner" // Shared CommandRunner interface
	"github.com/cutecuteneko/nekotree/internal/utils"
)

// gitErrBranchAlreadyExists is the sentinel git embeds in stderr when `-b
// <branch>` targets a branch that already exists. Git does not use a distinct
// exit code for this case, so string matching is the only reliable approach.
const gitErrBranchAlreadyExists = "already exists"

// gitErrNotAWorkingTree is the sentinel git embeds in stderr when `worktree
// remove` is called on a path that is not a registered worktree.
const gitErrNotAWorkingTree = "not a working tree"

type WorktreeManager struct {
	repoRoot string
	runner   runner.CommandRunner
}

func NewWorktreeManager(repoRoot string, r runner.CommandRunner) *WorktreeManager {
	absRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		// filepath.Abs only fails on systems where os.Getwd() fails; fall back to the raw path
		absRoot = repoRoot
	}
	if r == nil {
		r = &runner.RealRunner{}
	}
	return &WorktreeManager{
		repoRoot: absRoot,
		runner:   r,
	}
}

func (w *WorktreeManager) CreateWorktree(branch string) error {
	safeBranch, err := utils.Sanitize(branch)
	if err != nil {
		return fmt.Errorf("invalid branch name: %w", err)
	}

	repoName := filepath.Base(w.repoRoot)
	targetPath := filepath.Join(w.repoRoot, utils.BuildName(repoName, safeBranch))

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
		if strings.Contains(output, gitErrBranchAlreadyExists) {
			fmt.Printf("ℹ️  Branch '%s' exists, linking...\n", safeBranch)
			_, err2 := w.runner.CombinedOutput("git", "-C", w.repoRoot, "worktree", "add", safePath, safeBranch)
			return err2
		}
		return fmt.Errorf("git worktree add failed: %s: %w", output, err)
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
		if strings.Contains(string(out), gitErrNotAWorkingTree) {
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
	target := filepath.Join(w.repoRoot, utils.BuildName(repoName, branch))
	info, err := os.Stat(target)
	return err == nil && info.IsDir()
}
