//go:build integration
// +build integration

package integration

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"cubicheart.com/munchtoast/nekotree/internal/gitworktree"
)

func TestGitWorktreeIntegration(t *testing.T) {
	// 1. Setup a real temp git repo for integration
	tmpDir, _ := os.MkdirTemp("", "nekotree_int_repo_*")
	defer os.RemoveAll(tmpDir)

	runCmd(t, tmpDir, "git", "init")
	runCmd(t, tmpDir, "git", "config", "user.email", "int@test.com")
	runCmd(t, tmpDir, "git", "config", "user.name", "Int Test")
	os.WriteFile(filepath.Join(tmpDir, "init.txt"), []byte("base"), 0644)
	runCmd(t, tmpDir, "git", "add", ".")
	runCmd(t, tmpDir, "git", "commit", "-m", "initial")

	// 2. Test nekotree logic
	wm := gitworktree.NewWorktreeManager(tmpDir)
	branch := "int-feat"

	if err := wm.CreateWorktree(branch); err != nil {
		t.Fatalf("CreateWorktree failed: %v", err)
	}

	// FIX: Calculate the name exactly how the tool does: nekotree-<repo>-<branch>
	repoName := filepath.Base(tmpDir)
	expectedName := fmt.Sprintf("nekotree-%s-%s", repoName, branch)
	expectedPath := filepath.Join(tmpDir, expectedName)

	// Verify physical directory exists
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Worktree directory was not physically created at %s", expectedPath)
	}

	// 3. Test Removal
	if err := wm.RemoveWorktree(expectedPath); err != nil {
		t.Fatalf("RemoveWorktree failed: %v", err)
	}

	if _, err := os.Stat(expectedPath); !os.IsNotExist(err) {
		t.Errorf("Worktree directory still exists after removal")
	}
}

func runCmd(t *testing.T, dir string, name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("cmd %s %v failed: %v\nOutput: %s", name, args, err, string(out))
	}
}
