.PHONY: build test test-unit test-integration clean install help

# Build the binary
build:
	@echo "Building puff..."
	@mkdir -p bin
	@go build -o bin/puff ./cmd/puff
	@echo "Build complete: bin/puff"

# Run all tests
test: test-unit test-integration

# Run unit tests
test-unit:
	@echo "Running unit tests..."
	@go test -v ./pkg/...

# Run integration tests
test-integration: build
	@echo "Running integration tests..."
	@go test -v ./test/...

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@go clean
	@echo "Clean complete"

# Install to $GOPATH/bin
install:
	@echo "Installing puff..."
	@go install ./cmd/puff
	@echo "Install complete"

# Run tests with coverage
coverage:
	@echo "Running tests with coverage..."
	@go test -coverprofile=coverage.out ./pkg/...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Run linters
lint:
	@echo "Running linters..."
	@go vet ./...

# Show help
help:
	@echo "Available targets:"
	@echo "  build              - Build the puff binary"
	@echo "  test               - Run all tests"
	@echo "  test-unit          - Run unit tests only"
	@echo "  test-integration   - Run integration tests only"
	@echo "  clean              - Remove build artifacts"
	@echo "  install            - Install to GOPATH/bin"
	@echo "  coverage           - Generate test coverage report"
	@echo "  fmt                - Format code"
	@echo "  lint               - Run linters"
	@echo "  help               - Show this help message"
