# Makefile for flagd HTTP Connector

.PHONY: help build test test-coverage lint clean install-tools format vet mod-tidy example

# Default target
help: ## Show this help message
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Build
build: ## Build the package
	@echo "Building..."
	@go build ./...

# Testing
test: ## Run unit tests
	@echo "Running tests..."
	@go test -v ./...

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	@go test -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-integration: ## Run integration tests (requires flagd)
	@echo "Running integration tests..."
	@echo "Note: Make sure flagd is running on localhost:8013"
	@go test -v -tags=integration ./...

# Code quality
lint: install-tools ## Run linter
	@echo "Running linter..."
	@golangci-lint run

format: ## Format code
	@echo "Formatting code..."
	@go fmt ./...

vet: ## Run go vet
	@echo "Running go vet..."
	@go vet ./...

# Dependencies
mod-tidy: ## Tidy go modules
	@echo "Tidying modules..."
	@go mod tidy

mod-download: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download

# Tools
install-tools: ## Install development tools
	@echo "Installing tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Examples
example: ## Run example
	@echo "Running example..."
	@cd examples && go run main.go

# Docker targets
docker-flagd: ## Start flagd in Docker for testing
	@echo "Starting flagd in Docker..."
	@docker run -d --name flagd-test -p 8013:8013 \
		-v $(PWD)/examples/flags.json:/flags.json \
		ghcr.io/open-feature/flagd:latest start --uri file:./flags.json

docker-stop: ## Stop flagd Docker container
	@echo "Stopping flagd..."
	@docker stop flagd-test || true
	@docker rm flagd-test || true

# Release
tag: ## Create a new tag (usage: make tag VERSION=v1.0.0)
	@if [ -z "$(VERSION)" ]; then echo "Usage: make tag VERSION=v1.0.0"; exit 1; fi
	@echo "Creating tag $(VERSION)..."
	@git tag -a $(VERSION) -m "Release $(VERSION)"
	@echo "Tag created. Push with: git push origin $(VERSION)"

# Cleanup
clean: ## Clean build artifacts
	@echo "Cleaning..."
	@go clean ./...
	@rm -f coverage.out coverage.html

# Development workflow
dev-setup: install-tools mod-download ## Setup development environment
	@echo "Development environment ready!"

check: format vet lint test ## Run all checks
	@echo "All checks passed!"

# CI targets
ci: mod-tidy format vet lint test-coverage ## Run CI pipeline
	@echo "CI pipeline completed!"