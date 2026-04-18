# Contributing

Thanks for your interest in contributing to Nekotree!

## Development

1. **Fork and clone** the repository
2. **Install dependencies**:

```bash
./scripts/build.go install-tools
```

3. **Build and test**:

```bash
./scripts/build.go build
./scripts/build.go test
```

4. **Run locally**:

```bash
./scripts/build.go docs --serve
```

## Style Guide

- **Go**: Follow standard Go formatting (`gofmt`)
- **Markdown**: Use clear headings and proper formatting
- **Code comments**: Write `go:doc` comments for API documentation
- **Testing**: Write tests for new features

## Pull Request Guidelines

1. **Test locally** before submitting
2. **Include tests** for new features/fixes
3. **Update documentation** if needed
4. **Keep PRs small** and focused

## Questions?

Open an issue or PR for any questions or suggestions!
