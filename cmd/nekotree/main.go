package main

import (
    "fmt"
    "log"
    "os"
    "strings"

    "github.com/urfave/cli/v2"

    "cubicheart.com/munchtoast/nekotree/internal/config"
    "cubicheart.com/munchtoast/nekotree/internal/docker"
    "cubicheart.com/munchtoast/nekotree/internal/gitworktree"
    "cubicheart.com/munchtoast/nekotree/internal/volumes"
)

func main() {
    app := &cli.App{
        Name:  "nekotree",
        Usage: "Development environment manager with Git worktrees and Docker",
        Version: "1.0.0",
        Commands: []*cli.Command{
            createCommand(),
            startCommand(),
            stopCommand(),
            statusCommand(),
            execCommand(),
            removeCommand(),
        },
    }

    if err := app.Run(os.Args); err != nil {
        log.Fatal(err)
    }
}

func createCommand() *cli.Command {
    return &cli.Command{
        Name:  "create",
        Usage: "Create a new Git worktree and start Docker container",
        Flags: []cli.Flag{
            &cli.StringFlag{Name: "branch", Aliases: []string{"b"}, Required: true,
                Usage: "Feature branch name to create"},
            &cli.StringFlag{Name: "name", Aliases: []string{"n"}, Value: "nekotree",
                Usage: "Docker container name prefix"},
            &cli.StringFlag{Name: "image", Aliases: []string{"i"}, Required: true,
                Usage: "Base Docker image for development"},
        },
        Action: func(c *cli.Context) error {
            branch := c.String("branch")
            name := c.String("name")
            image := c.String("image")

            cfg, err := config.Load()
            if err != nil {
                return fmt.Errorf("failed to load config: %w", err)
            }

            worktreeMgr := gitworktree.NewWorktreeManager(cfg.WorktreeRoot)
            mountMgr := volumes.NewMountManager(worktreeMgr.GetBasePath())

            // Load additional mounts from env vars
            if err := mountMgr.LoadFromEnv(); err != nil {
                return fmt.Errorf("failed to load additional mounts: %w", err)
            }

            containerMgr := docker.NewContainerManager(name, image)

            // Create worktree
            err = worktreeMgr.CreateWorktree(branch)
            if err != nil {
                return fmt.Errorf("failed to create worktree: %w", err)
            }

            // Start container with mounted volumes
            if err = containerMgr.Start(cfg.WorktreeRoot); err != nil {
                return fmt.Errorf("failed to start container: %w", err)
            }

            fmt.Printf("✅ Worktree created at: %s\n", cfg.WorktreeRoot)
            fmt.Printf("🐳 Container started: %s (image: %s)\n", name, image)
            return nil
        },
    }
}

func startCommand() *cli.Command {
    return &cli.Command{
        Name:  "start",
        Usage: "Start an existing development container",
        Flags: []cli.Flag{
            &cli.StringFlag{Name: "name", Aliases: []string{"n"}, Value: "nekotree",
                Usage: "Docker container name prefix"},
        },
        Action: func(c *cli.Context) error {
            name := c.String("name")
            cfg, _ := config.Load()

            mgr := docker.NewContainerManager(name, cfg.BaseImage)
            if err := mgr.Start(cfg.WorktreeRoot); err != nil {
                return fmt.Errorf("failed to start container: %w", err)
            }
            return nil
        },
    }
}

func stopCommand() *cli.Command {
    return &cli.Command{
        Name:  "stop",
        Usage: "Stop the development container",
        Flags: []cli.Flag{
            &cli.StringFlag{Name: "name", Aliases: []string{"n"}, Value: "nekotree",
                Usage: "Docker container name prefix"},
        },
        Action: func(c *cli.Context) error {
            name := c.String("name")
            mgr := docker.NewContainerManager(name, "")

            if err := mgr.Stop(); err != nil {
                return fmt.Errorf("failed to stop container: %w", err)
            }
            fmt.Printf("🛑 Container stopped: %s\n", name)
            return nil
        },
    }
}

func statusCommand() *cli.Command {
    return &cli.Command{
        Name:  "status",
        Usage: "Show container and worktree status",
        Flags: []cli.Flag{
            &cli.StringFlag{Name: "name", Aliases: []string{"n"}, Value: "nekotree",
                Usage: "Docker container name prefix"},
        },
        Action: func(c *cli.Context) error {
            name := c.String("name")
            mgr := docker.NewContainerManager(name, "")

            if err := mgr.Status(); err != nil {
                return fmt.Errorf("container is not running: %w", err)
            }
            return nil
        },
    }
}

func execCommand() *cli.Command {
    return &cli.Command{
        Name:  "exec",
        Usage: "Run a command inside the development container",
        // ... flags ...
        Action: func(c *cli.Context) error {
            name := c.String("name")
            command := strings.Join(c.Args().Slice(), " ")
            if command == "" {
                command = "bash" // Default to interactive shell
            }

            mgr := docker.NewContainerManager(name, "")
            // Use the restored method name
            return mgr.ExecCommand(command)
        },
    }
}

func removeCommand() *cli.Command {
    return &cli.Command{
        Name:  "remove",
        Usage: "Remove the worktree and optionally clean up resources",
        Flags: []cli.Flag{
            &cli.StringFlag{Name: "name", Aliases: []string{"n"}, Value: "nekotree",
                Usage: "Docker container name prefix"},
            &cli.BoolFlag{Name: "force", Aliases: []string{"f"},
                Usage: "Force remove without confirmation"},
        },
        Action: func(c *cli.Context) error {
            name := c.String("name")
            force := c.Bool("force")

            if !force && !confirmDelete(name) {
                return fmt.Errorf("cancelled by user")
            }

            mgr := docker.NewContainerManager(name, "")
            if err := mgr.Stop(); err != nil {
                log.Println("Note: container may already be stopped")
            }

            worktreeMgr := gitworktree.NewWorktreeManager("")
            // List all worktrees
            worktrees, err := worktreeMgr.ListWorktrees()
            if err != nil {
                return fmt.Errorf("failed to list worktrees: %w", err)
            }

            // Find the associated worktree
            var worktreePath string
            for _, wt := range worktrees {
                if strings.Contains(wt, name) {
                    worktreePath = wt
                    break
                }
            }

            if worktreePath == "" {
                return fmt.Errorf("worktree associated with container %s not found", name)
            }

            // Remove the worktree
            err = worktreeMgr.RemoveWorktree(worktreePath)
            if err != nil {
                return fmt.Errorf("failed to remove worktree: %w", err)
            }

            fmt.Printf("🗑️   Removed resources for container: %s\n", name)
            return nil
        },
    }
}

func confirmDelete(name string) bool {
    var response string
    fmt.Printf("⚠️  This will stop and remove the container '%s'. Continue? (y/N): ", name)
    _, _ = fmt.Scan(&response)
    return strings.ToLower(response) == "y" || response == "Y"
}

func isKnownCommand(name string) bool {
    for _, cmd := range []*cli.Command{
        createCommand(), startCommand(), stopCommand(), statusCommand(), execCommand(), removeCommand(),
    } {
        if cmd.Name == name {
            return true
        }
    }
    return false
}

func printUsage() {
    fmt.Println("Usage: nekotree <command> [options]")
    fmt.Println("\nCommands:")
    for _, cmd := range []*cli.Command{
        createCommand(), startCommand(), stopCommand(), statusCommand(), execCommand(), removeCommand(),
    } {
        fmt.Printf("  %-10s %s\n", cmd.Name, cmd.Usage)
    }
}

