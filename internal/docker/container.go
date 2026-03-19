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
	return exec.Command(name, arg...).Run()
}

func (r *RealRunner) CombinedOutput(name string, arg ...string) ([]byte, error) {
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

// Start spins up the environment using docker-compose
func (c *ContainerManager) Start(worktreePath string) error {
	safeName, err := utils.Sanitize(c.name)
	if err != nil {
		return err
	}

	safeWorktree, err := utils.SanitizePath(worktreePath)
	if err != nil {
		return err
	}

	fmt.Printf("🚀 Starting environment: %s\n", safeName)

	// We pass variables as separate arguments to avoid shell injection
	args := []string{"compose", "-f", c.cfg.ComposeFile, "-p", safeName, "up", "-d"}

	// Prepare the command with injected environment variables
	cmd := exec.Command("docker", args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("WORKTREE_PATH=%s", safeWorktree))

	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("docker-compose failed: %s: %w", string(out), err)
	}
	return nil
}

// Stop cleans up the docker resources
func (c *ContainerManager) Stop() error {
	safeName, err := utils.Sanitize(c.name)
	if err != nil {
		return err
	}

	fmt.Printf("🗑️  Cleaning up environment: %s\n", safeName)

	// Handle cleanup errors gracefully
	if err := c.runner.Run("docker", "compose", "-p", safeName, "down"); err != nil {
		fmt.Printf("⚠️  Warning: compose down encountered an issue: %v\n", err)
	}

	if err := c.runner.Run("docker", "rm", "-f", safeName); err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
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

	// Note: Interactive shells (-it) must use os/exec directly as they need TTY control
	cmd := exec.Command("docker", "exec", "-it", safeName, "sh", "-c", shellCmd)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr

	return cmd.Run()
}

// List scans for all running containers managed by nekotree
func (c *ContainerManager) List() error {
	// Filter for containers starting with our prefix
	args := []string{
		"ps",
		"--filter", "name=nekotree-",
		"--format", "table {{.Names}}\t{{.Status}}\t{{.Image}}",
	}

	out, err := c.runner.CombinedOutput("docker", args...)
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	output := string(out)
	if strings.TrimSpace(output) == "" || strings.Count(output, "\n") < 2 {
		fmt.Println("🌳 No active nekotree environments found.")
		return nil
	}

	fmt.Println(output)
	return nil
}

// GetInfo returns metadata about the container (e.g., disk usage)
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
