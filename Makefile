# Makefile for Infrakit

# Variables
BINARY_NAME=infrakit
DIST_DIR=dist
GO=go
GOTEST=$(GO) test
GOBUILD=$(GO) build
GOCLEAN=$(GO) clean

# Build targets
.PHONY: all build clean test test-verbose test-coverage build-all install help

# Default target
all: test build

# Build the binary for current platform
build:
	@echo "Building $(BINARY_NAME)..."
	$(GOBUILD) -o $(BINARY_NAME) -v

# Build binaries for all platforms
build-all: clean
	@echo "Building for all platforms..."
	@mkdir -p $(DIST_DIR)
	@echo "Building Linux AMD64..."
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64
	@echo "Building macOS ARM64..."
	GOOS=darwin GOARCH=arm64 $(GOBUILD) -o $(DIST_DIR)/$(BINARY_NAME)-macos-arm64
	@echo "Building macOS AMD64..."
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $(DIST_DIR)/$(BINARY_NAME)-macos-amd64
	@echo "Building Windows AMD64..."
	GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(DIST_DIR)/$(BINARY_NAME)-windows-amd64.exe
	@echo "All binaries built successfully!"

# Run all tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Run tests with verbose output
test-verbose:
	@echo "Running tests with verbose output..."
	$(GOTEST) -v -count=1 ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	@echo "Coverage report:"
	$(GO) tool cover -func=coverage.out
	@echo ""
	@echo "To view HTML coverage report, run: go tool cover -html=coverage.out"

# Run tests for a specific package
test-cache:
	@echo "Testing cache package..."
	$(GOTEST) -v ./cache/...

test-fetcher:
	@echo "Testing fetcher package..."
	$(GOTEST) -v ./fetcher/...

# Run tests with race detection
test-race:
	@echo "Running tests with race detection..."
	$(GOTEST) -race -v ./...

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f coverage.out
	rm -rf $(DIST_DIR)

# Install the binary to $GOPATH/bin
install: build
	@echo "Installing $(BINARY_NAME)..."
	$(GO) install

# Run linter (requires golangci-lint)
lint:
	@echo "Running linter..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Install with: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin"; \
	fi

# Format code
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...

# Run go mod tidy
tidy:
	@echo "Tidying dependencies..."
	$(GO) mod tidy

# Verify dependencies
verify:
	@echo "Verifying dependencies..."
	$(GO) mod verify

# Run all checks (format, test, build)
check: fmt test build
	@echo "All checks passed!"

# CI target - runs in continuous integration
ci: fmt test-coverage build-all
	@echo "CI checks completed!"

# Help target
help:
	@echo "Infrakit Makefile targets:"
	@echo ""
	@echo "  make build         - Build binary for current platform"
	@echo "  make build-all     - Build binaries for all platforms"
	@echo "  make test          - Run all tests"
	@echo "  make test-verbose  - Run tests with verbose output"
	@echo "  make test-coverage - Run tests with coverage report"
	@echo "  make test-cache    - Run cache package tests only"
	@echo "  make test-fetcher  - Run fetcher package tests only"
	@echo "  make test-race     - Run tests with race detection"
	@echo "  make clean         - Clean build artifacts"
	@echo "  make install       - Install binary to GOPATH/bin"
	@echo "  make lint          - Run linter (requires golangci-lint)"
	@echo "  make fmt           - Format code"
	@echo "  make tidy          - Tidy dependencies"
	@echo "  make verify        - Verify dependencies"
	@echo "  make check         - Run format, test, and build"
	@echo "  make ci            - Run all CI checks"
	@echo "  make help          - Show this help message"
