package docker

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"cubicheart.com/munchtoast/nekotree/internal/config"
	"cubicheart.com/munchtoast/nekotree/internal/runner"
)

// mockRunner records every call and returns configurable output/error.
type mockRunner struct {
	calls  []string
	output []byte
	err    error
}

func (m *mockRunner) Run(name string, arg ...string) error {
	m.calls = append(m.calls, fmt.Sprintf("%s %s", name, strings.Join(arg, " ")))
	return m.err
}

func (m *mockRunner) CombinedOutput(name string, arg ...string) ([]byte, error) {
	m.calls = append(m.calls, fmt.Sprintf("%s %s", name, strings.Join(arg, " ")))
	return m.output, m.err
}

func (m *mockRunner) hasCall(substr string) bool {
	for _, c := range m.calls {
		if strings.Contains(c, substr) {
			return true
		}
	}
	return false
}

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
	if !mock.hasCall("docker run") {
		t.Error("expected docker run to be called")
	}
	if !mock.hasCall("tail -f /dev/null") {
		t.Errorf("expected tail -f /dev/null default keep-alive, calls: %v", mock.calls)
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
	if !mock.hasCall("make build") {
		t.Errorf("expected explicit command in docker run args, calls: %v", mock.calls)
	}
	if mock.hasCall("tail -f /dev/null") {
		t.Error("should not inject keep-alive when command is provided")
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
	if !mock.hasCall("-p") || !mock.hasCall("8080:3000") {
		t.Errorf("expected port flags to be passed, calls: %v", mock.calls)
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
	if !mock.hasCall("docker compose") {
		t.Errorf("expected docker compose up, calls: %v", mock.calls)
	}
	if !mock.hasCall("docker-compose.yaml") {
		t.Errorf("expected compose file in args, calls: %v", mock.calls)
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
	mock := &mockRunner{err: fmt.Errorf("docker daemon not running")}
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
	if !mock.hasCall("docker stop test-env") {
		t.Errorf("expected docker stop, calls: %v", mock.calls)
	}
	if !mock.hasCall("docker rm -v test-env") {
		t.Errorf("expected docker rm -v, calls: %v", mock.calls)
	}
}

func TestStop_NoSuchContainer(t *testing.T) {
	mock := &mockRunner{output: []byte("No such container"), err: fmt.Errorf("exit status 1")}
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
	if !mock.hasCall("compose") || !mock.hasCall("down -v") {
		t.Errorf("expected compose down -v for compose environment, calls: %v", mock.calls)
	}
}

func TestStop_RemoveFailure(t *testing.T) {
	mock := &mockRunner{output: []byte("permission denied"), err: fmt.Errorf("exit status 1")}
	cm := NewContainerManager("test-env", &config.Config{}, mock)

	err := cm.Stop()
	if err == nil {
		t.Error("expected error when docker rm fails with non-'No such container' output")
	}
}

// --- Exists ---

func TestExists_ContainerFound(t *testing.T) {
	mock := &mockRunner{output: []byte("abc123def456")}
	cm := NewContainerManager("test-env", &config.Config{}, mock)

	if !cm.Exists() {
		t.Error("expected Exists() to return true when container ID is returned")
	}
}

func TestExists_ContainerNotFound(t *testing.T) {
	mock := &mockRunner{output: []byte("")}
	cm := NewContainerManager("test-env", &config.Config{}, mock)

	if cm.Exists() {
		t.Error("expected Exists() to return false when output is empty")
	}
}

func TestExists_RunnerError(t *testing.T) {
	mock := &mockRunner{err: fmt.Errorf("docker not available")}
	cm := NewContainerManager("test-env", &config.Config{}, mock)

	if cm.Exists() {
		t.Error("expected Exists() to return false on runner error")
	}
}

// --- List ---

func TestList_NoEnvironments(t *testing.T) {
	// Single-line output (just the header row) means no environments
	mock := &mockRunner{output: []byte("NAMES\tSTATUS\tIMAGE")}
	cm := NewContainerManager("", &config.Config{}, mock)

	err := cm.List(io.Discard)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
}

func TestList_WithEnvironments(t *testing.T) {
	output := "NAMES\tSTATUS\tIMAGE\nnekotree-repo-feat\tUp 2 hours\talpine:latest"
	mock := &mockRunner{output: []byte(output)}
	cm := NewContainerManager("", &config.Config{}, mock)

	err := cm.List(io.Discard)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
}

func TestList_RunnerError(t *testing.T) {
	mock := &mockRunner{err: fmt.Errorf("connection refused")}
	cm := NewContainerManager("", &config.Config{}, mock)

	err := cm.List(io.Discard)
	if err == nil {
		t.Error("expected error when docker ps fails")
	}
}

// --- RunCommand ---

func TestRunCommand_Success(t *testing.T) {
	// Exists() checks for non-empty output; RunCommand then calls exec
	mock := &mockRunner{output: []byte("container-id")}
	cm := NewContainerManager("nekotree-repo-branch", &config.Config{}, mock)

	err := cm.RunCommand(io.Discard, "make build")
	if err != nil {
		t.Fatalf("RunCommand failed: %v", err)
	}
	if !mock.hasCall("docker exec") {
		t.Errorf("expected docker exec to be called, calls: %v", mock.calls)
	}
	if !mock.hasCall("make build") {
		t.Errorf("expected command to be passed to exec, calls: %v", mock.calls)
	}
}

func TestRunCommand_ContainerNotFound(t *testing.T) {
	mock := &mockRunner{output: []byte("")} // Exists() returns false
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
	mock := &mockRunner{output: []byte("4.0K\t/tmp/worktree")}
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
	mock := &mockRunner{err: fmt.Errorf("no such file")}
	cm := NewContainerManager("test-env", &config.Config{}, mock)

	info := cm.GetInfo("/tmp/worktree")
	if info != "Size: Unknown" {
		t.Errorf("expected 'Size: Unknown' on error, got: %s", info)
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
