# Define variables for Docker Compose file configurations
BASE_COMPOSE_FILE_FLAG := -f docker-compose.yml
DEV_COMPOSE_FILES := $(BASE_COMPOSE_FILE_FLAG) -f docker-compose.override.yml
PROD_COMPOSE_FILES := $(BASE_COMPOSE_FILE_FLAG) -f docker-compose.prod.yml

# Variables for executing commands in containers
APP_EXEC := docker compose exec app
DB_EXEC := docker compose exec db

# Ensure all non-file targets are declared .PHONY for clarity and correctness
.PHONY: help \
        init \
        dev dev-build dev-simple dev-clean \
        debug debug-clean \
        prod prod-build \
        build build-no-cache \
        db-migrate db-seed db-reset db-shell \
        test test-verbose test-coverage \
        lint format \
        logs logs-db logs-nginx \
        shell \
        clean clean-all \
        deps deps-update \
        status info

# Default target
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z0-9_-]+:.*##/ {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# --- Setup commands ---
init: ## Initialize project (create .env if it doesn't exist)
	@if [ ! -f .env ]; then \
		echo "Creating .env file from .env.example..."; \
		cp .env.example .env; \
		echo "✅ .env file created. Please edit it with your configuration."; \
	else \
		echo "ℹ️ .env file already exists."; \
	fi

# --- Development commands ---
dev: init ## Start development environment with Air hot reloading
	docker compose $(DEV_COMPOSE_FILES) up --watch

dev-build: init ## Build and start development environment with Air
	docker compose $(DEV_COMPOSE_FILES) up --build --watch

dev-simple: init ## Start development with basic docker compose watch (no Air, no override)
	@echo "Starting simple development mode (docker compose watch, uses only docker-compose.yml)..."
	docker compose $(BASE_COMPOSE_FILE_FLAG) up --watch

dev-clean: ## Clean, build, and start development environment
	docker compose $(DEV_COMPOSE_FILES) down --volumes --remove-orphans
	docker compose $(DEV_COMPOSE_FILES) up --build --watch

# --- Debug commands ---
debug: init ## Start debug environment with Delve debugger
	docker compose -f docker-compose.yml -f docker-compose.debug.yml up --build --watch

debug-clean: ## Clean, build, and start debug environment  
	docker compose -f docker-compose.yml -f docker-compose.debug.yml down --volumes --remove-orphans
	docker compose -f docker-compose.yml -f docker-compose.debug.yml up --build --watch

# --- DevContainer commands ---
devcontainer-test: ## Test devcontainer internet connectivity
	@echo "🌐 Testing internet connectivity..."
	@curl -s --connect-timeout 5 https://github.com && echo "✅ GitHub accessible" || echo "❌ GitHub not accessible"
	@curl -s --connect-timeout 5 https://proxy.golang.org && echo "✅ Go proxy accessible" || echo "❌ Go proxy not accessible"
	@echo "🐹 Testing Go tools..."
	@go version
	@gopls version
	@echo "✅ DevContainer connectivity test complete"

down: ## Stop development environment
	docker compose $(DEV_COMPOSE_FILES) down --remove-orphans

# --- Production commands ---
prod: ## Start production environment
	docker compose $(PROD_COMPOSE_FILES) up -d

prod-build: ## Build and start production environment
	docker compose $(PROD_COMPOSE_FILES) up --build -d

# --- Build commands ---
# 'docker compose build' by default uses docker-compose.yml and docker-compose.override.yml if present
build: ## Build the application services (respects override by default)
	docker compose build

build-no-cache: ## Build the application services without cache (respects override by default)
	docker compose build --no-cache

# --- Database commands ---
db-migrate: ## Run database migrations
	$(APP_EXEC) go run cmd/migrate/main.go

db-seed: ## Seed the database with sample data
	$(APP_EXEC) go run cmd/seed/main.go

db-reset: ## Reset database (drop, create, migrate, seed)
	$(APP_EXEC) go run cmd/reset/main.go

# --- Testing commands ---
test: ## Run tests
	$(APP_EXEC) go test ./...

test-verbose: ## Run tests with verbose output
	$(APP_EXEC) go test -v ./...

test-coverage: ## Run tests and generate HTML coverage report
	$(APP_EXEC) go test -coverprofile=coverage.out ./...
	$(APP_EXEC) go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# --- Code quality commands ---
lint: ## Run linter
	$(APP_EXEC) golangci-lint run

format: ## Format Go code
	$(APP_EXEC) gofmt -s -w .
	$(APP_EXEC) goimports -w .

# --- Utility commands ---
logs: ## Show application logs (app service)
	docker compose logs -f app

logs-db: ## Show database logs (db service)
	docker compose logs -f db

logs-nginx: ## Show nginx logs (nginx service)
	docker compose logs -f nginx

shell: ## Open shell in app container
	$(APP_EXEC) sh

db-shell: ## Open database shell (reads .env for credentials)
	@if [ ! -f .env ]; then \
		echo "⚠️ Error: .env file not found. Please run 'make init' and configure it."; \
		exit 1; \
	fi
	$(DB_EXEC) psql -U $$(grep POSTGRES_USER .env | cut -d '=' -f2) -d $$(grep POSTGRES_DB .env | cut -d '=' -f2)

# --- Cleanup commands ---
# 'docker compose down' by default uses docker-compose.yml and docker-compose.override.yml if present
clean: ## Stop and remove containers
	docker compose down --remove-orphans

clean-all: ## Stop containers, remove volumes, and clean build cache
	docker compose down --volumes --remove-orphans
	docker volume rm go-mod-cache go-build-cache 2>/dev/null || true
	docker volume rm strfkr_go-mod-cache strfkr_go-build-cache 2>/dev/null || true
	docker system prune -f

# --- Dependency management ---
deps: ## Download Go dependencies
	$(APP_EXEC) go mod download
	$(APP_EXEC) go mod tidy

deps-update: ## Update Go dependencies to latest versions
	$(APP_EXEC) go get -u ./...
	$(APP_EXEC) go mod tidy

# --- Status and info ---
status: ## Show container status
	docker compose ps

info: ## Show system information (Docker versions, container status)
	@echo "--- System Information ---"
	@echo "Docker version:"
	@docker --version
	@echo "\nDocker Compose version:"
	@docker compose version
	@echo "\nContainers status:"
	@docker compose ps
	@echo "-------------------------"