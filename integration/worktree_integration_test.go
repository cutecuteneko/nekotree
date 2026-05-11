//go:build integration
// +build integration

package integration

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/cutecuteneko/nekotree/internal/gitworktree"
)

func TestGitWorktreeIntegration(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "nekotree_int_repo_*")
	defer os.RemoveAll(tmpDir)

	runCmd(t, tmpDir, "git", "init")
	runCmd(t, tmpDir, "git", "config", "user.email", "int@test.com")
	runCmd(t, tmpDir, "git", "config", "user.name", "Int Test")
	os.WriteFile(filepath.Join(tmpDir, "init.txt"), []byte("base"), 0644)
	runCmd(t, tmpDir, "git", "add", ".")
	runCmd(t, tmpDir, "git", "commit", "-m", "initial")

	// FIX: Added 'nil' as the second argument to use the production RealRunner
	wm := gitworktree.NewWorktreeManager(tmpDir, nil)
	branch := "int-feat"

	if err := wm.CreateWorktree(branch); err != nil {
		t.Fatalf("CreateWorktree failed: %v", err)
	}

	repoName := filepath.Base(tmpDir)
	expectedName := fmt.Sprintf("nekotree-%s-%s", repoName, branch)
	expectedPath := filepath.Join(tmpDir, expectedName)

	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Worktree directory was not physically created at %s", expectedPath)
	}

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
