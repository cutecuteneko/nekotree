# 🌳 Nekotree

![Coverage](https://img.shields.io/badge/coverage-53.5%25-green?style=flat-square)
![Security](https://img.shields.io/badge/security-check--passed-blue?style=flat-square)

**On-demand development environments with Git Worktrees.**

Nekotree is a CLI tool designed to streamline the management of Git worktrees and their associated Docker-based development environments. It allows developers to quickly context-switch between branches without polluting their main working directory, while automatically spinning up containerized environments tailored to that specific branch.

## Key Features

* **Isolated Worktrees**: Automatically creates and manages Git worktrees using a naming convention: `nekotree-<repo>-<branch>`.
* **Container Orchestration**: Launches Docker containers (or Compose stacks) linked to the worktree filesystem.
* **Host Socket Passthrough**: Operates from within a manager container while scheduling environments directly on the host Docker daemon.
* **Persistent Mounts**: Supports global and project-specific volume mounts via `DEVENV_MOUNTS`.

## Claude Agent Integration

Nekotree ships with an **external skill** (`docs/nekotree-skill.md`) that can be loaded by a Claude agent operating in *any* repository — not just this one. The skill teaches the agent the full nekotree CLI workflow: creating environments, running commands inside them, and cleaning up.

**Requirements to use the external skill:**

- The `nekotree` binary is installed and on `$PATH`, **or**
- The nekotree repository has been cloned and built locally (`make build`), with the binary referenced by its full path (`./build/nekotree`)

No other setup is needed in the target repository. The skill is self-contained and includes prerequisites, CLI reference, and worked examples.

## Quick Start

### Installation
Ensure you have the `nekotree` binary in your path (or use the Docker-based manager).

```bash
# Create a new environment for a feature branch
nekotree create feature-login alpine:latest -f "-p 8080:80"

# Enter the environment
nekotree shell feature-login

# List active environments
nekotree list

# Cleanup
nekotree remove feature-login
```
