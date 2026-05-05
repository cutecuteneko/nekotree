# Nekotree

Nekotree creates isolated Git worktree + Docker container pairs for on-demand development environments. Spin up a fully containerised workspace for any branch with a single command, run commands inside it, and tear it down cleanly when done.

[![CI](https://github.com/cutecuteneko/nekotree/actions/workflows/build-docs-and-test.yml/badge.svg)](https://github.com/cutecuteneko/nekotree/actions/workflows/build-docs-and-test.yml)

## How It Works

Each environment is a pair:

- **Git worktree** — a separate checkout of a branch, living alongside your main repo
- **Docker container** — mounted to that worktree, named after the branch

```
myapp/
├── .git/
├── main.go                        # your main branch
└── nekotree-myapp-feature-login/  # worktree for feature-login
    └── main.go                    # isolated checkout
```

The container for `feature-login` mounts `nekotree-myapp-feature-login/` as `/workspace` and stays alive until you remove it.

## Quick Start

```bash
# Build the binary
make build   # → build/nekotree

# Create a worktree + container for a branch
nekotree create feature-login golang:latest

# Run a command inside it
nekotree run feature-login "cd /workspace && go test ./..."

# Open an interactive shell
nekotree shell feature-login

# List all active environments
nekotree list

# Remove the container and worktree
nekotree remove feature-login
```

## Installation

Requires Go 1.21+ and Docker.

```bash
git clone https://github.com/cutecuteneko/nekotree.git
cd nekotree
make build
# Add build/ to your PATH, or use the full path: ./build/nekotree
```

## CLI Reference

| Command | Aliases | Description |
|---|---|---|
| `create <branch> <image> [flags] [cmd]` | `c` | Create worktree + start container |
| `run <branch> <command...>` | `r` | Run a command in the container |
| `shell <branch>` | `sh`, `s` | Open an interactive shell |
| `list` | `ls` | List all nekotree containers |
| `remove <branch>` | `rm` | Stop container and remove worktree |

### `create` flags

| Flag | Description |
|---|---|
| `-f <flags>` | Extra Docker flags passed to `docker run` (e.g. `-f "-p 8080:3000"`) |

### Config file (optional)

Place `nekotree-config.json` at the repo root to activate Docker Compose mode:

```json
{ "compose_file": "docker-compose.yaml" }
```

### Environment variables

| Variable | Description |
|---|---|
| `DEVENV_MOUNTS` | Extra volume mounts: `/host:/container` or `/host:/container:ro` (comma-separated) |
| `DEBUG` | Enable verbose logging |

## Project Structure

```
nekotree/
├── cmd/nekotree/main.go          # CLI entry point
├── internal/
│   ├── config/config.go          # JSON config loading
│   ├── docker/container.go       # Container lifecycle
│   ├── gitworktree/worktree.go   # Git worktree create/remove
│   ├── runner/runner.go          # CommandRunner interface
│   ├── utils/validate.go         # Input sanitization + BuildName
│   └── volumes/mount.go          # DEVENV_MOUNTS volume handling
├── integration/                  # Integration tests (require Docker)
├── docs/                         # Documentation source
├── scripts/build.go              # Build, test, docs, release pipeline
└── Makefile                      # Thin wrapper around scripts/build.go
```

## Development

```bash
make install-tools  # Install gomarkdoc, govulncheck, gosec, mkdocs, etc.
make test           # Unit tests
make test-int       # Integration tests (requires Docker)
make test-all       # Both
make build          # Compile → build/nekotree
make docs           # Generate docs → site/
make serve-docs     # Live-reload docs server at http://localhost:8000
```

## CI/CD

| Workflow | Trigger | What it does |
|---|---|---|
| CI | push to main, all PRs | Unit tests, security scan, integration tests, build |
| Deploy Docs | push to main | Generate and deploy docs to GitHub Pages |
| Release | push `v*` tag | Cross-compile for linux/amd64, darwin/amd64, darwin/arm64 and publish GitHub Release |

To cut a release:

```bash
git tag v1.0.0 && git push origin v1.0.0
```

## Security

All external inputs (branch names, paths) are sanitized before use. `govulncheck` and `gosec` run on every CI build and fail on findings. See the [Security Report](https://cutecuteneko.github.io/nekotree/security/) in the docs.

## License

MIT
