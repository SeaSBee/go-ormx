# go-ormx Data Access Layer - Makefile

.PHONY: help build test test-unit test-integration test-benchmark clean lint fmt vet deps example

# Default target
help: ## Show this help message
	@echo "go-ormx Data Access Layer"
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Build targets
build: ## Build all packages
	@echo "Building all packages..."
	go build ./...

build-examples: ## Build examples
	@echo "Building examples..."
	go build ./examples/...

# Test targets
test: test-unit ## Run all tests

test-unit: ## Run unit tests
	@echo "Running unit tests..."
	go test -v -race -cover ./tests/unit/...

test-unit-verbose: ## Run unit tests with verbose output
	@echo "Running unit tests with verbose output..."
	go test -v -race -cover -coverprofile=coverage.out ./tests/unit/...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-integration: ## Run integration tests
	@echo "Running integration tests..."
	./tests/run_integration_tests.sh

test-integration-short: ## Run integration tests (short version)
	@echo "Running short integration tests..."
	./tests/run_integration_tests.sh --short

test-integration-coverage: ## Run integration tests with coverage
	@echo "Running integration tests with coverage..."
	./tests/run_integration_tests.sh --coverage

test-benchmarks: ## Run performance benchmarks
	@echo "Running performance benchmarks..."
	./tests/run_integration_tests.sh --benchmarks

test-stress: ## Run stress tests
	@echo "Running stress tests..."
	./tests/run_integration_tests.sh --stress

test-all: test-unit test-integration ## Run all tests (unit + integration)
	@echo "All tests completed!"

test-benchmark: ## Run benchmark tests
	@echo "Running benchmark tests..."
	go test -v -run=^$$ -bench=. -benchmem ./tests/benchmark/...

test-coverage: ## Generate test coverage report
	@echo "Generating test coverage report..."
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Code quality targets
lint: ## Run linter
	@echo "Running linter..."
	golangci-lint run

fmt: ## Format code
	@echo "Formatting code..."
	go fmt ./...

vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...

# Dependency management
deps: ## Download dependencies
	@echo "Downloading dependencies..."
	go mod download

deps-update: ## Update dependencies
	@echo "Updating dependencies..."
	go get -u ./...
	go mod tidy

deps-verify: ## Verify dependencies
	@echo "Verifying dependencies..."
	go mod verify

# Development helpers
clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	go clean ./...
	rm -f coverage.out coverage.html

example: build ## Run the basic usage example
	@echo "Running basic usage example..."
	go run ./examples/basic_usage.go

# Database helpers
db-up: ## Start database services (requires Docker)
	@echo "Starting database services..."
	docker-compose -f docker-compose.dev.yml up -d

db-down: ## Stop database services
	@echo "Stopping database services..."
	docker-compose -f docker-compose.dev.yml down

db-reset: ## Reset database services
	@echo "Resetting database services..."
	docker-compose -f docker-compose.dev.yml down -v
	docker-compose -f docker-compose.dev.yml up -d

# Documentation
docs: ## Generate documentation
	@echo "Generating documentation..."
	godoc -http=:6060
	@echo "Documentation server started at http://localhost:6060"

# Release helpers
check: lint vet test ## Run all checks before commit

pre-commit: fmt check ## Run pre-commit checks

# Install development tools
install-tools: ## Install development tools
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/godoc@latest

# Project info
info: ## Show project information
	@echo "go-ormx Data Access Layer"
	@echo "Version: $(shell grep 'Version =' go-ormx.go | cut -d'"' -f2)"
	@echo "Go version: $(shell go version)"
	@echo "Module: $(shell go list -m)"
	@echo "Dependencies:"
	@go list -m all | head -10