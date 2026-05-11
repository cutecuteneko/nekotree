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
	"cubicheart.com/munchtoast/nekotree/internal/runner"
	"cubicheart.com/munchtoast/nekotree/internal/utils"
	"github.com/urfave/cli/v2"
)

const (
	defaultConfigFile = "nekotree-config.json"
)

// loadConfig loads the nekotree config file. If the file is absent it returns
// an empty config (not an error). Real errors — malformed JSON, security
// violations — are returned so callers can fail fast rather than silently
// continuing with a wrong configuration.
func loadConfig() (*config.Config, error) {
	cfg, err := config.Load(defaultConfigFile)
	if err != nil {
		return nil, fmt.Errorf("could not load config: %w", err)
	}
	return cfg, nil
}

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
		Name:      "run",
		Aliases:   []string{"r"},
		Usage:     "Run: nekotree run <branch> <command>",
		ArgsUsage: "<branch> <command>",
		Action:    func(c *cli.Context) error { return runRunAction(c, nil) },
	}
}

func runRunAction(c *cli.Context, r runner.CommandRunner) error {
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

	repoName := filepath.Base(cwd)
	name := utils.BuildName(repoName, safeBranch)
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	cm := docker.NewContainerManager(name, cfg, r)

	if !cm.Exists() {
		// Worktree path matches what CreateWorktree() creates: <cwd>/nekotree-<repo>-<branch>
		w := gitworktree.NewWorktreeManager(cwd, r)
		if !w.Exists(safeBranch) {
			return fmt.Errorf("worktree not found for branch: %s — use 'nekotree create' first", safeBranch)
		}
		return fmt.Errorf("container for branch %s does not exist — use 'nekotree create' to recreate it", safeBranch)
	}

	return cm.RunCommand(os.Stdout, cmd)
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
			&cli.StringFlag{
				Name:    "env",
				Aliases: []string{"e"},
				Usage:   "Path to .env file (defaults to <compose-dir>/.env when using a compose file)",
			},
		},
		Action: func(c *cli.Context) error { return runCreateAction(c, nil) },
	}
}

func runCreateAction(c *cli.Context, r runner.CommandRunner) error {
	branch := c.Args().First()
	if branch == "" {
		return fmt.Errorf("branch name required")
	}

	safeBranch, err := utils.Sanitize(branch)
	if err != nil {
		return err
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	repoName := filepath.Base(cwd)
	uniqueName := utils.BuildName(repoName, safeBranch)
	targetPath := filepath.Join(cwd, uniqueName)

	// urfave/cli v2 stops flag parsing once positional args begin, so -f/--flag
	// tokens that appear after the image argument are silently treated as
	// positional args. We scan c.Args() manually to recover them.
	var positionalArgs []string
	var extraFlagValues []string
	var extraEnvValue string
	{
		raw := c.Args().Slice()
		for i := 0; i < len(raw); i++ {
			arg := raw[i]
			if arg == "-f" || arg == "--flag" {
				if i+1 < len(raw) {
					extraFlagValues = append(extraFlagValues, raw[i+1])
					i++ // skip the value
				}
			} else if strings.HasPrefix(arg, "-f=") {
				extraFlagValues = append(extraFlagValues, strings.TrimPrefix(arg, "-f="))
			} else if strings.HasPrefix(arg, "--flag=") {
				extraFlagValues = append(extraFlagValues, strings.TrimPrefix(arg, "--flag="))
			} else if arg == "-e" || arg == "--env" {
				if i+1 < len(raw) {
					extraEnvValue = raw[i+1]
					i++ // skip the value
				}
			} else if strings.HasPrefix(arg, "-e=") {
				extraEnvValue = strings.TrimPrefix(arg, "-e=")
			} else if strings.HasPrefix(arg, "--env=") {
				extraEnvValue = strings.TrimPrefix(arg, "--env=")
			} else {
				positionalArgs = append(positionalArgs, arg)
			}
		}
	}

	// Merge flags caught by urfave/cli (placed before positional args) with
	// those caught by the manual scan above (placed after positional args).
	allFlagValues := append(c.StringSlice("flag"), extraFlagValues...)
	var flattenedFlags []string
	for _, f := range allFlagValues {
		flattenedFlags = append(flattenedFlags, strings.Fields(f)...)
	}

	// positionalArgs[0] is the branch, [1] is envSpec, [2:] is command.
	var envSpec string
	var command string
	if len(positionalArgs) > 1 {
		envSpec = positionalArgs[1]
	}
	if len(positionalArgs) > 2 {
		command = strings.Join(positionalArgs[2:], " ")
	}

	imageName := ""
	if envSpec != "" {
		if info, err := os.Stat(envSpec); err == nil && !info.IsDir() {
			cfg.ComposeFile = envSpec
		} else {
			imageName = envSpec
		}
	}

	// Resolve .env file: explicit flag takes priority; extraEnvValue recovers
	// -e/--env tokens placed after positional args (urfave/cli v2 limitation).
	envFile := c.String("env")
	if envFile == "" {
		envFile = extraEnvValue
	}
	if envFile == "" && cfg.ComposeFile != "" {
		candidate := filepath.Join(filepath.Dir(cfg.ComposeFile), ".env")
		if _, err := os.Stat(candidate); err == nil {
			envFile = candidate
		}
	}
	if envFile != "" {
		safeEnvFile, err := utils.SanitizePath(envFile)
		if err != nil {
			return fmt.Errorf("invalid env file path: %w", err)
		}
		envFile = safeEnvFile
	}

	wm := gitworktree.NewWorktreeManager(cwd, r)
	if err := wm.CreateWorktree(safeBranch); err != nil {
		return err
	}

	cm := docker.NewContainerManager(uniqueName, cfg, r)

	var containerCommand []string
	if command == "" && cfg.ComposeFile == "" {
		containerCommand = []string{"tail", "-f", "/dev/null"}
	} else if command != "" {
		containerCommand = splitCommand(command)
	}

	fmt.Printf("🐳 Launching environment: %s\n", uniqueName)
	return cm.Start(docker.StartOptions{
		WorktreePath: targetPath,
		ImageName:    imageName,
		Flags:        flattenedFlags,
		Command:      containerCommand,
		EnvFile:      envFile,
	})
}

// splitCommand splits a command string into separate arguments
func splitCommand(cmd string) []string {
	if cmd == "" {
		return nil
	}
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
		Action:  func(c *cli.Context) error { return runListAction(c, nil) },
	}
}

func runListAction(c *cli.Context, r runner.CommandRunner) error {
	// List doesn't use config (no compose file needed for listing containers)
	cm := docker.NewContainerManager("", &config.Config{}, r)
	return cm.List(os.Stdout)
}

func shellCmd() *cli.Command {
	return &cli.Command{
		Name:      "shell",
		Aliases:   []string{"sh", "s"},
		Usage:     "Enter: nekotree shell <branch>",
		ArgsUsage: "<branch>",
		Action:    func(c *cli.Context) error { return runShellAction(c, nil) },
	}
}

func runShellAction(c *cli.Context, r runner.CommandRunner) error {
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

	name := utils.BuildName(filepath.Base(cwd), safeBranch)
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	cm := docker.NewContainerManager(name, cfg, r)

	return cm.Shell()
}

func removeCmd() *cli.Command {
	return &cli.Command{
		Name:    "remove",
		Aliases: []string{"rm"},
		Usage:   "Remove: nekotree remove <name-or-branch>",
		Action:  func(c *cli.Context) error { return runRemoveAction(c, nil) },
	}
}

func runRemoveAction(c *cli.Context, r runner.CommandRunner) error {
	input := c.Args().First()
	if input == "" {
		return fmt.Errorf("name or branch required")
	}

	safeInput, err := utils.Sanitize(input)
	if err != nil {
		return fmt.Errorf("invalid name: %w", err)
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not determine working directory: %w", err)
	}
	repoName := filepath.Base(cwd)
	prefix := fmt.Sprintf("nekotree-%s-", repoName)

	var targetName string
	if strings.HasPrefix(safeInput, prefix) {
		targetName = safeInput
	} else {
		targetName = prefix + safeInput
	}

	cm := docker.NewContainerManager(targetName, cfg, r)
	wm := gitworktree.NewWorktreeManager(cwd, r)

	containerExists := cm.Exists()
	worktreeExists := wm.Exists(safeInput)

	if !containerExists && !worktreeExists {
		fmt.Printf("ℹ️  No environment found for '%s'. Nothing to do.\n", safeInput)
		return nil
	}

	fmt.Printf("🗑️  Cleaning up environment: %s\n", targetName)

	if err := cm.Stop(); err != nil {
		fmt.Printf("⚠️  Warning: Docker cleanup had issues: %v\n", err)
	}

	if err := wm.RemoveWorktree(filepath.Join(cwd, targetName)); err != nil {
		fmt.Printf("⚠️  Warning: Worktree cleanup had issues: %v\n", err)
	}

	return nil
}
