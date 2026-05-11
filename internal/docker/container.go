package docker

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/cutecuteneko/nekotree/internal/config"
	"github.com/cutecuteneko/nekotree/internal/runner"
	"github.com/cutecuteneko/nekotree/internal/utils"
	"github.com/cutecuteneko/nekotree/internal/volumes"
)

// StartOptions configures how a container environment is launched.
type StartOptions struct {
	WorktreePath string
	ImageName    string
	Flags        []string
	Command      []string
	EnvFile      string // path to .env file; forwarded as --env-file to docker
}

// dockerErrNoSuchContainer is the sentinel substring Docker embeds in stderr
// when `docker rm` targets a container that does not exist. Docker does not
// use a distinct exit code for this case, so string matching is the only
// reliable approach.
const dockerErrNoSuchContainer = "No such container"

type ContainerManager struct {
	name   string
	cfg    *config.Config
	runner runner.CommandRunner
	labels map[string]string
}

// NewContainerManager initializes the manager. If r is nil, it defaults to RealRunner.
func NewContainerManager(name string, cfg *config.Config, r runner.CommandRunner) *ContainerManager {
	if r == nil {
		r = &runner.RealRunner{}
	}
	return &ContainerManager{
		name:   name,
		cfg:    cfg,
		runner: r,
		labels: map[string]string{},
	}
}

// Start spins up the environment. ImageName and Command are optional for Compose.
func (c *ContainerManager) Start(opts StartOptions) error {
	if c.cfg == nil {
		c.cfg = &config.Config{}
	}
	safeName, err := utils.Sanitize(c.name)
	if err != nil {
		return err
	}
	safeWorktree, err := utils.SanitizePath(opts.WorktreePath)
	if err != nil {
		return err
	}

	if opts.ImageName != "" {
		c.labels["com.nekotree.worktree.path"] = safeWorktree

		// Build volume flags via MountManager so DEVENV_MOUNTS is honoured.
		mm := volumes.NewMountManager(safeWorktree)
		if err := mm.LoadFromEnv(); err != nil {
			return fmt.Errorf("failed to load mount config: %w", err)
		}
		if err := mm.Validate(); err != nil {
			return fmt.Errorf("invalid mount: %w", err)
		}

		// Construct: docker run [base_flags] [volume_flags] [user_flags] [image] [command]
		args := []string{"run", "-d", "--name", safeName}
		args = append(args, mm.GetDockerFlags()...)

		if len(c.labels) > 0 {
			// Collect keys to ensure deterministic ordering
			keys := make([]string, 0, len(c.labels))
			for k := range c.labels {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			for _, k := range keys {
				args = append(args, "--label", fmt.Sprintf("%s=%s", k, c.labels[k]))
			}
		}

		// Add env file if specified
		if opts.EnvFile != "" {
			safeEnv, err := utils.SanitizePath(opts.EnvFile)
			if err != nil {
				return fmt.Errorf("invalid env file path: %w", err)
			}
			args = append(args, "--env-file", safeEnv)
		}

		// Add user flags (e.g., -p, -e) - strip quotes from flags
		flags := parseFlags(opts.Flags)
		args = append(args, flags...)

		// Add the Image
		args = append(args, opts.ImageName)

		// If no command given, use tail -f /dev/null to keep the container alive
		// indefinitely. Works on any POSIX image without a sleep binary or timeout.
		command := opts.Command
		if len(command) == 0 {
			command = []string{"tail", "-f", "/dev/null"}
		}
		args = append(args, command...)

		out, err := c.runner.CombinedOutput("docker", args...)
		if err != nil {
			return fmt.Errorf("docker run failed: %s: %w", string(out), err)
		}
		return nil
	}

	// Compose Logic
	args := []string{"compose"}
	if opts.EnvFile != "" {
		safeEnv, err := utils.SanitizePath(opts.EnvFile)
		if err != nil {
			return fmt.Errorf("invalid env file path: %w", err)
		}
		args = append(args, "--env-file", safeEnv)
	}
	args = append(args, "-f", c.cfg.ComposeFile, "-p", safeName, "up", "-d")
	out, err := c.runner.CombinedOutput("docker", args...)
	if err != nil {
		return fmt.Errorf("docker-compose failed: %s: %w", string(out), err)
	}
	return nil
}

// Stop cleans up the docker resources (works for both compose and standalone)
func (c *ContainerManager) Stop() error {
	if err := c.runner.Run("docker", "stop", c.name); err != nil {
		// Log but don't fail — container may already be stopped
		fmt.Fprintf(os.Stderr, "warning: docker stop %s: %v\n", c.name, err)
	}

	out, err := c.runner.CombinedOutput("docker", "rm", "-v", c.name)
	if err != nil {
		if strings.Contains(string(out), dockerErrNoSuchContainer) {
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

	// Pre-flight: verify at least sh exists before allocating a TTY.
	// #nosec G204 - safeName validated by utils.Sanitize
	out, err := exec.Command("docker", "exec", safeName, "sh", "-c", "exit 0").CombinedOutput()
	if err != nil {
		return fmt.Errorf("container %s has no usable shell (sh not found): %s", safeName, strings.TrimSpace(string(out)))
	}

	shellCmd := "command -v bash >/dev/null && bash || sh"

	// Note: Interactive shells (-it) MUST use os/exec directly as they need TTY control.
	// Mocks cannot simulate a terminal interaction easily.
	cmd := exec.Command("docker", "exec", "-it", safeName, "sh", "-c", shellCmd) // #nosec G204 - safeName validated by utils.Sanitize; shellCmd is a literal
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr

	return cmd.Run()
}

// List scans for all running containers managed by nekotree
func (c *ContainerManager) List(w io.Writer) error {
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
		_, _ = fmt.Fprintln(w, "🌳 No active nekotree environments found.")
		return nil
	}

	_, _ = fmt.Fprint(w, output)
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

// RunCommand executes a non-interactive command in the container, writing output to w.
func (c *ContainerManager) RunCommand(w io.Writer, cmd string) error {
	safeName, err := utils.Sanitize(c.name)
	if err != nil {
		return err
	}

	// Ensure container is running
	if !c.Exists() {
		return fmt.Errorf("container not found: %s", c.name)
	}

	// Execute command
	out, err := c.runner.CombinedOutput("docker", "exec", safeName, "sh", "-c", cmd)
	if len(out) > 0 {
		_, _ = fmt.Fprint(w, string(out))
	}
	return err
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
