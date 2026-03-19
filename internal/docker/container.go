package docker

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"cubicheart.com/munchtoast/nekotree/internal/volumes"
)

type Commander interface {
	Run(name string, arg ...string) error
}

type RealCommander struct{}

func (r *RealCommander) Run(name string, arg ...string) error {
	cmd := exec.Command(name, arg...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}

type ContainerManager struct {
	Name        string
	Image       string
	ComposeFile string
	Mounts      *volumes.MountManager
	Exec        Commander
}

func NewContainerManager(name, image, composeFile string) *ContainerManager {
	return &ContainerManager{
		Name:        name,
		Image:       image,
		ComposeFile: composeFile,
		Mounts:      &volumes.MountManager{},
		Exec:        &RealCommander{},
	}
}

func (c *ContainerManager) Start(worktreePath string) error {
	if c.ComposeFile != "" {
		fmt.Printf("🚀 Starting Compose stack: %s\n", c.Name)
		cmd := exec.Command("docker", "compose", "-f", c.ComposeFile, "-p", c.Name, "up", "-d")
		cmd.Env = append(os.Environ(), fmt.Sprintf("WORKTREE_PATH=%s", worktreePath))
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		return cmd.Run()
	}

	c.Mounts.WorktreeRoot = worktreePath

	// args start with basic run commands
	args := []string{"run", "-d", "--name", c.Name}

	// Add all volume flags from the manager
	args = append(args, c.Mounts.GetDockerFlags()...)

	// Add the image and the keep-alive command
	args = append(args, c.Image, "tail", "-f", "/dev/null")

	return c.Exec.Run("docker", args...)
}

func (c *ContainerManager) Stop() error {
	fmt.Printf("🗑️  Cleaning up environment: %s\n", c.Name)
	exec.Command("docker", "compose", "-p", c.Name, "down").Run()
	return exec.Command("docker", "rm", "-f", c.Name).Run()
}

func (c *ContainerManager) ExecCommand() error {
	shellCmd := "command -v bash >/dev/null && bash || sh"
	var cmd *exec.Cmd
	// Standard exec into the container name
	cmd = exec.Command("docker", "exec", "-it", c.Name, "sh", "-c", shellCmd)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}

func (c *ContainerManager) List(worktreeRoot string) error {
	fmt.Printf("%-20s %-12s %-10s %-10s %-20s\n", "BRANCH", "STATUS", "DISK (WT)", "DISK (IMG)", "IMAGE")
	fmt.Println(strings.Repeat("-", 80))

	out, _ := exec.Command("docker", "ps", "-a", "--filter", "name=nekotree-", "--format", "{{.Names}}\t{{.Status}}\t{{.Image}}").Output()
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		name := parts[0]
		status := parts[1]
		image := parts[2]

		parentDir := filepath.Dir(worktreeRoot)
		wtPath := filepath.Join(parentDir, name)
		wtSize := getDirSize(wtPath)

		inspect, _ := exec.Command("docker", "ps", "-a", "--filter", "name="+name, "--format", "{{.Size}}").Output()
		contSize := strings.TrimSpace(string(inspect))

		branchName := strings.TrimPrefix(name, "nekotree-")
		fmt.Printf("%-20s %-12s %-10s %-10s %-20s\n", branchName, status, wtSize, contSize, image)
	}
	return nil
}

func getDirSize(path string) string {
	out, err := exec.Command("du", "-sh", path).Output()
	if err != nil {
		return "N/A"
	}
	fields := strings.Fields(string(out))
	if len(fields) > 0 {
		return fields[0]
	}
	return "0B"
}
