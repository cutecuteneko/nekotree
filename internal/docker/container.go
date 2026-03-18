package docker

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"cubicheart.com/munchtoast/nekotree/internal/volumes"
)

// Commander defines the interface for running shell commands
type Commander interface {
	Run(name string, arg ...string) error
	Output(name string, arg ...string) ([]byte, error)
}

// RealCommander is the production implementation
type RealCommander struct{}

func (r *RealCommander) Run(name string, arg ...string) error {
	cmd := exec.Command(name, arg...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (r *RealCommander) Output(name string, arg ...string) ([]byte, error) {
	return exec.Command(name, arg...).Output()
}

type ContainerManager struct {
	Name   string
	Image  string
	Mounts *volumes.MountManager
	Exec   Commander // Injected dependency
}

func NewContainerManager(name, image string, mounts ...volumes.Mount) *ContainerManager {
	return &ContainerManager{
		Name:   name,
		Image:  image,
		Mounts: volumes.NewMountManager("", mounts...),
		Exec:   &RealCommander{},
	}
}

func (c *ContainerManager) Start(worktreePath string) error {
	c.Mounts.WorktreeRoot = worktreePath
	if err := c.Mounts.Validate(); err != nil {
		return fmt.Errorf("volume validation failed: %w", err)
	}

	flags := c.Mounts.GetDockerFlags()
	
	// Construct arguments for 'docker run'
	args := []string{"run", "--rm", "-it", "--name", c.Name}
	args = append(args, flags...)
	args = append(args, c.Image, "bash")

	return c.Exec.Run("docker", args...)
}

func (c *ContainerManager) Stop() error {
	// We use Run instead of exec.Command directly
	return c.Exec.Run("docker", "stop", c.Name)
}

// RESTORED: Exec runs a command inside the running container
func (c *ContainerManager) ExecCommand(command string) error {
	return c.Exec.Run("docker", "exec", "-it", c.Name, "bash", "-c", command)
}

// RESTORED: Status returns container status information
func (c *ContainerManager) Status() error {
	out, err := c.Exec.Output("docker", "inspect", "--format", "{{.State.Status}}", c.Name)
	if err != nil {
		return fmt.Errorf("container is not running or does not exist: %w", err)
	}

	fmt.Printf("Container '%s' status: %s\n", c.Name, strings.TrimSpace(string(out)))
	return nil
}
