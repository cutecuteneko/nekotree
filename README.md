# Nekotree

Documentation website built with MkDocs and Go.

## Quick Start

```bash
# Install required tools
make install-tools

# Build the static site
make build

# Generate docs (site goes to build/site/)
make docs

# Local dev server
make serve-docs
```

## Project Structure

```
├── scripts/build.go       # Main build script (no Makefile needed)
├── docs/                   # Manual documentation
│   ├── index.md           # Homepage
│   ├── architecture.md    # System architecture
│   └── img/               # Static images
├── mkdocs.yaml            # MkDocs configuration
└── .github/workflows/     # CI workflows
```

## CI/CD

- **Tests** run on push/PR to main
- **GitHub Pages** auto-deploys on main branch pushes
- **Static site** generated to `build/site/`

## License

MIT
