# CLAUDE.md

## Project Context

This is **Nekotree**, a Go CLI tool for managing on-demand development environments using Git worktrees and Docker containers. It uses `scripts/build.go` as the central build script (with a Makefile wrapper for convenience) and runs CI tests/workflows on GitHub.

**Skill references:**
- `.claude/nekotree-internal.md` — how to work on this project (build, test, add features)
- `docs/nekotree-skill.md` — how external agents use the nekotree binary in other repos

## Architecture

- **CLI**: `cmd/nekotree/` — commands: create, run, shell, list, remove
- **Build system**: `scripts/build.go` (single Go script, Makefile is a thin wrapper)
- **Documentation**: `docs/` directory with manual markdown; generated to `build/site/`
- **CI/CD**: GitHub Actions workflows for testing and deployment
- **Docker**: Container management in `internal/docker/`
- **Git worktrees**: Isolated environments via `internal/gitworktree/`

## Key Files

| File | Purpose |
|------|---------|
| `scripts/build.go` | Central build script (all build/test/docs operations) |
| `Makefile` | Thin wrapper around `scripts/build.go` |
| `mkdocs.yaml` | MkDocs configuration for static site |
| `docs/index.md` | Homepage content |
| `docs/architecture.md` | System architecture documentation |
| `.github/workflows/` | CI/CD GitHub Actions workflows |
| `internal/config/config.go` | Config loading |
| `internal/docker/container.go` | Docker container lifecycle |
| `internal/gitworktree/worktree.go` | Git worktree management |
| `internal/utils/validate.go` | Path/name sanitization |
| `internal/volumes/mount.go` | Volume mount management |

## Build Commands

Use the Makefile for convenience, or `go run scripts/build.go` directly — they are equivalent:

```bash
make build          # go run scripts/build.go build
make test           # go run scripts/build.go test
make test-int       # go run scripts/build.go test --int  (requires Docker)
make test-all       # unit + integration tests
make docs           # go run scripts/build.go docs --build
make serve-docs     # go run scripts/build.go docs --serve
make clean          # clean build artifacts
make install-tools  # go run scripts/build.go install-tools
```

## Project Workflow

1. **Local development**: `make serve-docs` for live docs preview
2. **Testing**: `make test` for unit tests; `make test-int` requires Docker
3. **CI**: Pushes to main trigger tests and deploy to GitHub Pages
4. **Docs**: Manual docs in `docs/`, auto-generated API docs from Go packages

## Important Notes

- **GitHub Pages**: Site auto-deploys on main branch pushes (public repos only, or org members)
- **Docker required**: Integration tests (`make test-int`) need Docker running
- **Static site**: All docs generated to `build/site/` directory
- **Generated files**: `docs/security.md`, `docs/coverage.md`, `docs/api/*.md` are generated at build time — not committed to the repo

## Common Tasks

- **Update homepage**: Edit `docs/index.md`
- **Add new API docs**: Write `go:doc` comments in Go packages
- **Change build config**: Modify `internal/config/config.go`
- **Add container**: Update `internal/docker/container.go`
- **Fix build**: `make build`

## Testing

- Unit tests: `make test`
- Integration tests: `make test-int`
- All tests run in CI on GitHub before deployment

## Deployment

- Push to main → GitHub Actions CI → Deploy to GitHub Pages
- Site served from `build/site/`
- Only static assets (no backend required)

## Security

- `govulncheck` and `gosec` run during builds for vulnerability scanning
- Path/name inputs sanitized via `internal/utils/validate.go`
- Results documented in generated `docs/security.md`

## Docker Command Passing

```bash
# Run /bin/bash inside container
nekotree create feature-login alpine:latest /bin/bash

# Run npm start
nekotree create feature-login node:18 npm start

# With port mapping
nekotree create feature-login node:18 -f "-p 8080:3000" -- npm start

# With compose file (no default command)
nekotree create feature-login docker-compose.yaml

# Compose with custom command
nekotree create feature-login docker-compose.yaml -- npm start
```

## Volume Mounts via Environment

`DEVENV_MOUNTS` accepts comma-separated `host:container` or `host:container:ro` entries:

```bash
export DEVENV_MOUNTS="/src:/workspace,/data:/data:ro"
```
