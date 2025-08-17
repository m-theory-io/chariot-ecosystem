# Makefile for Charioteer - Cross-platform build

# Project settings
BINARY_NAME=charioteer
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build flags
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.Commit=$(COMMIT)"

# Output directories
BUILD_DIR=build
DIST_DIR=dist

# Platforms
LINUX_AMD64_PLATFORM=linux/amd64
LINUX_ARM64_PLATFORM=linux/arm64
DARWIN_PLATFORM=darwin/arm64
WINDOWS_PLATFORM=windows/amd64

# Binary names with platform suffixes
LINUX_AMD64_BINARY=$(BINARY_NAME)-linux-amd64
LINUX_ARM64_BINARY=$(BINARY_NAME)-linux-arm64
DARWIN_BINARY=$(BINARY_NAME)-darwin-arm64
WINDOWS_BINARY=$(BINARY_NAME)-windows-amd64.exe

.PHONY: all build clean linux linux-amd64 linux-arm64 jetson darwin windows install test fmt vet deps help

# Default target
all: clean linux-amd64 linux-arm64 darwin

# Build for current platform
build:
	@echo "Building for current platform..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .

# Build for Linux (AMD64)
linux-amd64:
	@echo "Building for Linux (AMD64)..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(LINUX_AMD64_BINARY) .

# Build for Linux (ARM64) - Jetson
linux-arm64:
	@echo "Building for Linux (ARM64) - Jetson..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(LINUX_ARM64_BINARY) .

# Alias for Jetson
jetson: linux-arm64

# Build for both Linux architectures
linux: linux-amd64 linux-arm64

# Build for macOS (Apple M1/ARM64)
darwin:
	@echo "Building for macOS (Apple M1/ARM64)..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(DARWIN_BINARY) .

# Build for Windows (AMD64)
windows:
	@echo "Building for Windows (AMD64)..."
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(WINDOWS_BINARY) .

# Build for all platforms
all-platforms: linux darwin windows

# Create distribution archives
dist: all-platforms
	@echo "Creating distribution archives..."
	@mkdir -p $(DIST_DIR)
	@cd $(BUILD_DIR) && tar -czf ../$(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz $(LINUX_AMD64_BINARY)
	@cd $(BUILD_DIR) && tar -czf ../$(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-linux-arm64.tar.gz $(LINUX_ARM64_BINARY)
	@cd $(BUILD_DIR) && tar -czf ../$(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-darwin-arm64.tar.gz $(DARWIN_BINARY)
	@cd $(BUILD_DIR) && zip -q ../$(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-windows-amd64.zip $(WINDOWS_BINARY)
	@echo "Distribution archives created in $(DIST_DIR)/"

# Install for current platform
install: build
	@echo "Installing $(BINARY_NAME)..."
	go install $(LDFLAGS) .

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Vet code
vet:
	@echo "Vetting code..."
	go vet ./...

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR) $(DIST_DIR)

# Run the application (current platform)
run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BUILD_DIR)/$(BINARY_NAME)

# Run with common flags
run-dev: build
	@echo "Running $(BINARY_NAME) in development mode..."
	./$(BUILD_DIR)/$(BINARY_NAME) -backend=https://localhost:8087 -port=8080 -insecure=true

# Development build with race detection
dev:
	@echo "Building development version with race detection..."
	@mkdir -p $(BUILD_DIR)
	go build -race $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-dev .

# Build with debugging symbols
debug:
	@echo "Building debug version..."
	@mkdir -p $(BUILD_DIR)
	go build -gcflags="all=-N -l" $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-debug .

# Check for required tools
check-tools:
	@echo "Checking for required tools..."
	@which go > /dev/null || (echo "Go is not installed" && exit 1)
	@which git > /dev/null || (echo "Git is not installed" && exit 1)
	@echo "All required tools are available"

# Show build information
info:
	@echo "Build Information:"
	@echo "  Binary Name: $(BINARY_NAME)"
	@echo "  Version: $(VERSION)"
	@echo "  Build Time: $(BUILD_TIME)"
	@echo "  Commit: $(COMMIT)"
	@echo "  Go Version: $(shell go version)"
	@echo ""
	@echo "Supported Platforms:"
	@echo "  Linux AMD64: $(LINUX_AMD64_PLATFORM)"
	@echo "  Linux ARM64: $(LINUX_ARM64_PLATFORM) (Jetson)"
	@echo "  macOS: $(DARWIN_PLATFORM)"
	@echo "  Windows: $(WINDOWS_PLATFORM)"

# Help target
help:
	@echo "Available targets:"
	@echo "  all           - Build for Linux (both architectures) and macOS (default)"
	@echo "  build         - Build for current platform"
	@echo "  linux         - Build for both Linux architectures (AMD64 and ARM64)"
	@echo "  linux-amd64   - Build for Linux (AMD64)"
	@echo "  linux-arm64   - Build for Linux (ARM64) - Jetson"
	@echo "  jetson        - Build for Jetson (alias for linux-arm64)"
	@echo "  darwin        - Build for macOS (Apple M1/ARM64)"
	@echo "  windows       - Build for Windows (AMD64)"
	@echo "  all-platforms - Build for all platforms"
	@echo "  dist          - Create distribution archives"
	@echo "  install       - Install for current platform"
	@echo "  test          - Run tests"
	@echo "  fmt           - Format code"
	@echo "  vet           - Vet code"
	@echo "  deps          - Download dependencies"
	@echo "  clean         - Clean build artifacts"
	@echo "  run           - Run the application"
	@echo "  run-dev       - Run with development flags"
	@echo "  dev           - Build development version with race detection"
	@echo "  debug         - Build debug version with symbols"
	@echo "  check-tools   - Check for required tools"
	@echo "  info          - Show build information"
	@echo "  help          - Show this help message"