.PHONY: all build clean test lint help go-build

# Variables
GO := go
BINARY_DIR := bin

all: build

help:
	@echo "Aetherium Build System"
	@echo ""
	@echo "Usage:"
	@echo "  make build          - Build all Go services"
	@echo "  make test           - Run tests"
	@echo "  make lint           - Run linters"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make run-gateway    - Run API Gateway"
	@echo "  make run-worker     - Run Agent Worker"

# Build Go services
go-build:
	@echo "Building Go services..."
	@mkdir -p $(BINARY_DIR)
	$(GO) build -o $(BINARY_DIR)/api-gateway ./services/gateway/cmd/api-gateway
	$(GO) build -o $(BINARY_DIR)/worker ./services/core/cmd/worker
	$(GO) build -o $(BINARY_DIR)/aether-cli ./services/core/cmd/cli
	$(GO) build -o $(BINARY_DIR)/fc-agent ./services/core/cmd/fc-agent
	$(GO) build -o $(BINARY_DIR)/migrate ./services/core/cmd/migrate

build: go-build
	@echo "Build complete!"

# Run services
run-gateway: build
	./$(BINARY_DIR)/api-gateway

run-worker: build
	sudo ./$(BINARY_DIR)/worker

# Testing
test:
	$(GO) test -v -race ./libs/common/... ./libs/types/... ./services/core/... ./services/gateway/... ./services/k8s-manager/... ./tests/integration/...

test-coverage:
	$(GO) test -v -race -coverprofile=coverage.out ./libs/common/... ./libs/types/... ./services/core/... ./services/gateway/... ./services/k8s-manager/... ./tests/integration/...
	$(GO) tool cover -html=coverage.out -o coverage.html

# Linting
lint:
	golangci-lint run ./...

fmt:
	$(GO) fmt ./...

# Clean
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BINARY_DIR)
	rm -f coverage.out coverage.html

# Dependencies
deps:
	$(GO) mod download
	$(GO) mod tidy

# Docker
docker-build:
	docker-compose build

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

# Database migrations
migrate-up:
	$(GO) run cmd/migrate/main.go up

migrate-down:
	$(GO) run cmd/migrate/main.go down

migrate-create:
	@read -p "Migration name: " name; \
	$(GO) run cmd/migrate/main.go create $$name
