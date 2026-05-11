# Nekotree Internal Development Skill

This skill covers how to work on the **nekotree project itself** — building, testing, adding features, and maintaining the codebase. It is distinct from the external skill (`docs/nekotree-skill.md`), which describes how to *use* nekotree in other repositories.

---

## Project Layout

```
nekotree/
├── cmd/nekotree/main.go          # CLI entry point (commands: create, run, shell, list, remove)
├── internal/
│   ├── config/config.go          # JSON config loading, path sanitization
│   ├── docker/container.go       # Docker container lifecycle (uses CommandRunner interface)
│   ├── gitworktree/worktree.go   # Git worktree create/remove
│   ├── utils/validate.go         # Sanitize() and SanitizePath() — used throughout
│   └── volumes/mount.go          # DEVENV_MOUNTS parsing and Docker -v flag generation
├── integration/                  # Integration tests (require Docker)
├── docs/                         # Manual docs; auto-generated docs written here at build time
├── scripts/build.go              # Central build script
└── Makefile                      # Thin wrapper around scripts/build.go
```

---

## Build Commands

All build operations go through the Makefile (which delegates to `scripts/build.go`):

```bash
make lint           # Run linters and security scans
make lint-full      # Lint + build + unit tests
make build          # Compile binary → build/nekotree
make test           # Run unit tests
make test-int       # Run integration tests (requires Docker running)
make test-all       # Run unit + integration tests
make docs           # Generate static docs → build/site/
make serve-docs     # Local MkDocs dev server (live reload)
make clean          # Remove build/
make install-tools  # Install gomarkdoc, goplantuml, govulncheck, gosec, mkdocs
```

## Lint Workflow

The `make lint` target runs security and code quality checks:

1. **govulncheck** — Scans for known CVEs in dependencies
2. **gosec** — Static analysis for security anti-patterns
3. **golangci-lint** — Comprehensive Go linter

The `make lint-full` target combines linting with building and unit testing for a complete verification pipeline.

Equivalent `go run` forms (if Makefile is unavailable):

```bash
go run scripts/build.go build
go run scripts/build.go test
go run scripts/build.go test --int
go run scripts/build.go docs --build
go run scripts/build.go docs --serve
go run scripts/build.go install-tools
```

---

## Adding a New CLI Command

1. Add a `func xyzCmd() *cli.Command` in `cmd/nekotree/main.go`
2. Register it in the `Commands` slice in `main()`
3. Use `utils.Sanitize()` for any name/branch input, `utils.SanitizePath()` for paths
4. Use `docker.NewContainerManager(name, cfg, nil)` — pass `nil` runner to get `RealRunner`
5. Always default `cfg` to `&config.Config{}` if `config.Load()` returns nil
6. Add a unit test in `cmd/nekotree/main_test.go`

---

## Adding a New Container Operation

1. Add the method to `ContainerManager` in `internal/docker/container.go`
2. Use `c.runner.CombinedOutput(...)` or `c.runner.Run(...)` — **never** use `exec.Command` directly (except in `Shell()` which needs TTY control)
3. Add the method to the mock runner in `internal/docker/container_test.go` if testing
4. The `CommandRunner` interface must stay minimal — `Run` and `CombinedOutput` only
5. The `CommandRunner` interface and `RealRunner` struct live in `internal/runner/runner.go` — both `docker` and `gitworktree` packages import from there

Note: `RunCommand` passes the entire command string to `sh -c` inside the container. Callers using `nekotree run` from the CLI must quote compound commands (e.g. `"cd /workspace && go build ./..."`) so `&&` is not consumed by the local shell.

---

## Worktree Naming Convention

Worktrees are created at: `<cwd>/nekotree-<repo>-<branch>/`

Example: in repo `myapp`, branch `feature-login` → `myapp/nekotree-myapp-feature-login/`

The container is named the same: `nekotree-myapp-feature-login`

This naming is enforced in `gitworktree.CreateWorktree()` and must be kept consistent in any code that constructs `targetPath` for `docker.Start()`.

---

## Testing Patterns

### Standard mock (single static output)

```go
type mockRunner struct {
    calls  []string
    output []byte
    err    error
}

func (m *mockRunner) Run(name string, arg ...string) error {
    m.calls = append(m.calls, fmt.Sprintf("%s %s", name, strings.Join(arg, " ")))
    return m.err
}
func (m *mockRunner) CombinedOutput(name string, arg ...string) ([]byte, error) {
    m.calls = append(m.calls, fmt.Sprintf("%s %s", name, strings.Join(arg, " ")))
    return m.output, m.err
}
func (m *mockRunner) hasCall(substr string) bool { ... }
```

Pass it as `NewContainerManager(name, cfg, mock)`.

### Sequential mock (different output per call)

When an action makes multiple runner calls that need different outputs (e.g. `runRunAction` calls `Exists()` → `Start()` → `Exists()` again → `RunCommand()`), use `sequentialMock`:

```go
type sequentialMock struct {
    calls   []string
    outputs [][]byte
    errs    []error
    idx     int
}
// next() returns outputs[idx] (clamped to last entry for extra calls)
```

See `TestRunAction_NoContainer_WorktreeExists_AutoStarts` in `cmd/nekotree/main_test.go` for the canonical example.

### Injecting the mock into CLI actions

The CLI actions accept `r runner.CommandRunner` as a second param (nil → `RealRunner` in production). In tests, swap the action:

```go
app.Commands[0].Action = func(c *cli.Context) error {
    return runCreateAction(c, mock)
}
```

Integration tests in `integration/` use real Docker — run with `make test-int`.

---

## Input Validation Rules

All external inputs (branch names, paths, commands) must pass through `internal/utils`:

- `utils.Sanitize(input)` — for branch names and container names. Allows `[a-zA-Z0-9._-]`.
- `utils.SanitizePath(path)` — for filesystem paths. Allows `[a-zA-Z0-9\-._/]`, rejects traversal.

Never skip sanitization. Never call `exec.Command` with user-supplied strings that haven't been sanitized.

---

## Documentation

Docs are a mix of manual and auto-generated:

| File | Source |
|---|---|
| `docs/index.md` | Manual — edit directly |
| `docs/architecture.md` | Manual — edit directly |
| `docs/nekotree-skill.md` | Manual — the external agent skill reference |
| `docs/api/*.md` | Auto-generated by `gomarkdoc` at `make docs` |
| `docs/security.md` | Auto-generated by `govulncheck`/`gosec` at `make docs` |
| `docs/coverage.md` | Auto-generated from test coverage at `make docs` |

To update API docs: improve `go:doc` comments in the relevant `internal/` package, then run `make docs`.

---

## Config File

`nekotree-config.json` (optional, at the repo root) controls Compose mode:

```json
{
  "compose_file": "docker-compose.yaml"
}
```

| Field | Type | Description |
|---|---|---|
| `compose_file` | string | Path to a compose file — activates Compose mode in `Start()` |

Config is loaded by `config.Load(defaultConfigFile)` in every action that creates a container. If the file is absent, `Load` returns `&Config{}, nil` (an empty config, no error). Never fatal.

---

## Security Scans

Run locally before committing:

```bash
govulncheck ./...       # check for known CVEs in dependencies
gosec -quiet ./...      # static analysis for security anti-patterns
```

Install if missing: `make install-tools`

Both are also run as a separate CI step that fails the build on findings. `#nosec G204` annotations are used only for verified false positives (e.g. `Shell()` which requires `exec.Command` directly for TTY control).

---

## Binary Size Check

`make build` validates the binary is within ±10% of the expected baseline (~5.8 MB). If you add or remove a large dependency and the build fails with a size error, update the `expectedSize` and bounds in `scripts/build.go`.

Current expected range: ~5.4 MB – 6.9 MB (checked automatically; `du -sh build/nekotree` to inspect manually).

---

## Pull Request Template

A PR template lives at `.github/PULL_REQUEST_TEMPLATE.md` and is auto-loaded by GitHub when opening a PR. It includes:

- **Summary** — 1–3 bullet points describing what the PR does
- **Type of change** — checkbox mapping to conventional commit types (`fix:`, `feat:`, `enhance:`, `refactor:`, `test:`, `docs:`, `chore:`)
- **Checklist** — mirrors the verification checklist in this skill: lint, build, test, new tests, input sanitization, no `exec.Command` with unsanitized input, no secrets
- **Testing notes** — how the change was tested and any edge cases
- **Related issues** — `Closes #` / `Fixes #` links

When opening a PR via `gh pr create`, pass `--body-file .github/PULL_REQUEST_TEMPLATE.md` and fill in the placeholders, or let GitHub auto-populate the template in the web UI.

---

## CI/CD

One workflow: `.github/workflows/build-docs-and-test.yml`

Triggers on push/PR to `main`. Pipeline:
1. install-tools
2. unit tests
3. security scan (`govulncheck` + `gosec`) — **fails build on findings**
4. integration tests
5. build binary
6. generate docs
7. deploy to GitHub Pages

Integration tests run in CI — Docker is available in the GitHub Actions `ubuntu-latest` runner.

---

## Environment Variables

| Variable | Description |
|---|---|
| `DEVENV_MOUNTS` | Comma-separated additional volume mounts: `/host:/container` or `/host:/container:ro` |
| `DEBUG` | Enables verbose logging |

---

## Feature Development Workflow

### Branch Discipline

**Never work directly on main/master.** Always create a feature branch:

```bash
# Create a descriptive branch name
BRANCH="feature-<short-description>"
# Example: BRANCH="feature-user-auth"

# Create and switch to the branch
git checkout -b "$BRANCH"
```

Branch naming rules:
- Use only `a-z`, `0-9`, `-`
- Keep it short and descriptive
- Avoid special characters or spaces

### Build & Test Enforcement

For every commit, run the full verification pipeline:

```bash
# 1. Lint — security and code quality checks
make lint

# 2. Build — confirm compilation
make build

# 3. Unit tests — confirm logic correctness
make test

# 4. Integration tests — confirm Docker/container behavior
make test-int
```

**Never commit if any step fails.** Leave the environment intact for debugging.

### Commit Message Convention

Use conventional commits with a concise header:

```
<type>: <header>

<optional body>

<optional footer>
```

**Types:**
| Type | Description | Example |
|------|-------------|---------|
| `fix:` | Bug fix | `fix: resolve nil pointer in container cleanup` |
| `feat:` | New feature | `feat: add port mapping support` |
| `enhance:` | Enhancement/improvement | `enhance: improve error messages in create` |
| `refactor:` | Code restructuring | `refactor: simplify volume mount logic` |
| `docs:` | Documentation changes | `docs: update architecture.md` |
| `test:` | Test additions/modifications | `test: add unit tests for sanitize` |
| `chore:` | Maintenance tasks | `chore: update dependencies` |

**Rules:**
- Header: max 50 characters, imperative mood
- Body: wrap at 72 characters
- Be specific and accurate — describe what changed, not why
- Omit trailing punctuation in header

### Pre-commit Validation

Every commit must pass pre-commit hooks:

```bash
# Run hooks locally before committing
git commit -m "..."
```

If pre-commit fails:
- Do NOT amend with `--no-verify`
- Fix the underlying issue
- Re-run the build/test pipeline
- Commit again

### Test Coverage Requirement

Every new feature must include tests:

| Feature Type | Required Test Type |
|--------------|-------------------|
| CLI commands | Unit tests in `cmd/nekotree/*_test.go` |
| Internal packages | Unit tests in `internal/*/*_test.go` |
| Container operations | Integration tests in `integration/*_test.go` |
| Build scripts | Unit tests for `scripts/build.go` |

**Test patterns:**
- Use mocks for external dependencies (Docker, git)
- Test edge cases and error paths
- Add integration tests for features requiring Docker

### Verification Checklist

Before committing:

- [ ] `make lint` passes
- [ ] `make build` succeeds
- [ ] `make test` passes
- [ ] `make test-int` passes
- [ ] Pre-commit hooks pass
- [ ] Commit message follows convention
- [ ] New features have tests
- [ ] No sensitive data committed

---

## Common Workflows

**Make a change and verify it:**
```bash
# Edit code, then:
make lint         # run linters and security checks
make build        # confirm it compiles
make test         # confirm unit tests pass
make test-int     # confirm integration tests pass (needs Docker)
```

**Update docs after a code change:**
```bash
make docs         # regenerates api/, security.md, coverage.md
make serve-docs   # preview at http://localhost:8000
```

**Check binary size (validated automatically on build):**
```bash
du -sh build/nekotree
# Expected ~6MB ±10%. build.go will fail if outside range.
```
