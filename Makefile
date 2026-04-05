.PHONY: all build build-all clean test lint lint-fix lint-report fix-permissions fix-all fmt vet install-tools help check install-hooks require-sudo install uninstall install-service uninstall-service start-service stop-service status-service

# Variables
BINARY_NAME_SERVER=rcode-server
BINARY_NAME_CLIENT=rcode
BUILD_DIR=bin
INSTALL_DIR=/usr/local/bin
UNAME_S := $(shell uname -s)
LAUNCH_AGENT_LABEL=com.foxytanuki.rcode-server
LAUNCH_AGENT_PLIST=$(HOME)/Library/LaunchAgents/$(LAUNCH_AGENT_LABEL).plist

# Version info from git
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILDTIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
GITHASH := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go parameters
# Check if mise is available, otherwise use go directly
MISE_EXISTS := $(shell command -v mise 2> /dev/null)
ifdef MISE_EXISTS
    GOCMD=mise exec -- go
else
    GOCMD=go
endif

GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

# Build flags
VERSION_PKG=github.com/foxytanuki/rcode/internal/version
LDFLAGS=-ldflags "-s -w \
	-X $(VERSION_PKG).Version=$(VERSION) \
	-X $(VERSION_PKG).BuildTime=$(BUILDTIME) \
	-X $(VERSION_PKG).GitHash=$(GITHASH)"

# Default target
all: test build

## help: Show this help message
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

## build: Build both server and client binaries for current platform
build: build-server build-client

## build-server: Build the server binary for current platform
build-server:
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME_SERVER) ./cmd/server

## build-client: Build the client binary for current platform
build-client:
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME_CLIENT) ./cmd/rcode

## build-all: Cross-compile binaries for all platforms
build-all: build-darwin-amd64 build-darwin-arm64 build-linux-amd64 build-linux-arm64

## build-darwin-amd64: Build for macOS (Intel)
build-darwin-amd64:
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME_SERVER)-darwin-amd64 ./cmd/server
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME_CLIENT)-darwin-amd64 ./cmd/rcode

## build-darwin-arm64: Build for macOS (Apple Silicon)
build-darwin-arm64:
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME_SERVER)-darwin-arm64 ./cmd/server
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME_CLIENT)-darwin-arm64 ./cmd/rcode

## build-linux-amd64: Build for Linux (x86_64)
build-linux-amd64:
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME_SERVER)-linux-amd64 ./cmd/server
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME_CLIENT)-linux-amd64 ./cmd/rcode

## build-linux-arm64: Build for Linux (ARM64)
build-linux-arm64:
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME_SERVER)-linux-arm64 ./cmd/server
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME_CLIENT)-linux-arm64 ./cmd/rcode

## test: Run all tests
test:
	$(GOTEST) -v -race -coverprofile=coverage.out ./...

## test-short: Run short tests
test-short:
	$(GOTEST) -v -short ./...

## test-coverage: Run tests with coverage report
test-coverage: test
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

## benchmark: Run benchmarks
benchmark:
	$(GOTEST) -bench=. -benchmem ./...

## lint: Run golangci-lint
lint:
	@if command -v mise > /dev/null; then \
		mise exec golangci-lint -- golangci-lint run ./...; \
	elif command -v golangci-lint > /dev/null; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Install via 'mise use golangci-lint@latest' or 'make install-tools'"; \
		exit 1; \
	fi

## lint-fix: Auto-fix lint issues where possible
lint-fix:
	@echo "🔧 Running automatic lint fixes..."
	@if command -v mise > /dev/null; then \
		mise exec golangci-lint -- golangci-lint run --fix ./... 2>&1 | head -1 || true; \
	elif command -v golangci-lint > /dev/null; then \
		golangci-lint run --fix ./... 2>&1 | head -1 || true; \
	else \
		echo "golangci-lint not installed. Install via 'mise use golangci-lint@latest' or 'make install-tools'"; \
		exit 1; \
	fi
	@echo "✅ Auto-fix completed (some issues may need manual fixing)"

## lint-report: Generate lint report
lint-report:
	@./scripts/lint-report.sh

## fix-permissions: Fix file permissions (0755->0750, 0644->0600)
fix-permissions:
	@echo "🔐 Fixing file permissions..."
	@find . -name "*.go" -type f -exec sed -i 's/0755/0750/g; s/0644/0600/g' {} \;
	@echo "✅ Permissions fixed"

## fix-all: Run all automatic fixes (lint, permissions, fmt)
fix-all: fmt fix-permissions lint-fix
	@echo "✨ All automatic fixes completed"
	@echo "📊 Remaining issues:"
	@make lint 2>&1 | tail -5 || true

## fmt: Format code with simplifications
fmt:
	@if command -v mise > /dev/null; then \
		mise exec go -- gofmt -s -w .; \
	else \
		gofmt -s -w .; \
	fi
	@echo "Code formatted"

## vet: Run go vet
vet:
	$(GOVET) ./...
	@echo "Vet completed"

## tidy: Tidy and verify module dependencies
tidy:
	$(GOMOD) tidy
	$(GOMOD) verify

## clean: Clean build artifacts
clean:
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

## require-sudo: Refresh sudo credentials for privileged targets
require-sudo:
	sudo -v

## install: Install binaries to system (requires sudo)
install: build require-sudo
	@echo "Installing binaries to $(INSTALL_DIR)..."
	@was_loaded=0; \
	if [ "$(UNAME_S)" = "Darwin" ] && launchctl print gui/$$(id -u)/$(LAUNCH_AGENT_LABEL) >/dev/null 2>&1; then \
		was_loaded=1; \
		launchctl bootout gui/$$(id -u) "$(LAUNCH_AGENT_PLIST)" >/dev/null 2>&1 || true; \
	fi; \
	sudo mkdir -p $(INSTALL_DIR); \
	sudo cp $(BUILD_DIR)/$(BINARY_NAME_SERVER) $(INSTALL_DIR)/; \
	sudo cp $(BUILD_DIR)/$(BINARY_NAME_CLIENT) $(INSTALL_DIR)/; \
	if [ "$$was_loaded" -eq 1 ]; then \
		launchctl bootstrap gui/$$(id -u) "$(LAUNCH_AGENT_PLIST)" >/dev/null 2>&1 || true; \
		launchctl kickstart -k gui/$$(id -u)/$(LAUNCH_AGENT_LABEL) >/dev/null 2>&1 || true; \
	fi
	@echo "Installation complete"

## uninstall: Uninstall binaries from system (requires sudo)
uninstall: require-sudo
	@echo "Removing binaries from $(INSTALL_DIR)..."
	sudo rm -f $(INSTALL_DIR)/$(BINARY_NAME_SERVER)
	sudo rm -f $(INSTALL_DIR)/$(BINARY_NAME_CLIENT)
	@echo "Uninstallation complete"

## install-tools: Install required development tools
install-tools:
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Tools installed"

## run-server: Run the server locally
run-server:
	$(GOCMD) run ./cmd/server

## run-client: Run the client locally
run-client:
	$(GOCMD) run ./cmd/rcode

## docker-build: Build Docker images
docker-build:
	docker build -t $(BINARY_NAME_SERVER):$(VERSION) -f Dockerfile.server .
	docker build -t $(BINARY_NAME_CLIENT):$(VERSION) -f Dockerfile.client .

## deps: Download dependencies
deps:
	$(GOMOD) download

## deps-update: Update dependencies to latest versions
deps-update:
	$(GOGET) -u ./...
	$(GOMOD) tidy

## version: Display version
version:
	@echo $(VERSION)

## check: Run all checks (fmt, vet, test, build)
check:
	@./scripts/check.sh

## install-hooks: Install git hooks via lefthook
install-hooks:
	@echo "Installing git hooks..."
	@mise exec lefthook -- lefthook install
	@echo "Git hooks installed"

## run-hooks: Run git hooks manually
run-hooks:
	@mise exec lefthook -- lefthook run pre-commit

## install-service: Install rcode-server as a system service
install-service: build-server require-sudo
	@echo "Installing rcode-server as a system service..."
	@if [ "$(UNAME_S)" = "Darwin" ] && launchctl print gui/$$(id -u)/$(LAUNCH_AGENT_LABEL) >/dev/null 2>&1; then \
		launchctl bootout gui/$$(id -u) "$(LAUNCH_AGENT_PLIST)" >/dev/null 2>&1 || true; \
	fi
	@sudo mkdir -p $(INSTALL_DIR)
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME_SERVER) $(INSTALL_DIR)/
	@$(INSTALL_DIR)/$(BINARY_NAME_SERVER) -install-service
	@echo "Service installed. It will start automatically on login."

## uninstall-service: Uninstall rcode-server system service
uninstall-service:
	@echo "Uninstalling rcode-server service..."
	@$(INSTALL_DIR)/$(BINARY_NAME_SERVER) -uninstall-service || true
	@echo "Service uninstalled."

## start-service: Start rcode-server service
start-service:
	@$(INSTALL_DIR)/$(BINARY_NAME_SERVER) -start-service

## stop-service: Stop rcode-server service
stop-service:
	@$(INSTALL_DIR)/$(BINARY_NAME_SERVER) -stop-service

## status-service: Check status of rcode-server service
status-service:
	@$(INSTALL_DIR)/$(BINARY_NAME_SERVER) -status-service
