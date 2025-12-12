.PHONY: all build clean test lint fmt help

GO := go
BINARY_DIR := ../../bin
SERVICE := k8s-manager

all: build

help:
	@echo "$(SERVICE) Service Build System"
	@echo ""
	@echo "Usage:"
	@echo "  make build    - Build k8s-manager binary"
	@echo "  make test     - Run tests"
	@echo "  make lint     - Run linters"
	@echo "  make fmt      - Format code"
	@echo "  make clean    - Clean build artifacts"
	@echo ""

build:
	@echo "Building $(SERVICE) service (placeholder)..."
	@mkdir -p $(BINARY_DIR)
	@echo "ℹ️  K8s manager service is in development"

test:
	@echo "No tests yet for $(SERVICE)"

lint:
	golangci-lint run ./...

fmt:
	$(GO) fmt ./...

clean:
	$(GO) clean

deps:
	$(GO) mod download
	$(GO) mod tidy
