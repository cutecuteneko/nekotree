package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"cubicheart.com/munchtoast/nekotree/internal/testutil"
	"github.com/urfave/cli/v2"
)

// ---------------------------------------------------------------------------
// Test infrastructure
// ---------------------------------------------------------------------------

// mockRunner is a package-local alias kept for backward compatibility with
// test helpers in this file that use *mockRunner. The shared implementation
// lives in internal/testutil.
type mockRunner = testutil.MockRunner

// setupTestRepo creates a real temporary git repo with an initial commit.
// Required because git worktree commands need a valid .git context.
func setupTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# test"), 0644); err != nil {
		t.Fatalf("write README: %v", err)
	}
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "init")
	return dir
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// cliApp builds a minimal cli.App wired to the given action function so we
// can invoke it with controlled args without going through os.Args.
func appWith(cmd *cli.Command) *cli.App {
	return &cli.App{Commands: []*cli.Command{cmd}}
}

// chdirTest changes to dir and registers a cleanup that restores the original
// working directory, preventing cwd leakage between tests.
func chdirTest(t *testing.T, dir string) {
	t.Helper()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })
}

// ---------------------------------------------------------------------------
// App / command registry
// ---------------------------------------------------------------------------

func TestAppCommands(t *testing.T) {
	app := &cli.App{
		Commands: []*cli.Command{
			createCmd(),
			runCmd(),
			shellCmd(),
			listCmd(),
			removeCmd(),
		},
	}
	for _, name := range []string{"create", "run", "shell", "list", "remove"} {
		if app.Command(name) == nil {
			t.Errorf("command %q missing from registry", name)
		}
	}
}

func TestAppCommandAliases(t *testing.T) {
	app := &cli.App{
		Commands: []*cli.Command{
			createCmd(),
			runCmd(),
			shellCmd(),
			listCmd(),
			removeCmd(),
		},
	}
	cases := map[string][]string{
		"create": {"c"},
		"run":    {"r"},
		"shell":  {"sh", "s"},
		"list":   {"ls"},
		"remove": {"rm"},
	}
	for cmdName, want := range cases {
		cmd := app.Command(cmdName)
		if cmd == nil {
			t.Errorf("command %q not found", cmdName)
			continue
		}
		for _, alias := range want {
			found := false
			for _, a := range cmd.Aliases {
				if a == alias {
					found = true
				}
			}
			if !found {
				t.Errorf("%q missing alias %q, got %v", cmdName, alias, cmd.Aliases)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// splitCommand
// ---------------------------------------------------------------------------

func TestSplitCommand_Empty(t *testing.T) {
	if r := splitCommand(""); r != nil {
		t.Errorf("expected nil, got %v", r)
	}
}

func TestSplitCommand_SingleWord(t *testing.T) {
	r := splitCommand("bash")
	if len(r) != 1 || r[0] != "bash" {
		t.Errorf("expected [bash], got %v", r)
	}
}

func TestSplitCommand_MultipleWords(t *testing.T) {
	r := splitCommand("npm run build")
	if len(r) != 3 || r[0] != "npm" || r[1] != "run" || r[2] != "build" {
		t.Errorf("unexpected: %v", r)
	}
}

func TestSplitCommand_QuotedArgument(t *testing.T) {
	r := splitCommand(`echo "hello world"`)
	if len(r) != 2 || r[1] != "hello world" {
		t.Errorf("expected echo + 'hello world', got %v", r)
	}
}

func TestSplitCommand_LeadingTrailingSpaces(t *testing.T) {
	r := splitCommand("  make   build  ")
	if len(r) != 2 {
		t.Errorf("expected 2 parts, got %d: %v", len(r), r)
	}
}

// ---------------------------------------------------------------------------
// runCreateAction
// ---------------------------------------------------------------------------

func runCreate(t *testing.T, mock *mockRunner, args ...string) error {
	t.Helper()
	app := appWith(createCmd())
	// Swap real action with injectable one
	app.Commands[0].Action = func(c *cli.Context) error {
		return runCreateAction(c, mock)
	}
	return app.Run(append([]string{"app", "create"}, args...))
}

func TestCreateAction_MissingBranch(t *testing.T) {
	mock := &mockRunner{}
	err := runCreate(t, mock)
	if err == nil || !strings.Contains(err.Error(), "branch name required") {
		t.Errorf("expected 'branch name required', got: %v", err)
	}
}

func TestCreateAction_InvalidBranch(t *testing.T) {
	mock := &mockRunner{}
	err := runCreate(t, mock, "bad branch;inject")
	if err == nil {
		t.Error("expected sanitization error for invalid branch name")
	}
}

func TestCreateAction_WithImage(t *testing.T) {
	repoDir := setupTestRepo(t)
	chdirTest(t, repoDir)

	mock := &mockRunner{}
	err := runCreate(t, mock, "feature-test", "golang:latest")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mock.HasCall("git") {
		t.Error("expected git worktree call")
	}
	if !mock.HasCall("docker run") {
		t.Errorf("expected docker run, calls: %v", mock.Calls)
	}
	if !mock.HasCall("golang:latest") {
		t.Errorf("expected image in docker run args, calls: %v", mock.Calls)
	}
}

func TestCreateAction_DefaultKeepAlive(t *testing.T) {
	repoDir := setupTestRepo(t)
	chdirTest(t, repoDir)

	mock := &mockRunner{}
	// Provide an image but no command → should inject tail -f /dev/null
	err := runCreate(t, mock, "feature-keepalive", "alpine:latest")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mock.HasCall("tail -f /dev/null") {
		t.Errorf("expected tail -f /dev/null keep-alive, calls: %v", mock.Calls)
	}
}

func TestCreateAction_WithComposeFile(t *testing.T) {
	repoDir := setupTestRepo(t)
	chdirTest(t, repoDir)

	// Write a real file so os.Stat detects it as a compose file
	composePath := filepath.Join(repoDir, "docker-compose.yaml")
	if err := os.WriteFile(composePath, []byte("version: '3'"), 0644); err != nil {
		t.Fatal(err)
	}

	mock := &mockRunner{}
	err := runCreate(t, mock, "feature-compose", composePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mock.HasCall("docker compose") {
		t.Errorf("expected docker compose up, calls: %v", mock.Calls)
	}
}

func TestCreateAction_WithFlags(t *testing.T) {
	repoDir := setupTestRepo(t)
	chdirTest(t, repoDir)

	mock := &mockRunner{}
	err := runCreate(t, mock, "-f", "-p 8080:3000", "feature-ports", "node:18")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mock.HasCall("-p") || !mock.HasCall("8080:3000") {
		t.Errorf("expected port flag forwarded to docker run, calls: %v", mock.Calls)
	}
}

func TestCreateAction_WithFlagsAfterImage(t *testing.T) {
	repoDir := setupTestRepo(t)
	chdirTest(t, repoDir)

	// -f appears after the image, which urfave/cli v2 does not parse as a flag.
	// The bug caused "-f" to be treated as the container command.
	mock := &mockRunner{}
	err := runCreate(t, mock, "feature-ports", "node:18", "-f", "-p 8080:3000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mock.HasCall("-p") || !mock.HasCall("8080:3000") {
		t.Errorf("expected port flag forwarded to docker run, calls: %v", mock.Calls)
	}
	if mock.HasCall("exec: \"-f\"") {
		t.Error("'-f' must not appear as the container command")
	}
}

func TestCreateAction_WithExplicitCommand(t *testing.T) {
	repoDir := setupTestRepo(t)
	chdirTest(t, repoDir)

	mock := &mockRunner{}
	err := runCreate(t, mock, "feature-cmd", "node:18", "npm", "start")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mock.HasCall("npm") {
		t.Errorf("expected explicit command forwarded, calls: %v", mock.Calls)
	}
	if mock.HasCall("tail -f /dev/null") {
		t.Error("should not inject keep-alive when explicit command provided")
	}
}

func TestCreateAction_WithEnvFlag(t *testing.T) {
	repoDir := setupTestRepo(t)
	chdirTest(t, repoDir)

	envFile := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(envFile, []byte("KEY=val"), 0600); err != nil {
		t.Fatal(err)
	}

	mock := &mockRunner{}
	err := runCreate(t, mock, "--env", envFile, "feature-env", "alpine:latest")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mock.HasCall("--env-file") {
		t.Errorf("expected --env-file in docker run args, calls: %v", mock.Calls)
	}
	if !mock.HasCall(envFile) {
		t.Errorf("expected env file path in docker run args, calls: %v", mock.Calls)
	}
}

func TestCreateAction_ComposeAutoEnvDefault(t *testing.T) {
	repoDir := setupTestRepo(t)
	chdirTest(t, repoDir)

	// Write compose file and sibling .env in the same directory
	composePath := filepath.Join(repoDir, "docker-compose.yaml")
	if err := os.WriteFile(composePath, []byte("version: '3'"), 0644); err != nil {
		t.Fatal(err)
	}
	envPath := filepath.Join(repoDir, ".env")
	if err := os.WriteFile(envPath, []byte("KEY=val"), 0600); err != nil {
		t.Fatal(err)
	}

	mock := &mockRunner{}
	err := runCreate(t, mock, "feature-auto-env", composePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mock.HasCall("--env-file") {
		t.Errorf("expected --env-file auto-detected from compose dir, calls: %v", mock.Calls)
	}
	if !mock.HasCall(envPath) {
		t.Errorf("expected sibling .env path in docker compose args, calls: %v", mock.Calls)
	}
}

func TestCreateAction_ComposeNoEnvFile(t *testing.T) {
	repoDir := setupTestRepo(t)
	chdirTest(t, repoDir)

	// Write compose file but no .env sibling
	composePath := filepath.Join(repoDir, "docker-compose.yaml")
	if err := os.WriteFile(composePath, []byte("version: '3'"), 0644); err != nil {
		t.Fatal(err)
	}

	mock := &mockRunner{}
	err := runCreate(t, mock, "feature-no-env", composePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.HasCall("--env-file") {
		t.Errorf("expected no --env-file when .env does not exist, calls: %v", mock.Calls)
	}
}

// ---------------------------------------------------------------------------
// runRunAction
// ---------------------------------------------------------------------------

func runRun(t *testing.T, mock *mockRunner, args ...string) error {
	t.Helper()
	app := appWith(runCmd())
	app.Commands[0].Action = func(c *cli.Context) error {
		return runRunAction(c, mock)
	}
	return app.Run(append([]string{"app", "run"}, args...))
}

func TestRunAction_MissingBranch(t *testing.T) {
	mock := &mockRunner{}
	err := runRun(t, mock)
	if err == nil || !strings.Contains(err.Error(), "branch required") {
		t.Errorf("expected 'branch required', got: %v", err)
	}
}

func TestRunAction_InvalidBranch(t *testing.T) {
	mock := &mockRunner{}
	err := runRun(t, mock, "bad;branch", "make", "test")
	if err == nil {
		t.Error("expected sanitization error")
	}
}

func TestRunAction_ContainerExists_RunsCommand(t *testing.T) {
	repoDir := setupTestRepo(t)
	chdirTest(t, repoDir)

	// Return non-empty output so Exists() returns true, then exec output
	mock := &mockRunner{Output: []byte("container-id")}
	err := runRun(t, mock, "my-branch", "make", "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mock.HasCall("docker exec") {
		t.Errorf("expected docker exec, calls: %v", mock.Calls)
	}
	if !mock.HasCall("make test") {
		t.Errorf("expected command in exec, calls: %v", mock.Calls)
	}
}

func TestRunAction_NoContainer_NoWorktree(t *testing.T) {
	repoDir := setupTestRepo(t)
	chdirTest(t, repoDir)

	// Empty output → Exists() false; worktree directory won't exist either
	mock := &mockRunner{Output: []byte("")}
	err := runRun(t, mock, "no-such-branch", "make", "test")
	if err == nil || !strings.Contains(err.Error(), "worktree not found") {
		t.Errorf("expected 'worktree not found', got: %v", err)
	}
}

func TestRunAction_NoContainer_WorktreeExists_ReturnsError(t *testing.T) {
	repoDir := setupTestRepo(t)
	chdirTest(t, repoDir)

	// Create the worktree directory so w.Exists() returns true
	repoName := filepath.Base(repoDir)
	worktreeDir := filepath.Join(repoDir, fmt.Sprintf("nekotree-%s-auto-branch", repoName))
	if err := os.Mkdir(worktreeDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Container does not exist (empty output from docker ps)
	mock := &mockRunner{Output: []byte("")}
	err := runRun(t, mock, "auto-branch", "make", "build")
	if err == nil {
		t.Fatal("expected error when container is missing but worktree exists")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("expected 'does not exist' error, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// runListAction
// ---------------------------------------------------------------------------

func runList(t *testing.T, mock *mockRunner) error {
	t.Helper()
	app := appWith(listCmd())
	app.Commands[0].Action = func(c *cli.Context) error {
		return runListAction(c, mock)
	}
	return app.Run([]string{"app", "list"})
}

func TestListAction_Empty(t *testing.T) {
	mock := &mockRunner{Output: []byte("NAMES\tSTATUS\tIMAGE")}
	if err := runList(t, mock); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !mock.HasCall("docker ps") {
		t.Errorf("expected docker ps, calls: %v", mock.Calls)
	}
}

func TestListAction_WithResults(t *testing.T) {
	output := "NAMES\tSTATUS\tIMAGE\nnekotree-repo-feat\tUp\talpine:latest"
	mock := &mockRunner{Output: []byte(output)}
	if err := runList(t, mock); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestListAction_RunnerError(t *testing.T) {
	mock := &mockRunner{Err: fmt.Errorf("docker unavailable")}
	err := runList(t, mock)
	if err == nil {
		t.Error("expected error when docker ps fails")
	}
}

// ---------------------------------------------------------------------------
// runShellAction
// ---------------------------------------------------------------------------

func runShell(t *testing.T, mock *mockRunner, args ...string) error {
	t.Helper()
	app := appWith(shellCmd())
	app.Commands[0].Action = func(c *cli.Context) error {
		return runShellAction(c, mock)
	}
	return app.Run(append([]string{"app", "shell"}, args...))
}

func TestShellAction_MissingBranch(t *testing.T) {
	mock := &mockRunner{}
	err := runShell(t, mock)
	if err == nil || !strings.Contains(err.Error(), "branch required") {
		t.Errorf("expected 'branch required', got: %v", err)
	}
}

func TestShellAction_InvalidBranch(t *testing.T) {
	mock := &mockRunner{}
	err := runShell(t, mock, "bad;branch")
	if err == nil {
		t.Error("expected sanitization error for invalid branch name")
	}
}

// ---------------------------------------------------------------------------
// runRemoveAction
// ---------------------------------------------------------------------------

func runRemove(t *testing.T, mock *mockRunner, args ...string) error {
	t.Helper()
	app := appWith(removeCmd())
	app.Commands[0].Action = func(c *cli.Context) error {
		return runRemoveAction(c, mock)
	}
	return app.Run(append([]string{"app", "remove"}, args...))
}

func TestRemoveAction_MissingInput(t *testing.T) {
	mock := &mockRunner{}
	err := runRemove(t, mock)
	if err == nil || !strings.Contains(err.Error(), "name or branch required") {
		t.Errorf("expected 'name or branch required', got: %v", err)
	}
}

func TestRemoveAction_NothingExists(t *testing.T) {
	repoDir := setupTestRepo(t)
	chdirTest(t, repoDir)

	// Empty output → both Exists() calls return false → nothing to do
	mock := &mockRunner{Output: []byte("")}
	err := runRemove(t, mock, "ghost-branch")
	if err != nil {
		t.Errorf("expected nil when nothing to remove, got: %v", err)
	}
}

func TestRemoveAction_FullPrefixPassthrough(t *testing.T) {
	repoDir := setupTestRepo(t)
	repoName := filepath.Base(repoDir)
	chdirTest(t, repoDir)

	fullName := fmt.Sprintf("nekotree-%s-my-branch", repoName)
	// Return container ID so Exists() is true, then stop/rm succeed
	mock := &mockRunner{Output: []byte("container-id")}
	err := runRemove(t, mock, fullName)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should stop using the exact fullName, not double-prefixed
	if !mock.HasCall(fmt.Sprintf("docker stop %s", fullName)) {
		t.Errorf("expected stop with full name %s, calls: %v", fullName, mock.Calls)
	}
}

func TestRemoveAction_BranchNamePrefixed(t *testing.T) {
	repoDir := setupTestRepo(t)
	repoName := filepath.Base(repoDir)
	chdirTest(t, repoDir)

	mock := &mockRunner{Output: []byte("container-id")}
	err := runRemove(t, mock, "my-branch")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should prepend prefix
	expectedName := fmt.Sprintf("nekotree-%s-my-branch", repoName)
	if !mock.HasCall(fmt.Sprintf("docker stop %s", expectedName)) {
		t.Errorf("expected stop with prefixed name %s, calls: %v", expectedName, mock.Calls)
	}
}

func TestRemoveAction_CallsStopAndWorktreeRemove(t *testing.T) {
	repoDir := setupTestRepo(t)
	chdirTest(t, repoDir)

	mock := &mockRunner{Output: []byte("container-id")}
	if err := runRemove(t, mock, "cleanup-branch"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mock.HasCall("docker stop") {
		t.Errorf("expected docker stop, calls: %v", mock.Calls)
	}
	if !mock.HasCall("git") {
		t.Errorf("expected git worktree prune/remove, calls: %v", mock.Calls)
	}
}
