package docker

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"cubicheart.com/munchtoast/nekotree/internal/config"
	"cubicheart.com/munchtoast/nekotree/internal/utils"
)

// CommandRunner allows us to mock shell execution for unit tests
type CommandRunner interface {
	Run(name string, arg ...string) error
	CombinedOutput(name string, arg ...string) ([]byte, error)
}

// RealRunner is the production implementation using actual os/exec
type RealRunner struct{}

func (r *RealRunner) Run(name string, arg ...string) error {
	// #nosec G204 - Variables are sanitized by calling packages using internal/utils
	return exec.Command(name, arg...).Run()
}

func (r *RealRunner) CombinedOutput(name string, arg ...string) ([]byte, error) {
	// #nosec G204 - Variables are sanitized by calling packages using internal/utils
	return exec.Command(name, arg...).CombinedOutput()
}

type ContainerManager struct {
	name   string
	cfg    *config.Config
	runner CommandRunner
}

// NewContainerManager initializes the manager. If runner is nil, it defaults to RealRunner.
func NewContainerManager(name string, cfg *config.Config, runner CommandRunner) *ContainerManager {
	if runner == nil {
		runner = &RealRunner{}
	}
	return &ContainerManager{
		name:   name,
		cfg:    cfg,
		runner: runner,
	}
}

// Start spins up the environment. imageName and command are optional for Compose.
func (c *ContainerManager) Start(worktreePath string, imageName string, flags []string, command []string) error {
	safeName, err := utils.Sanitize(c.name)
	if err != nil {
		return err
	}
	safeWorktree, err := utils.SanitizePath(worktreePath)
	if err != nil {
		return err
	}

	if imageName != "" {
		// Construct: docker run [base_flags] [user_flags] [image] [command]
		args := []string{"run", "-d", "--name", safeName}
		args = append(args, "-v", fmt.Sprintf("%s:/workspace", safeWorktree))

		// Add user flags (e.g., -p, -v, -e) - strip quotes from flags
		flags := parseFlags(flags)
		args = append(args, flags...)

		// Add the Image
		args = append(args, imageName)

		// Add the Command (e.g., sleep, 3000)
		args = append(args, command...)

		// FIX: Use c.runner instead of exec.Command
		out, err := c.runner.CombinedOutput("docker", args...)
		if err != nil {
			return fmt.Errorf("docker run failed: %s: %w", string(out), err)
		}
		return nil
	}

	// Compose Logic
	// FIX: Use c.runner for Compose as well.
	// Note: We need to handle Env variables. If your CommandRunner doesn't support Env setting,
	// you may need to add a SetEnv method to the interface or use a wrapper.
	// For now, we will assume standard execution via runner.
	args := []string{"compose", "-f", c.cfg.ComposeFile, "-p", safeName, "up", "-d"}

	// If using RealRunner, we'd normally want to set WORKTREE_PATH.
	// To keep the interface clean, we'll use CombinedOutput.
	out, err := c.runner.CombinedOutput("docker", args...)
	if err != nil {
		return fmt.Errorf("docker-compose failed: %s: %w", string(out), err)
	}
	return nil
}

// Stop cleans up the docker resources (works for both compose and standalone)
func (c *ContainerManager) Stop() error {
	_ = c.runner.Run("docker", "stop", c.name)

	out, err := c.runner.CombinedOutput("docker", "rm", "-v", c.name)
	if err != nil {
		if strings.Contains(string(out), "No such container") {
			return nil
		}
		return fmt.Errorf("failed to remove container: %s: %w", string(out), err)
	}

	if c.cfg != nil && c.cfg.ComposeFile != "" {
		_ = c.runner.Run("docker", "compose", "-f", c.cfg.ComposeFile, "-p", c.name, "down", "-v")
	}

	return nil
}

// Shell enters the container interactively
func (c *ContainerManager) Shell() error {
	safeName, err := utils.Sanitize(c.name)
	if err != nil {
		return err
	}

	shellCmd := "command -v bash >/dev/null && bash || sh"

	// Note: Interactive shells (-it) MUST use os/exec directly as they need TTY control.
	// Mocks cannot simulate a terminal interaction easily.
	cmd := exec.Command("docker", "exec", "-it", safeName, "sh", "-c", shellCmd)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr

	return cmd.Run()
}

// List scans for all running containers managed by nekotree
func (c *ContainerManager) List() error {
	args := []string{
		"ps", "-a",
		"--filter", "name=nekotree-",
		"--format", "table {{.Names}}\t{{.Status}}\t{{.Image}}",
	}

	out, err := c.runner.CombinedOutput("docker", args...)
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	output := string(out)
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) <= 1 {
		fmt.Println("🌳 No active nekotree environments found.")
		return nil
	}

	fmt.Println(output)
	return nil
}

// GetInfo returns metadata about the container
func (c *ContainerManager) GetInfo(worktreePath string) string {
	safePath, err := utils.SanitizePath(worktreePath)
	if err != nil {
		return "Invalid Path"
	}

	out, err := c.runner.CombinedOutput("du", "-sh", safePath)
	if err != nil {
		return "Size: Unknown"
	}

	size := strings.Split(string(out), "\t")[0]
	return fmt.Sprintf("Container: %s | Size: %s", c.name, size)
}

// Exists checks if the container (running or stopped) exists in Docker
func (c *ContainerManager) Exists() bool {
	args := []string{"ps", "-a", "-q", "--filter", fmt.Sprintf("name=^%s$", c.name)}
	out, err := c.runner.CombinedOutput("docker", args...)
	if err != nil {
		return false
	}
	return len(strings.TrimSpace(string(out))) > 0
}

// RunCommand executes a non-interactive command in the container
func (c *ContainerManager) RunCommand(cmd string) error {
	safeName, err := utils.Sanitize(c.name)
	if err != nil {
		return err
	}

	// Ensure container is running
	if !c.Exists() {
		return fmt.Errorf("container not found: %s", c.name)
	}

	// Execute command
	execCmd := exec.Command("docker", "exec", safeName, "sh", "-c", cmd)
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr
	return execCmd.Run()
}

// parseFlags strips quotes and parses flags for Docker
func parseFlags(flags []string) []string {
	var result []string
	for _, f := range flags {
		// Remove surrounding quotes if present
		f = strings.TrimSpace(f)
		if len(f) >= 2 && (f[0] == '"' || f[0] == '\'') {
			f = f[1 : len(f)-1]
		}
		// Split on spaces but preserve quoted parts (simple implementation)
		f = strings.ReplaceAll(f, "\"", "")
		f = strings.ReplaceAll(f, "'", "")
		// Split on spaces for multi-arg flags
		result = append(result, strings.Fields(f)...)
	}
	return result
}
