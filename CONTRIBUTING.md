# Contributing

Thanks for your interest in contributing to Nekotree!

## Development Setup

1. **Fork and clone** the repository
2. **Install dependencies** (requires Go 1.21+ and Docker):

```bash
make install-tools
```

3. **Build and test**:

```bash
make build       # compile → build/nekotree
make test        # unit tests
make test-int    # integration tests (requires Docker running)
```

4. **Run the binary locally**:

```bash
./build/nekotree --help
```

## Before Submitting a PR

Run the full check locally to catch issues before CI does:

```bash
make test-all         # unit + integration tests
govulncheck ./...     # check for known CVEs
gosec -quiet ./...    # static security analysis
make build            # confirm it compiles and passes size check
```

## Style Guide

- **Go**: Standard formatting (`gofmt`) — CI will catch violations via `gosec`
- **Code comments**: Write `go:doc` comments for exported symbols; API docs are auto-generated from them at `make docs`
- **Input handling**: All external inputs must go through `utils.Sanitize()` or `utils.SanitizePath()` — never pass user strings directly to `exec.Command`
- **Testing**: Use the `mockRunner` / `sequentialMock` pattern for unit tests; real Docker for integration tests in `integration/`

## Pull Request Guidelines

1. **Keep PRs small and focused** — one feature or fix per PR
2. **Include tests** for new features and bug fixes
3. **Update docs** if you change CLI behaviour or add config options — run `make docs` to regenerate API docs
4. **Security scans must pass** — `govulncheck` and `gosec` run in CI and fail the build on findings; use `#nosec` annotations only for verified false positives with an explanatory comment

## Automated Bots

- **Dependabot** opens weekly PRs for Go module and GitHub Actions updates
- **Mergify** auto-merges Dependabot PRs once CI passes — no manual review needed for routine dependency bumps
- **CodeRabbit** reviews all PRs automatically — address its comments before requesting human review

## Questions?

Open an issue or start a discussion — PRs are welcome!
