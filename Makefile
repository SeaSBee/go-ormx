# Makefile for Go ORMX
.PHONY: help build test lint clean docker-build docker-run migrate dev bench security-test integration-test

# Default target
help:
	@echo "Available targets:"
	@echo "  build        - Build the application"
	@echo "  test         - Run tests"
	@echo "  lint         - Run linter"
	@echo "  clean        - Clean build artifacts"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Run Docker containers"
	@echo "  migrate      - Run database migrations"
	@echo "  dev          - Start development environment"
	@echo "  bench        - Run benchmarks"
	@echo "  security-test - Run security tests"
	@echo "  integration-test - Run integration tests"

# Build
build:
	@echo "Building Go ORMX..."
	go build -o bin/ormx ./cmd/ormx-cli

# Test
test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Lint
lint:
	@echo "Running linter..."
	golangci-lint run

# Clean
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -f coverage.out coverage.html

# Docker
docker-build:
	@echo "Building Docker image..."
	docker-compose build

docker-run:
	@echo "Starting Docker containers..."
	docker-compose up -d

# Migrations
migrate:
	@echo "Running database migrations..."
	migrate -path database/migrations -database "postgres://postgres:password@localhost:5432/ormx_dev?sslmode=disable" up

# Development
dev:
	@echo "Starting development environment..."
	docker-compose up -d
	@sleep 10
	@echo "Running migrations..."
	@make migrate
	@echo "Starting application..."
	go run ./cmd/ormx-cli

# Benchmarks
bench:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./tests/benchmark/...

# Security tests
security-test:
	@echo "Running security tests..."
	go test -v ./tests/security/...

# Integration tests
integration-test:
	@echo "Running integration tests..."
	go test -v ./tests/integration/...

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Vet code
vet:
	@echo "Vetting code..."
	go vet ./...

# Generate mocks
mocks:
	@echo "Generating mocks..."
	mockgen -source=pkg/repository/interfaces.go -destination=tests/mocks/repository_mock.go
	mockgen -source=pkg/models/interfaces.go -destination=tests/mocks/models_mock.go

# Run all checks
check: fmt vet lint test

# Setup development environment
setup:
	@echo "Setting up development environment..."
	@make deps
	@make docker-run
	@sleep 15
	@make migrate
	@echo "Development environment ready!"

# Clean development environment
clean-dev:
	@echo "Cleaning development environment..."
	docker-compose down -v
	@make clean
