# ==============================================================================
# Nekotree Makefile
# ==============================================================================

# Variables - Path Resolution
BINARY_NAME=nekotree
BUILD_DIR=build
SITE_DIR=site
# Dynamically resolve Go binary paths
GOPATH_BIN=$(shell go env GOPATH)/bin

# Tool Binaries
GOMARKDOC_BIN=$(GOPATH_BIN)/gomarkdoc
GOPLANTUML_BIN=$(GOPATH_BIN)/goplantuml
GOVULNCHECK_BIN=$(GOPATH_BIN)/govulncheck
GOSEC_BIN=$(GOPATH_BIN)/gosec
TEST_REPORT_BIN=$(GOPATH_BIN)/test-report
DOCKER_BIN=$(shell which docker)

# Project Settings
IMAGE_NAME=nekotree
CONTAINER_NAME=nekotree
# Default to current directory, but overridable: make docker-up HOST_PROJECT_PATH=/home/yunimoo/Gitea
HOST_PROJECT_PATH?=$(shell pwd)

.PHONY: all build test test-int clean install-tools venv \
        prepare-docs generate-api-docs generate-uml-diagrams \
        generate-security-reports build-docs serve-docs \
        release docker-build docker-up docker-down shell

all: build test

# --- Development Tools & Venv ---

install-tools: venv
	@echo "🛠️ Installing Go tools..."
	go install github.com/princjef/gomarkdoc/cmd/gomarkdoc@latest
	go install github.com/jfeliu007/goplantuml/cmd/goplantuml@latest
	@echo "🛡️ Installing Security & Reporting tools..."
	go install golang.org/x/vuln/cmd/govulncheck@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	go install github.com/blugnu/test-report@latest

venv:
	python3 -m venv venv
	./venv/bin/pip install --upgrade pip
	./venv/bin/pip install -r requirements.txt

# --- Documentation Pipeline ---

prepare-docs: venv
	@echo "📂 Preparing documentation source in $(BUILD_DIR)/docs..."
	@mkdir -p $(BUILD_DIR)/docs/api $(BUILD_DIR)/docs/img $(BUILD_DIR)/docs/uml $(BUILD_DIR)/docs/coverage
	# Copy manual docs into the build directory
	@cp docs/index.md $(BUILD_DIR)/docs/
	@cp docs/architecture.md $(BUILD_DIR)/docs/
	@cp docs/404.md $(BUILD_DIR)/docs/ 2>/dev/null || true
	# Copy existing images if any
	@cp -r docs/img/* $(BUILD_DIR)/docs/img/ 2>/dev/null || true

generate-api-docs: prepare-docs
	@echo "📝 Generating API Markdown..."
	$(GOMARKDOC_BIN) --format github ./internal/config/... -o $(BUILD_DIR)/docs/api/config.md
	$(GOMARKDOC_BIN) --format github ./internal/docker/... -o $(BUILD_DIR)/docs/api/docker.md
	$(GOMARKDOC_BIN) --format github ./internal/gitworktree/... -o $(BUILD_DIR)/docs/api/git.md

generate-uml-diagrams: prepare-docs
	@echo "📝 Generating UML diagrams..."
	$(GOPLANTUML_BIN) -recursive ./internal > $(BUILD_DIR)/docs/uml/api.puml
	$(DOCKER_BIN) run --rm -v $(PWD)/$(BUILD_DIR)/docs/uml:/data plantuml/plantuml -o /data /data/api.puml
	@mv -f $(BUILD_DIR)/docs/uml/api.png $(BUILD_DIR)/docs/img/api.png

generate-security-reports: prepare-docs
	@echo "🛡️  Generating Security & Coverage reports..."
	
	# 1. Calculate Coverage & Generate HTML Report
	@go test -coverprofile=$(BUILD_DIR)/cover.out ./... > /dev/null
	$(eval COVERAGE_PCT=$(shell go tool cover -func=$(BUILD_DIR)/cover.out | grep total | awk '{print $$3}' | sed 's/%//'))
	@go tool cover -html=$(BUILD_DIR)/cover.out -o $(BUILD_DIR)/docs/coverage/index.html
	@echo "📊 Current Coverage: $(COVERAGE_PCT)%"

	# 2. Update Badges in build/docs/index.md
	@sed -i "s/coverage-[0-9]\{1,3\}%/coverage-$(COVERAGE_PCT)%/g" $(BUILD_DIR)/docs/index.md
	@$(GOVULNCHECK_BIN) ./... > /dev/null 2>&1; \
	if [ $$? -eq 0 ]; then \
		sed -i "s/security-check--[a-z]*/security-check--passed/g" $(BUILD_DIR)/docs/index.md; \
		sed -i "s/blue/brightgreen/g" $(BUILD_DIR)/docs/index.md; \
	else \
		sed -i "s/security-check--[a-z]*/security-check--vulnerable/g" $(BUILD_DIR)/docs/index.md; \
		sed -i "s/blue/red/g" $(BUILD_DIR)/docs/index.md; \
	fi

	# 3. Generate Vulnerability Scan Report
	@echo "# 🛡️ Vulnerability Scan" > $(BUILD_DIR)/docs/security.md
	@echo "Last scanned: $(shell date)" >> $(BUILD_DIR)/docs/security.md
	@echo "## govulncheck Results" >> $(BUILD_DIR)/docs/security.md
	@echo '```text' >> $(BUILD_DIR)/docs/security.md
	-$(GOVULNCHECK_BIN) ./... >> $(BUILD_DIR)/docs/security.md 2>&1
	@echo '```' >> $(BUILD_DIR)/docs/security.md
	
	# 4. Generate Static Analysis (Gosec)
	@echo "## Static Analysis (Gosec)" >> $(BUILD_DIR)/docs/security.md
	@echo '```text' >> $(BUILD_DIR)/docs/security.md
	-$(GOSEC_BIN) -fmt=text ./... >> $(BUILD_DIR)/docs/security.md 2>&1
	@echo '```' >> $(BUILD_DIR)/docs/security.md

	# 5. Generate Detailed Coverage Markdown
	@echo "# 📊 Code Test Coverage" > $(BUILD_DIR)/docs/coverage.md
	@echo "[Click here to view the detailed Interactive Coverage Report](./coverage/index.html)" >> $(BUILD_DIR)/docs/coverage.md
	@echo "---" >> $(BUILD_DIR)/docs/coverage.md
	-go test -json ./... | $(TEST_REPORT_BIN) -o $(BUILD_DIR)/docs/coverage.md

build-docs: venv generate-api-docs generate-uml-diagrams generate-security-reports
	@echo "🏗️  Building final static site to /$(SITE_DIR)..."
	./venv/bin/mkdocs build --config-file mkdocs.yml --site-dir $(SITE_DIR)

serve-docs: venv generate-api-docs generate-uml-diagrams generate-security-reports
	@echo "🌐 Serving from $(BUILD_DIR)/docs at http://127.0.0.1:8000"
	./venv/bin/mkdocs serve --config-file mkdocs.yml

# --- Build & Release ---

build:
	@echo "🔨 Building $(BINARY_NAME) (Static)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/nekotree

release:
	@echo "📦 Creating releases..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/nekotree
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/nekotree
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/nekotree

# --- Docker Orchestration ---

docker-build:
	@echo "🐳 Building local Docker image..."
	$(DOCKER_BIN) build -t $(IMAGE_NAME):latest .

docker-up:
	@echo "🚀 Starting nekotree manager..."
	$(DOCKER_BIN) run -d \
		--name $(CONTAINER_NAME) \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v $(HOST_PROJECT_PATH):/workspace \
		-w /workspace \
		-e NEKOTREE_HOST_PATH=/workspace:$(HOST_PROJECT_PATH) \
		$(IMAGE_NAME):latest

docker-down:
	@echo "🛑 Stopping and removing nekotree container..."
	$(DOCKER_BIN) rm -f $(CONTAINER_NAME) 2>/dev/null || true

shell:
	$(DOCKER_BIN) exec -it $(CONTAINER_NAME) bash

# --- Testing & Cleanup ---

test:
	@echo "🧪 Running unit tests..."
	go test -v ./internal/...

clean:
	@echo "🧹 Cleaning up project..."
	@rm -rf $(BUILD_DIR)
	@rm -rf venv
	@rm -rf $(SITE_DIR)
	@$(DOCKER_BIN) rm -f $(CONTAINER_NAME) 2>/dev/null || true
	@echo "✨ Clean complete."
