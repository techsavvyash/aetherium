.PHONY: all build clean test lint fmt help run-worker run-gateway

GO := go
BINARY_DIR := ../../bin
SERVICE := core

all: build

help:
	@echo "$(SERVICE) Service Build System"
	@echo ""
	@echo "Usage:"
	@echo "  make build          - Build all binaries"
	@echo "  make test           - Run tests"
	@echo "  make lint           - Run linters"
	@echo "  make fmt            - Format code"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make run-worker     - Run worker daemon (requires sudo)"
	@echo ""

build:
	@echo "Building $(SERVICE) service..."
	@mkdir -p $(BINARY_DIR)
	$(GO) build -o $(BINARY_DIR)/worker ./cmd/worker
	$(GO) build -o $(BINARY_DIR)/fc-agent ./cmd/fc-agent
	$(GO) build -o $(BINARY_DIR)/aether-cli ./cmd/cli
	$(GO) build -o $(BINARY_DIR)/migrate ./cmd/migrate
	@echo "✅ Build complete!"

run-worker: build
	@echo "Starting worker daemon..."
	sudo $(BINARY_DIR)/worker

test:
	$(GO) test -v -race ./...

test-coverage:
	$(GO) test -v -race -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

lint:
	golangci-lint run ./...

fmt:
	$(GO) fmt ./...

clean:
	$(GO) clean
	rm -f coverage.out coverage.html

deps:
	$(GO) mod download
	$(GO) mod tidy

migrate-up:
	$(GO) run ./cmd/migrate/main.go up

migrate-down:
	$(GO) run ./cmd/migrate/main.go down
