# Go Fiber Template - Makefile
# Development and testing commands

.PHONY: help build run test test-unit test-integration test-coverage clean dev lint format docker-build docker-run

# Default target
help: ## Show this help message
	@echo "Available commands:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Development commands
dev: ## Run the application with hot reload using Air
	air

run: ## Run the application directly
	go run cmd/api/main.go

build: ## Build the application
	go build -o bin/api cmd/api/main.go

clean: ## Clean build artifacts and test cache
	go clean
	rm -rf bin/
	go clean -testcache

# Testing commands
test: ## Run all tests
	go test ./...

test-unit: ## Run unit tests only
	@echo "Running unit tests..."
	go test ./tests/examples/unit/... -v

test-integration: ## Run integration tests only
	@echo "Running integration tests..."
	go test ./tests/examples/integration/... -v

test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-coverage-func: ## Show test coverage by function
	go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out

test-benchmark: ## Run benchmark tests
	go test -bench=. ./...

test-race: ## Run tests with race condition detection
	go test -race ./...

test-verbose: ## Run tests with verbose output
	go test -v ./...

test-short: ## Run tests in short mode (skip long-running tests)
	go test -short ./...

# Specific test commands
test-services: ## Run service layer tests
	go test ./application/serviceimpl/... -v

test-repositories: ## Run repository tests
	go test ./infrastructure/postgres/... -v

test-handlers: ## Run handler tests
	go test ./interfaces/api/handlers/... -v

test-websocket: ## Run WebSocket tests
	go test ./tests/examples/integration/ -run TestWebSocket -v

test-auth: ## Run authentication tests
	go test ./tests/examples/unit/ -run TestUserService -v
	go test ./tests/examples/integration/ -run TestAuthHandler -v

# Code quality commands
lint: ## Run golangci-lint
	golangci-lint run

format: ## Format code using gofmt
	go fmt ./...

vet: ## Run go vet
	go vet ./...

mod-tidy: ## Tidy module dependencies
	go mod tidy

mod-download: ## Download module dependencies
	go mod download

# Database commands
migrate: ## Run database migrations (included in app startup)
	@echo "Migrations run automatically on app startup"

db-seed: ## Seed database with test data (for development)
	@echo "Seeding database..."
	@echo "Note: Implement seeding logic in your application if needed"

# Docker commands
docker-build: ## Build Docker image
	docker build -t gofiber-template .

docker-run: ## Run application in Docker container
	docker-compose up --build

docker-stop: ## Stop Docker containers
	docker-compose down

docker-logs: ## Show Docker container logs
	docker-compose logs -f

# Monitoring and profiling
profile-cpu: ## Run CPU profiling
	go test -cpuprofile cpu.prof -bench=. ./...
	go tool pprof cpu.prof

profile-mem: ## Run memory profiling
	go test -memprofile mem.prof -bench=. ./...
	go tool pprof mem.prof

# Security commands
security-check: ## Run security checks using gosec
	gosec ./...

# Documentation commands
docs: ## Generate documentation
	godoc -http=:6060
	@echo "Documentation available at http://localhost:6060"

# Example test commands
example-user-test: ## Run user service test examples
	@echo "Running user service tests..."
	go test ./tests/examples/unit/ -run TestUserService -v

example-integration-test: ## Run integration test examples
	@echo "Running integration tests..."
	go test ./tests/examples/integration/ -v

example-websocket-test: ## Run WebSocket test examples
	@echo "Running WebSocket tests..."
	go test ./tests/examples/integration/ -run TestWebSocket -v

# Performance testing
load-test: ## Run basic load test (requires hey tool)
	@echo "Running load test on /health endpoint..."
	@echo "Install hey first: go install github.com/rakyll/hey@latest"
	hey -n 1000 -c 10 http://localhost:3000/health

stress-test: ## Run stress test (requires hey tool)
	@echo "Running stress test..."
	hey -n 10000 -c 100 -t 30 http://localhost:3000/api/v1/health

# CI/CD helpers
ci-test: ## Run tests in CI environment
	go test -v -race -coverprofile=coverage.out ./...

ci-lint: ## Run linting in CI environment
	golangci-lint run --timeout 5m

ci-build: ## Build for CI environment
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/api cmd/api/main.go

# Utility commands
check-tools: ## Check if required tools are installed
	@echo "Checking required tools..."
	@command -v go >/dev/null 2>&1 || { echo "Go is not installed"; exit 1; }
	@command -v air >/dev/null 2>&1 || { echo "Air is not installed. Install with: go install github.com/cosmtrek/air@latest"; }
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint is not installed. Install from: https://golangci-lint.run/usage/install/"; }
	@echo "All required tools are available"

install-tools: ## Install development tools
	@echo "Installing development tools..."
	go install github.com/cosmtrek/air@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/rakyll/hey@latest
	@echo "Development tools installed"

# Environment setup
setup-env: ## Setup development environment
	@echo "Setting up development environment..."
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo "Created .env file from .env.example"; \
		echo "Please update .env with your configuration"; \
	fi
	make install-tools
	make mod-download
	@echo "Development environment setup complete"

# Quick start command
start: setup-env ## Quick start - setup environment and run the app
	@echo "Starting GoFiber Template..."
	make dev

# Test data commands
test-create-user: ## Create a test user via API (requires running server)
	curl -X POST http://localhost:3000/api/v1/auth/register \
		-H "Content-Type: application/json" \
		-d '{"email":"test@example.com","username":"testuser","password":"password123","first_name":"Test","last_name":"User"}'

test-login: ## Login with test user (requires running server)
	curl -X POST http://localhost:3000/api/v1/auth/login \
		-H "Content-Type: application/json" \
		-d '{"email":"test@example.com","password":"password123"}'

# Project statistics
stats: ## Show project statistics
	@echo "Project Statistics:"
	@echo "=================="
	@echo "Go files: $$(find . -name '*.go' | wc -l)"
	@echo "Lines of code: $$(find . -name '*.go' -exec wc -l {} + | tail -1)"
	@echo "Test files: $$(find . -name '*_test.go' | wc -l)"
	@echo "Packages: $$(go list ./... | wc -l)"
	@echo "Dependencies: $$(go list -m all | wc -l)"

# Default goal
.DEFAULT_GOAL := help