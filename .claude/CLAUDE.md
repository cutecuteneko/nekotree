# CLAUDE.md

## Project Context

This is **Nekotree**, a MkDocs-based static documentation website built with Go. It uses `scripts/build.go` as the central build script (no Makefile dependency) and runs CI tests/workflows on GitHub.

## Architecture

- **Documentation**: `docs/` directory with manual markdown templates
- **Build system**: `scripts/build.go` (single Go binary, no Makefile needed)
- **Static site**: Generated to `build/site/`
- **CI/CD**: GitHub Actions workflows for testing and deployment
- **Docker**: Container definitions in `internal/docker/`
- **Git worktrees**: Isolated environments via `internal/gitworktree/`

## Key Files

| File | Purpose |
|------|---------|
| `scripts/build.go` | Central build script (all build/test/docs operations) |
| `mkdocs.yaml` | MkDocs configuration for static site |
| `docs/index.md` | Homepage content |
| `docs/architecture.md` | System architecture documentation |
| `.github/workflows/` | CI/CD GitHub Actions workflows |
| `internal/config/config.go` | Build configuration |
| `internal/docker/container.go` | Docker container definitions |

## Build Commands

```bash
# All commands go through scripts/build.go
./scripts/build.go build          # Build binary
./scripts/build.go test           # Run unit tests
./scripts/build.go test --int     # Run integration tests (requires Docker)
./scripts/build.go docs --build   # Generate docs to build/site/
./scripts/build.go docs --serve   # Local dev server
./scripts/build.go install-tools  # Install dependencies
```

## Project Workflow

1. **Local development**: Use `make docs` or `go run scripts/build.go docs --serve`
2. **Testing**: Run unit tests (`./scripts/build.go test`), integration tests require Docker
3. **CI**: Pushes to main trigger tests and deploy to GitHub Pages
4. **Docs**: Manual docs in `docs/`, auto-generated docs from Go packages

## Important Notes

- **No Makefile needed**: `scripts/build.go` is the actual build script; Makefile is just a thin wrapper
- **GitHub Pages**: Site auto-deploys on main branch pushes (public repos only, or org members)
- **Docker required**: Integration tests (`--int`) need Docker running
- **Static site**: All docs generated to `build/site/` directory

## Common Tasks

- **Update homepage**: Edit `docs/index.md`
- **Add new API docs**: Write `go:doc` comments in Go packages
- **Change build config**: Modify `internal/config/config.go`
- **Add container**: Update `internal/docker/container.go`
- **Fix build**: `go run scripts/build.go build`

## Testing

- Unit tests: `./scripts/build.go test`
- Integration tests: `./scripts/build.go test --int`
- All tests run in CI on GitHub before deployment

## Deployment

- Push to main → GitHub Actions test → Deploy to GitHub Pages
- Site served from `build/site/`
- Only static assets (no backend required)

## Security

- `govulncheck` and `gosec` run during builds for vulnerability scanning
- Results documented in generated docs (security.md)
