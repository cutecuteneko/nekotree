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

const defaultConfigFile = "nekotree-config.json"

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

func createCmd() *cli.Command {
	return &cli.Command{
		Name:      "create",
		Aliases:   []string{"c"},
		Usage:     "Create: nekotree create <branch> [env-spec] [-f flag]",
		ArgsUsage: "<branch> [env-spec]",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:    "flag",
				Aliases: []string{"f"},
				Usage:   "Raw docker flags (e.g. -f \"-v /tmp:/tmp\")",
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

			// Determine if env-spec is a file (Compose) or an image string
			envSpec := c.Args().Get(1)
			imageName := ""
			if envSpec != "" {
				if info, err := os.Stat(envSpec); err == nil && !info.IsDir() {
					cfg.ComposeFile = envSpec
				} else {
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

			fmt.Printf("🐳 Launching environment: %s\n", uniqueName)
			// Pass: worktreePath, imageName, flags, command (nil)
			return cm.Start(targetPath, imageName, flattenedFlags, nil)
		},
	}
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
