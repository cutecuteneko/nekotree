# Nekotree Wrapper Makefile
.PHONY: all build test test-int test-all docs release clean install-tools

# The 'run' command allows us to execute the build script without pre-compiling it
BUILD_SCRIPT = go run scripts/build.go

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
