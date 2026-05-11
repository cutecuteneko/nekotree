# Nekotree Wrapper Makefile

# Lint targets:
#   lint       - Run govulncheck, gosec, and golangci-lint
#   lint-full  - Lint + build + unit tests
.PHONY: all build test test-int test-all lint lint-full docs release clean install-tools act-ci act-build-note

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

# act targets: run workflows locally via https://github.com/nektos/act
# .actrc at repo root is auto-loaded by act

act-ci:
	act push -W .github/workflows/build-docs-and-test.yml -e .github/act/build-docs-and-test.event.json

act-build-note:
	act pull_request_target -W .github/workflows/build-note.yml -e .github/act/build-note.event.json 2>&1 | tee /tmp/act-build-note.log; \
	echo "--- Build note JSON ---"; \
	grep "json_note=" /tmp/act-build-note.log | sed 's/.*json_note=//' | python3 -m json.tool
