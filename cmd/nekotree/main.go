package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"cubicheart.com/munchtoast/nekotree/internal/config"
	"cubicheart.com/munchtoast/nekotree/internal/docker"
	"cubicheart.com/munchtoast/nekotree/internal/gitworktree"
	"cubicheart.com/munchtoast/nekotree/internal/volumes"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:    "nekotree",
		Usage:   "On-demand development environments with Git Worktrees",
		Version: "0.1.0",
		Commands: []*cli.Command{
			createCmd(),
			shellCmd(),
			listCmd(),
			removeCmd(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

// createCmd handles the creation of a new worktree and container.
// It is idempotent: if the folder or branch exists, it skips the Git step.
func createCmd() *cli.Command {
	return &cli.Command{
		Name:    "create",
		Aliases: []string{"c"},
		Usage:   "Create a new worktree and start its container/stack",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "branch", Aliases: []string{"b"}, Required: true},
			&cli.StringFlag{Name: "image", Aliases: []string{"i"}},
			&cli.StringFlag{Name: "compose", Aliases: []string{"yml"}},
		},
		Action: func(c *cli.Context) error {
			cfg, _ := config.Load()
			branch := c.String("branch")
			name := "nekotree-" + branch

			image := c.String("image")
			if image == "" {
				image = cfg.DefaultImage
			}
			compose := c.String("compose")
			if compose == "" {
				compose = cfg.ComposeFile
			}

			// Use Current Working Directory as the project root
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current directory: %w", err)
			}

			// Target path is INSIDE the current project folder
			targetPath := filepath.Join(cwd, name)
			absTarget, _ := filepath.Abs(targetPath)

			// 1. Git Setup
			wm := gitworktree.NewWorktreeManager(cwd)
			fmt.Printf("🌱 Preparing Git worktree: %s\n", absTarget)
			if err := wm.CreateWorktree(branch); err != nil {
				return err
			}

			// 2. Docker Setup
			cm := docker.NewContainerManager(name, image, compose)
			cm.Mounts = volumes.NewMountManager(absTarget)
			cm.Mounts.LoadFromEnv()

			fmt.Printf("🐳 Launching environment: %s\n", name)
			return cm.Start(absTarget)
		},
	}
}

// shellCmd attaches to an existing nekotree container.
func shellCmd() *cli.Command {
	return &cli.Command{
		Name:    "shell",
		Aliases: []string{"sh", "s"},
		Usage:   "Enter the environment shell for a branch",
		Action: func(c *cli.Context) error {
			branch := c.Args().First()
			if branch == "" {
				return fmt.Errorf("branch name required")
			}

			name := branch
			if !strings.HasPrefix(name, "nekotree-") {
				name = "nekotree-" + name
			}

			cm := docker.NewContainerManager(name, "", "")
			return cm.ExecCommand()
		},
	}
}

// listCmd displays a status table with disk usage indicators.
func listCmd() *cli.Command {
	return &cli.Command{
		Name:    "list",
		Aliases: []string{"ls"},
		Usage:   "List all active nekotree environments and disk usage",
		Action: func(c *cli.Context) error {
			cwd, _ := os.Getwd()
			cm := docker.NewContainerManager("", "", "")
			return cm.List(cwd)
		},
	}
}

// removeCmd cleans up the container and the git worktree registration.
func removeCmd() *cli.Command {
	return &cli.Command{
		Name:    "remove",
		Aliases: []string{"rm"},
		Usage:   "Remove a worktree and stop its containers",
		Action: func(c *cli.Context) error {
			branch := c.Args().First()
			if branch == "" {
				return fmt.Errorf("branch name required")
			}

			name := branch
			if !strings.HasPrefix(name, "nekotree-") {
				name = "nekotree-" + name
			}

			cwd, _ := os.Getwd()
			targetPath := filepath.Join(cwd, name)

			// SAFETY: Don't delete the directory if we are currently sitting in it
			if strings.Contains(cwd, name) {
				return fmt.Errorf("❌ ERROR: You are currently inside %s. 'cd ..' before removing", name)
			}

			// 1. Docker Cleanup
			cm := docker.NewContainerManager(name, "", "")
			fmt.Printf("🗑️  Cleaning up container: %s\n", name)
			cm.Stop()

			// 2. Git Cleanup
			wm := gitworktree.NewWorktreeManager(cwd)
			fmt.Printf("🧹 Removing worktree metadata and files: %s\n", targetPath)
			if err := wm.RemoveWorktree(targetPath); err != nil {
				fmt.Println("⚠️  Git cleanup failed, forcing manual folder removal...")
				return os.RemoveAll(targetPath)
			}

			fmt.Println("✅ Environment removed.")
			return nil
		},
	}
}
