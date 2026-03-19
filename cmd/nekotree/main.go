package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

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
		Usage:     "Create: nekotree create <branch> [env-spec]",
		ArgsUsage: "<branch> [env-spec]",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:    "flag",
				Aliases: []string{"f"},
				Usage:   "Raw docker flags",
			},
		},
		Action: func(c *cli.Context) error {
			branch := c.Args().First()
			if branch == "" {
				return fmt.Errorf("branch name required")
			}

			// Validate branch early
			safeBranch, err := utils.Sanitize(branch)
			if err != nil {
				return err
			}

			cfg, err := config.Load(defaultConfigFile)
			if err != nil {
				// Fallback to empty config if file is missing
				cfg = &config.Config{}
			}

			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			repoName := filepath.Base(cwd)

			uniqueName := fmt.Sprintf("nekotree-%s-%s", repoName, safeBranch)
			targetPath := filepath.Join(cwd, uniqueName)

			envSpec := c.Args().Get(1)
			if envSpec != "" {
				if info, err := os.Stat(envSpec); err == nil && !info.IsDir() {
					cfg.ComposeFile = envSpec
				}
			}

			// 1. Git Logic
			wm := gitworktree.NewWorktreeManager(cwd, nil)
			if err := wm.CreateWorktree(safeBranch); err != nil {
				return err
			}

			// 2. Docker Logic - Passing nil for the runner defaults to RealRunner
			cm := docker.NewContainerManager(uniqueName, cfg, nil)

			fmt.Printf("🐳 Launching environment: %s\n", uniqueName)
			return cm.Start(targetPath)
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
		Name:      "remove",
		Aliases:   []string{"rm"},
		Usage:     "Remove: nekotree remove <branch>",
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
			targetPath := filepath.Join(cwd, name)

			cfg, _ := config.Load(defaultConfigFile)
			cm := docker.NewContainerManager(name, cfg, nil)

			if err := cm.Stop(); err != nil {
				fmt.Printf("⚠️  Container cleanup failed: %v\n", err)
			}

			wm := gitworktree.NewWorktreeManager(cwd, nil)
			return wm.RemoveWorktree(targetPath)
		},
	}
}
