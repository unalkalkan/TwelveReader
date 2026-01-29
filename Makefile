.PHONY: help build test lint clean run install-tools

# Default target
.DEFAULT_GOAL := help

# Binary name
BINARY_NAME=twelvereader
BUILD_DIR=./bin

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

help: ## Display this help message
	@echo "TwelveReader Server - Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

build: ## Build the server binary
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/server

test: ## Run all tests
	@echo "Running tests..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...

test-coverage: test ## Run tests with coverage report
	@echo "Generating coverage report..."
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

lint: ## Run linter (go vet and go fmt check)
	@echo "Running linter..."
	$(GOVET) ./...
	@echo "Checking formatting..."
	@test -z "$$(gofmt -l .)" || (echo "Code is not formatted. Run 'make fmt'" && exit 1)

fmt: ## Format code
	@echo "Formatting code..."
	$(GOFMT) ./...

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html

run: build ## Build and run the server
	@echo "Starting server..."
	$(BUILD_DIR)/$(BINARY_NAME)

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

install-tools: ## Install development tools
	@echo "Installing development tools..."
	$(GOGET) golang.org/x/tools/cmd/goimports@latest

dev: ## Run server in development mode
	@echo "Running in development mode..."
	$(GOCMD) run ./cmd/server -config config/dev.example.yaml

# Docker targets
.PHONY: docker-build docker-run docker-up docker-down docker-dev docker-logs

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t twelvereader:latest .

docker-run: docker-build ## Build and run Docker container
	@echo "Running Docker container..."
	docker run -p 8080:8080 -v $(PWD)/config/docker.yaml:/app/config/config.yaml:ro twelvereader:latest

docker-up: ## Start services with docker-compose
	@echo "Starting services..."
	docker compose up -d

docker-down: ## Stop services with docker-compose
	@echo "Stopping services..."
	docker compose down

docker-dev: ## Start development environment with docker-compose
	@echo "Starting development environment..."
	docker compose -f docker-compose.dev.yaml up

docker-logs: ## View docker-compose logs
	docker compose logs -f

.PHONY: all
all: clean deps lint test build ## Run all checks and build
