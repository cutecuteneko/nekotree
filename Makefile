# Makefile
BINARY_NAME=nekotree
BUILD_DIR=build
GOMARKDOC_BIN=$(shell go env GOPATH)/bin/gomarkdoc
GOPLANTUML_BIN=$(shell go env GOPATH)/bin/goplantuml

.PHONY: all build test clean docs-serve install-tools venv generate-api-docs release

all: build test

install-tools: venv
	@echo "🛠️ Installing gomarkdoc..."
	go install github.com/princjef/gomarkdoc/cmd/gomarkdoc@latest
	go get github.com/jfeliu007/goplantuml/parser
	go install github.com/jfeliu007/goplantuml/cmd/goplantuml@latest

venv:
	python3 -m venv venv
	./venv/bin/pip install --upgrade pip
	./venv/bin/pip install -r requirements.txt

generate-api-docs:
	@echo "📝 Generating Read the Docs compatible API files..."
	@mkdir -p docs/api
	$(GOMARKDOC_BIN) --format github ./internal/config/... -o docs/api/config.md
	$(GOMARKDOC_BIN) --format github ./internal/docker/... -o docs/api/docker.md
	$(GOMARKDOC_BIN) --format github ./internal/gitworktree/... -o docs/api/git.md

generate-uml-diagrams:
	@echo "📝 Generating UML diagrams of API files..."
	@mkdir -p docs/uml
	$(GOPLANTUML_BIN) -recursive ./internal ./cmd > docs/uml/complete.puml

docs-serve: venv generate-api-docs
	@echo "🌐 Serving Read the Docs theme at http://127.0.0.1:8000"
	./venv/bin/mkdocs serve

build:
	@echo "🔨 Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/nekotree

# Cross-compile for Linux and macOS
release:
	@echo "📦 Creating releases..."
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/nekotree
	GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/nekotree
	GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/nekotree

test:
	@echo "🧪 Running tests..."
	./scripts/test.sh

clean:
	@echo "🧹 Cleaning up project..."
	@rm -rf $(BUILD_DIR)
	@rm -rf venv
	@rm -rf docs/api
	@rm -rf site
	@echo "✨ Clean complete."
