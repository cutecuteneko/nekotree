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
	"cubicheart.com/munchtoast/nekotree/internal/utils"
	"github.com/urfave/cli/v2"
)

const (
	defaultConfigFile = "nekotree-config.json"
)

func main() {
	app := &cli.App{
		Name:    "nekotree",
		Usage:   "On-demand development environments with Git Worktrees",
		Version: "0.1.0",
		Commands: []*cli.Command{
			createCmd(),
			runCmd(),
			shellCmd(),
			listCmd(),
			removeCmd(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func runCmd() *cli.Command {
	return &cli.Command{
		Name:  "run",
		Usage: "Run: nekotree run <branch> <command>",
		ArgsUsage: "<branch> <command>",
		Action: func(c *cli.Context) error {
			branch := c.Args().Get(0)
			if branch == "" {
				return fmt.Errorf("branch required")
			}

			cmd := strings.Join(c.Args().Tail(), " ")

			cwd, err := os.Getwd()
			if err != nil {
				return err
			}

			safeBranch, err := utils.Sanitize(branch)
			if err != nil {
				return err
			}

			name := fmt.Sprintf("nekotree-%s-%s", filepath.Base(cwd), safeBranch)
			cfg, _ := config.Load(defaultConfigFile)
			cm := docker.NewContainerManager(name, cfg, nil)

			// Start container if needed
			if !cm.Exists() {
				// Default image and flags
				targetPath := filepath.Join(cwd, "build/worktrees", safeBranch)
				w := gitworktree.NewWorktreeManager(cwd, nil)
				if w.Exists(safeBranch) {
          err = cm.Start(targetPath, "alpine:latest", nil, nil)
          if err != nil {
            return err
          }
				} else {
					return fmt.Errorf("worktree not found for branch: %s", safeBranch)
				}
			}

			return cm.RunCommand(cmd)
		},
	}
}

func createCmd() *cli.Command {
	return &cli.Command{
		Name:      "create",
		Aliases:   []string{"c"},
		Usage:     "Create: nekotree create <branch> [image|compose] [command] [-f flag]",
		ArgsUsage: "<branch> [image|compose] [command]",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:    "flag",
				Aliases: []string{"f"},
				Usage:   "Raw docker flags (e.g. -f \"-p 8080:8080\")",
			},
		},
		Action: func(c *cli.Context) error {
			branch := c.Args().First()
			if branch == "" {
				return fmt.Errorf("branch name required")
			}

			safeBranch, err := utils.Sanitize(branch)
			if err != nil {
				return err
			}

			cfg, _ := config.Load(defaultConfigFile)
			if cfg == nil {
				cfg = &config.Config{}
			}

			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			repoName := filepath.Base(cwd)
			uniqueName := fmt.Sprintf("nekotree-%s-%s", repoName, safeBranch)
			targetPath := filepath.Join(cwd, uniqueName)

			// Parse remaining arguments: env-spec (image or compose) and command
			envSpec := c.Args().Get(1)
			command := strings.Join(c.Args().Slice()[2:], " ")

			// Determine if env-spec is a file (Compose) or an image string
			imageName := ""
			if envSpec != "" {
				if info, err := os.Stat(envSpec); err == nil && !info.IsDir() {
					// It's a file path -> Compose file
					cfg.ComposeFile = envSpec
				} else {
					// It's a string -> Image name
					imageName = envSpec
				}
			}

			wm := gitworktree.NewWorktreeManager(cwd, nil)
			if err := wm.CreateWorktree(safeBranch); err != nil {
				return err
			}

			cm := docker.NewContainerManager(uniqueName, cfg, nil)

			extraFlags := c.StringSlice("flag")
			var flattenedFlags []string
			for _, f := range extraFlags {
				flattenedFlags = append(flattenedFlags, strings.Fields(f)...)
			}

			// If no command provided and no compose file, default to sleep to keep container alive
			var containerCommand []string
			if command == "" && cfg.ComposeFile == "" {
				containerCommand = []string{"sleep", "3600"}
			} else if command != "" {
				containerCommand = splitCommand(command)
			}

			fmt.Printf("🐳 Launching environment: %s\n", uniqueName)
			// Pass: worktreePath, imageName, flags, command
			return cm.Start(targetPath, imageName, flattenedFlags, containerCommand)
		},
	}
}

// splitCommand splits a command string into separate arguments
func splitCommand(cmd string) []string {
	if cmd == "" {
		return nil
	}
	// Use shell-like parsing
	quoted := false
	var result []string
	var current strings.Builder
	for i := 0; i < len(cmd); i++ {
		c := cmd[i]
		if c == '"' {
			quoted = !quoted
		} else if c == ' ' && !quoted {
			if current.Len() > 0 {
				result = append(result, current.String())
				current.Reset()
			}
		} else {
			current.WriteByte(c)
		}
	}
	if current.Len() > 0 {
		result = append(result, current.String())
	}
	return result
}

func listCmd() *cli.Command {
	return &cli.Command{
		Name:    "list",
		Aliases: []string{"ls"},
		Usage:   "List all active nekotree environments",
		Action: func(c *cli.Context) error {
			cfg, _ := config.Load(defaultConfigFile)
			// Pass nil to use the real Docker runner
			cm := docker.NewContainerManager("", cfg, nil)
			return cm.List()
		},
	}
}

func shellCmd() *cli.Command {
	return &cli.Command{
		Name:      "shell",
		Aliases:   []string{"sh", "s"},
		Usage:     "Enter: nekotree shell <branch>",
		ArgsUsage: "<branch>",
		Action: func(c *cli.Context) error {
			branch := c.Args().First()
			if branch == "" {
				return fmt.Errorf("branch required")
			}

			safeBranch, err := utils.Sanitize(branch)
			if err != nil {
				return err
			}

			cwd, err := os.Getwd()
			if err != nil {
				return err
			}

			name := fmt.Sprintf("nekotree-%s-%s", filepath.Base(cwd), safeBranch)
			cfg, _ := config.Load(defaultConfigFile)
			cm := docker.NewContainerManager(name, cfg, nil)

			return cm.Shell()
		},
	}
}

func removeCmd() *cli.Command {
	return &cli.Command{
		Name:    "remove",
		Aliases: []string{"rm"},
		Usage:   "Remove: nekotree remove <name-or-branch>",
		Action: func(c *cli.Context) error {
			input := c.Args().First()
			if input == "" {
				return fmt.Errorf("name or branch required")
			}

			cfg, _ := config.Load(defaultConfigFile)
			cwd, _ := os.Getwd()
			repoName := filepath.Base(cwd)
			prefix := fmt.Sprintf("nekotree-%s-", repoName)

			var targetName string
			if strings.HasPrefix(input, prefix) {
				targetName = input
			} else {
				targetName = prefix + input
			}

			cm := docker.NewContainerManager(targetName, cfg, nil)
			wm := gitworktree.NewWorktreeManager(cwd, nil)

			containerExists := cm.Exists()
			worktreeExists := wm.Exists(input)

			if !containerExists && !worktreeExists {
				fmt.Printf("ℹ️  No environment found for '%s'. Nothing to do.\n", input)
				return nil
			}

			fmt.Printf("🗑️  Cleaning up environment: %s\n", targetName)

			if err := cm.Stop(); err != nil {
				fmt.Printf("⚠️  Warning: Docker cleanup had issues: %v\n", err)
			}

			if err := wm.RemoveWorktree(targetName); err != nil {
				fmt.Printf("⚠️  Warning: Worktree cleanup had issues: %v\n", err)
			}

			return nil
		},
	}
}
