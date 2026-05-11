package docker

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"cubicheart.com/munchtoast/nekotree/internal/config"
	"cubicheart.com/munchtoast/nekotree/internal/runner"
	"cubicheart.com/munchtoast/nekotree/internal/testutil"
)

// mockRunner is a package-local alias for the shared implementation.
type mockRunner = testutil.MockRunner

// --- NewContainerManager ---

func TestNewContainerManager_DefaultsToRealRunner(t *testing.T) {
	cm := NewContainerManager("test", &config.Config{}, nil)
	if cm.runner == nil {
		t.Fatal("expected non-nil runner when nil is passed")
	}
	if _, ok := cm.runner.(*runner.RealRunner); !ok {
		t.Errorf("expected RealRunner, got %T", cm.runner)
	}
}

func TestNewContainerManager_UsesProvidedRunner(t *testing.T) {
	mock := &mockRunner{}
	cm := NewContainerManager("test", &config.Config{}, mock)
	if cm.runner != mock {
		t.Error("expected the provided mock runner to be used")
	}
}

// --- Start ---

func TestStart_ImageWithDefaultKeepAlive(t *testing.T) {
	mock := &mockRunner{}
	cm := NewContainerManager("nekotree-repo-branch", &config.Config{}, mock)

	err := cm.Start(StartOptions{WorktreePath: "/tmp/worktree", ImageName: "alpine:latest"})
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if !mock.HasCall("docker run") {
		t.Error("expected docker run to be called")
	}
	if !mock.HasCall("tail -f /dev/null") {
		t.Errorf("expected tail -f /dev/null default keep-alive, calls: %v", mock.Calls)
	}
}

func TestStart_ImageWithExplicitCommand(t *testing.T) {
	mock := &mockRunner{}
	cm := NewContainerManager("nekotree-repo-branch", &config.Config{}, mock)

	err := cm.Start(StartOptions{
		WorktreePath: "/tmp/worktree",
		ImageName:    "golang:latest",
		Command:      []string{"make", "build"},
	})
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if !mock.HasCall("make build") {
		t.Errorf("expected explicit command in docker run args, calls: %v", mock.Calls)
	}
	if mock.HasCall("tail -f /dev/null") {
		t.Error("should not inject keep-alive when command is provided")
	}
	if !mock.HasCall("--label") || !mock.HasCall("com.nekotree.worktree.path=/tmp/worktree") {
		t.Errorf("expected label in docker run command, calls: %v", mock.Calls)
	}
}

func TestStart_ImageWithFlags(t *testing.T) {
	mock := &mockRunner{}
	cm := NewContainerManager("nekotree-repo-branch", &config.Config{}, mock)

	err := cm.Start(StartOptions{
		WorktreePath: "/tmp/worktree",
		ImageName:    "node:18",
		Flags:        []string{"-p 8080:3000"},
	})
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if !mock.HasCall("-p") || !mock.HasCall("8080:3000") {
		t.Errorf("expected port flags to be passed, calls: %v", mock.Calls)
	}
}

func TestStart_ComposeFile(t *testing.T) {
	mock := &mockRunner{}
	cfg := &config.Config{ComposeFile: "docker-compose.yaml"}
	cm := NewContainerManager("nekotree-repo-branch", cfg, mock)

	err := cm.Start(StartOptions{WorktreePath: "/tmp/worktree"})
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if !mock.HasCall("docker compose") {
		t.Errorf("expected docker compose up, calls: %v", mock.Calls)
	}
	if !mock.HasCall("docker-compose.yaml") {
		t.Errorf("expected compose file in args, calls: %v", mock.Calls)
	}
}

func TestStart_NilConfig(t *testing.T) {
	mock := &mockRunner{}
	cm := NewContainerManager("nekotree-repo-branch", nil, mock)

	err := cm.Start(StartOptions{WorktreePath: "/tmp/worktree", ImageName: "alpine:latest"})
	if err != nil {
		t.Fatalf("Start with nil config failed: %v", err)
	}
	if !mock.HasCall("docker run") {
		t.Error("expected docker run to be called even when config was nil")
	}
}

func TestStart_InvalidContainerName(t *testing.T) {
	mock := &mockRunner{}
	cm := NewContainerManager("bad name; rm -rf", &config.Config{}, mock)

	err := cm.Start(StartOptions{WorktreePath: "/tmp/worktree", ImageName: "alpine:latest"})
	if err == nil {
		t.Error("expected error for invalid container name")
	}
}

func TestStart_InvalidWorktreePath(t *testing.T) {
	mock := &mockRunner{}
	cm := NewContainerManager("nekotree-repo-branch", &config.Config{}, mock)

	err := cm.Start(StartOptions{WorktreePath: "../../etc/passwd", ImageName: "alpine:latest"})
	if err == nil {
		t.Error("expected error for directory traversal path")
	}
}

func TestStart_RunnerError(t *testing.T) {
	mock := &mockRunner{Err: fmt.Errorf("docker daemon not running")}
	cm := NewContainerManager("nekotree-repo-branch", &config.Config{}, mock)

	err := cm.Start(StartOptions{WorktreePath: "/tmp/worktree", ImageName: "alpine:latest"})
	if err == nil {
		t.Error("expected error when runner returns error")
	}
}

// --- Stop ---

func TestStop_StopsAndRemovesContainer(t *testing.T) {
	mock := &mockRunner{}
	cm := NewContainerManager("test-env", &config.Config{}, mock)

	err := cm.Stop()
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
	if !mock.HasCall("docker stop test-env") {
		t.Errorf("expected docker stop, calls: %v", mock.Calls)
	}
	if !mock.HasCall("docker rm -v test-env") {
		t.Errorf("expected docker rm -v, calls: %v", mock.Calls)
	}
}

func TestStop_NoSuchContainer(t *testing.T) {
	mock := &mockRunner{Output: []byte("No such container"), Err: fmt.Errorf("exit status 1")}
	cm := NewContainerManager("test-env", &config.Config{}, mock)

	err := cm.Stop()
	if err != nil {
		t.Errorf("expected nil error for missing container, got: %v", err)
	}
}

func TestStop_ComposeTeardown(t *testing.T) {
	mock := &mockRunner{}
	cfg := &config.Config{ComposeFile: "docker-compose.yaml"}
	cm := NewContainerManager("test-env", cfg, mock)

	err := cm.Stop()
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
	if !mock.HasCall("compose") || !mock.HasCall("down -v") {
		t.Errorf("expected compose down -v for compose environment, calls: %v", mock.Calls)
	}
}

func TestStop_RemoveFailure(t *testing.T) {
	mock := &mockRunner{Output: []byte("permission denied"), Err: fmt.Errorf("exit status 1")}
	cm := NewContainerManager("test-env", &config.Config{}, mock)

	err := cm.Stop()
	if err == nil {
		t.Error("expected error when docker rm fails with non-'No such container' output")
	}
}

// --- Exists ---

func TestExists_ContainerFound(t *testing.T) {
	mock := &mockRunner{Output: []byte("abc123def456")}
	cm := NewContainerManager("test-env", &config.Config{}, mock)

	if !cm.Exists() {
		t.Error("expected Exists() to return true when container ID is returned")
	}
}

func TestExists_ContainerNotFound(t *testing.T) {
	mock := &mockRunner{Output: []byte("")}
	cm := NewContainerManager("test-env", &config.Config{}, mock)

	if cm.Exists() {
		t.Error("expected Exists() to return false when output is empty")
	}
}

func TestExists_RunnerError(t *testing.T) {
	mock := &mockRunner{Err: fmt.Errorf("docker not available")}
	cm := NewContainerManager("test-env", &config.Config{}, mock)

	if cm.Exists() {
		t.Error("expected Exists() to return false on runner error")
	}
}

// --- List ---

func TestList_NoEnvironments(t *testing.T) {
	// Single-line output (just the header row) means no environments
	mock := &mockRunner{Output: []byte("NAMES\tSTATUS\tIMAGE")}
	cm := NewContainerManager("", &config.Config{}, mock)

	err := cm.List(io.Discard)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
}

func TestList_WithEnvironments(t *testing.T) {
	output := "NAMES\tSTATUS\tIMAGE\nnekotree-repo-feat\tUp 2 hours\talpine:latest"
	mock := &mockRunner{Output: []byte(output)}
	cm := NewContainerManager("", &config.Config{}, mock)

	err := cm.List(io.Discard)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
}

func TestList_RunnerError(t *testing.T) {
	mock := &mockRunner{Err: fmt.Errorf("connection refused")}
	cm := NewContainerManager("", &config.Config{}, mock)

	err := cm.List(io.Discard)
	if err == nil {
		t.Error("expected error when docker ps fails")
	}
}

// --- RunCommand ---

func TestRunCommand_Success(t *testing.T) {
	// Exists() checks for non-empty output; RunCommand then calls exec
	mock := &mockRunner{Output: []byte("container-id")}
	cm := NewContainerManager("nekotree-repo-branch", &config.Config{}, mock)

	err := cm.RunCommand(io.Discard, "make build")
	if err != nil {
		t.Fatalf("RunCommand failed: %v", err)
	}
	if !mock.HasCall("docker exec") {
		t.Errorf("expected docker exec to be called, calls: %v", mock.Calls)
	}
	if !mock.HasCall("make build") {
		t.Errorf("expected command to be passed to exec, calls: %v", mock.Calls)
	}
}

func TestRunCommand_ContainerNotFound(t *testing.T) {
	mock := &mockRunner{Output: []byte("")} // Exists() returns false
	cm := NewContainerManager("nekotree-repo-branch", &config.Config{}, mock)

	err := cm.RunCommand(io.Discard, "make build")
	if err == nil {
		t.Error("expected error when container does not exist")
	}
	if !strings.Contains(err.Error(), "container not found") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRunCommand_InvalidName(t *testing.T) {
	mock := &mockRunner{}
	cm := NewContainerManager("bad; injection", &config.Config{}, mock)

	err := cm.RunCommand(io.Discard, "ls")
	if err == nil {
		t.Error("expected sanitization error for invalid container name")
	}
}

// --- GetInfo ---

func TestGetInfo_ValidPath(t *testing.T) {
	mock := &mockRunner{Output: []byte("4.0K\t/tmp/worktree")}
	cm := NewContainerManager("test-env", &config.Config{}, mock)

	info := cm.GetInfo("/tmp/worktree")
	if !strings.Contains(info, "test-env") {
		t.Errorf("expected container name in info, got: %s", info)
	}
	if !strings.Contains(info, "4.0K") {
		t.Errorf("expected size in info, got: %s", info)
	}
}

func TestGetInfo_InvalidPath(t *testing.T) {
	mock := &mockRunner{}
	cm := NewContainerManager("test-env", &config.Config{}, mock)

	info := cm.GetInfo("../../etc/passwd")
	if info != "Invalid Path" {
		t.Errorf("expected 'Invalid Path' for traversal path, got: %s", info)
	}
}

func TestGetInfo_DuError(t *testing.T) {
	mock := &mockRunner{Err: fmt.Errorf("no such file")}
	cm := NewContainerManager("test-env", &config.Config{}, mock)

	info := cm.GetInfo("/tmp/worktree")
	if info != "Size: Unknown" {
		t.Errorf("expected 'Size: Unknown' on error, got: %s", info)
	}
}

// --- parseFlags ---

func TestStart_ImageWithEnvFile(t *testing.T) {
	mock := &mockRunner{}
	cm := NewContainerManager("nekotree-repo-branch", &config.Config{}, mock)

	err := cm.Start(StartOptions{
		WorktreePath: "/tmp/worktree",
		ImageName:    "alpine:latest",
		EnvFile:      "/tmp/.env",
	})
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if !mock.HasCall("--env-file") {
		t.Errorf("expected --env-file in docker run args, calls: %v", mock.Calls)
	}
	if !mock.HasCall("/tmp/.env") {
		t.Errorf("expected env file path in docker run args, calls: %v", mock.Calls)
	}
}

func TestStart_ImageWithInvalidEnvFile(t *testing.T) {
	mock := &mockRunner{}
	cm := NewContainerManager("nekotree-repo-branch", &config.Config{}, mock)

	err := cm.Start(StartOptions{
		WorktreePath: "/tmp/worktree",
		ImageName:    "alpine:latest",
		EnvFile:      "../../etc/passwd",
	})
	if err == nil {
		t.Error("expected error for directory traversal env file path")
	}
}

func TestStart_ComposeWithEnvFile(t *testing.T) {
	mock := &mockRunner{}
	cfg := &config.Config{ComposeFile: "docker-compose.yaml"}
	cm := NewContainerManager("nekotree-repo-branch", cfg, mock)

	err := cm.Start(StartOptions{
		WorktreePath: "/tmp/worktree",
		EnvFile:      "/tmp/.env",
	})
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if !mock.HasCall("--env-file") {
		t.Errorf("expected --env-file in docker compose args, calls: %v", mock.Calls)
	}
	if !mock.HasCall("/tmp/.env") {
		t.Errorf("expected env file path in docker compose args, calls: %v", mock.Calls)
	}
}

func TestStart_ComposeWithInvalidEnvFile(t *testing.T) {
	mock := &mockRunner{}
	cfg := &config.Config{ComposeFile: "docker-compose.yaml"}
	cm := NewContainerManager("nekotree-repo-branch", cfg, mock)

	err := cm.Start(StartOptions{
		WorktreePath: "/tmp/worktree",
		EnvFile:      "../../etc/passwd",
	})
	if err == nil {
		t.Error("expected error for directory traversal env file path in compose mode")
	}
}

// --- parseFlags ---

func TestParseFlags_Empty(t *testing.T) {
	result := parseFlags(nil)
	if len(result) != 0 {
		t.Errorf("expected empty result for nil flags, got: %v", result)
	}
}

func TestParseFlags_SingleFlag(t *testing.T) {
	result := parseFlags([]string{"-p 8080:3000"})
	if len(result) != 2 || result[0] != "-p" || result[1] != "8080:3000" {
		t.Errorf("expected [\"-p\", \"8080:3000\"], got: %v", result)
	}
}

func TestParseFlags_QuotedFlag(t *testing.T) {
	result := parseFlags([]string{`"-p 8080:3000"`})
	if len(result) != 2 || result[0] != "-p" || result[1] != "8080:3000" {
		t.Errorf("expected quotes stripped and split, got: %v", result)
	}
}

func TestParseFlags_MultipleFlags(t *testing.T) {
	result := parseFlags([]string{"-p 8080:3000", "-e NODE_ENV=prod"})
	if len(result) != 4 {
		t.Errorf("expected 4 elements, got %d: %v", len(result), result)
	}
}
