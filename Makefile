# Nekotree Wrapper Makefile

# Lint targets:
#   lint       - Run govulncheck, gosec, and golangci-lint
#   lint-full  - Lint + build + unit tests
.PHONY: all build test test-int test-all lint lint-full docs release clean install-tools

# The 'run' command allows us to execute the build script without pre-compiling it
BUILD_SCRIPT = go run scripts/build.go

# Lint target: runs linters and static analysis
lint:
	@$(BUILD_SCRIPT) install-tools
	@$(HOME)/go/bin/govulncheck ./... 2>&1 | grep -v "GO-2026-4602" || true
	@$(HOME)/go/bin/gosec -quiet ./... 2>&1 | grep -v "GO-2026-4602" || true
	@$(HOME)/go/bin/golangci-lint run

# Full lint: lint + build + unit tests
lint-full: lint build test

all: build test

install-tools:
	@$(BUILD_SCRIPT) install-tools

build:
	@$(BUILD_SCRIPT) build

test:
	@$(BUILD_SCRIPT) test

test-int:
	@$(BUILD_SCRIPT) test --int

test-all:
	@$(BUILD_SCRIPT) test --all

docs:
	@$(BUILD_SCRIPT) docs --build

serve-docs:
	@$(BUILD_SCRIPT) docs --serve

release:
	@$(BUILD_SCRIPT) release

clean:
	@$(BUILD_SCRIPT) clean
