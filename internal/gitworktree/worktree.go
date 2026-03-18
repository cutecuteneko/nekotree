package gitworktree

import (
    "os/exec"
    "path/filepath"
    "strings"
)

type WorktreeManager struct {
    BasePath string
}

func NewWorktreeManager(basePath string) *WorktreeManager {
    return &WorktreeManager{
        BasePath: basePath,
    }
}

func (wm *WorktreeManager) GetBasePath() string {
    return wm.BasePath
}

func (wm *WorktreeManager) CreateWorktree(branchName string) error {
    // Define where the new worktree will actually sit (e.g., a subfolder named after the branch)
    targetPath := filepath.Join(wm.BasePath, "..", "nekotree-"+branchName)
    absTarget, _ := filepath.Abs(targetPath)

    // Command: git worktree add <path> -b <new-branch>
    cmd := exec.Command("git", "-C", wm.BasePath, "worktree", "add", "-b", branchName, absTarget)
    return runSilent(cmd)
}

func (wm *WorktreeManager) ListWorktrees() ([]string, error) {
    out, err := exec.Command("git", "-C", wm.BasePath, "worktree", "list").Output()
    if err != nil {
        return nil, err
    }

    var worktrees []string
    lines := strings.Split(strings.TrimSpace(string(out)), "\n")
    for _, line := range lines {
        // The first field in 'git worktree list' is always the absolute path
        fields := strings.Fields(line)
        if len(fields) > 0 {
            worktrees = append(worktrees, fields[0])
        }
    }
    return worktrees, nil
}

func (wm *WorktreeManager) RemoveWorktree(path string) error {
    cmd := exec.Command("git", "-C", wm.BasePath, "worktree", "remove", path, "--force")
    return runSilent(cmd)
}

// Helper functions
func splitLines(s string) []string {
    var lines []string
    for _, line := range strings.Split(s, "\n") {
        if trimmed := strings.TrimSpace(line); trimmed != "" {
            lines = append(lines, trimmed)
        }
    }
    return lines
}

func splitFields(line string) []string {
    fields := strings.Fields(line)
    var result []string
    for _, f := range fields {
        if s := strings.Split(f, ":"); len(s) > 0 && s[0] != "." {
            result = append(result, s...)
        } else if f == "." {
            continue // Skip the current worktree indicator
        } else {
            result = append(result, f)
        }
    }
    return result
}

func runSilent(cmd *exec.Cmd) error {
    cmd.Stdout = nil
    cmd.Stderr = nil
    return cmd.Run()
}

