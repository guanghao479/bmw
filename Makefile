.PHONY: help dev dev-backend dev-frontend build test clean

# Default target
help: ## Show this help message
	@echo "Seattle Family Activities - Development Commands"
	@echo "==============================================="
	@echo ""
	@echo "Available commands:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

dev: ## Start both backend and frontend servers
	@./scripts/dev

dev-backend: ## Start only the backend server
	@./scripts/dev-backend

dev-frontend: ## Start only the frontend server  
	@./scripts/dev-frontend

build: ## Build Lambda functions
	@echo "🔨 Building Lambda functions..."
	@cd backend && mkdir -p ../testing/bin
	@cd backend && go build -o ../testing/bin/admin_api ./cmd/admin_api
	@cd backend && go build -o ../testing/bin/scraping_orchestrator ./cmd/scraping_orchestrator
	@echo "✅ Build complete"

test: ## Run backend tests
	@echo "🧪 Running backend tests..."
	@cd backend && go test ./internal/models -v
	@echo "✅ Tests complete"

test-integration: ## Run integration tests (requires API keys)
	@echo "🧪 Running integration tests..."
	@cd backend && ./scripts/run_integration_tests.sh

clean: ## Clean build artifacts
	@echo "🧹 Cleaning build artifacts..."
	@rm -rf testing/bin/
	@rm -f backend/env.json
	@echo "✅ Clean complete"

infra-deploy: ## Deploy infrastructure
	@echo "🚀 Deploying infrastructure..."
	@cd infrastructure && npm run deploy

infra-diff: ## Show infrastructure deployment diff
	@echo "📋 Showing infrastructure deployment diff..."
	@cd infrastructure && npm run diff