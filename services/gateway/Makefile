.PHONY: all build clean test lint fmt help run

GO := go
BINARY_DIR := ../../bin
SERVICE := gateway

all: build

help:
	@echo "$(SERVICE) Service Build System"
	@echo ""
	@echo "Usage:"
	@echo "  make build    - Build gateway binary"
	@echo "  make test     - Run tests"
	@echo "  make lint     - Run linters"
	@echo "  make fmt      - Format code"
	@echo "  make run      - Run gateway service"
	@echo "  make clean    - Clean build artifacts"
	@echo ""

build:
	@echo "Building $(SERVICE) service..."
	@mkdir -p $(BINARY_DIR)
	$(GO) build -o $(BINARY_DIR)/api-gateway ./cmd/api-gateway
	@echo "✅ Build complete!"

run: build
	@echo "Starting gateway service..."
	$(BINARY_DIR)/api-gateway

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
