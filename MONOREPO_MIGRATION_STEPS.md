# Monorepo Migration Implementation Steps

## Before You Start

**Prerequisites:**
- Backup current repository
- All changes committed to git
- Review MONOREPO_RESTRUCTURE.md

**Estimated time:** 4-6 hours

## Step 1: Create Directory Structure

```bash
# Create base directories
mkdir -p services/{core,gateway,k8s-manager,dashboard}
mkdir -p infrastructure/{helm,pulumi}
mkdir -p libs/{common,types}
mkdir -p tests/{integration,e2e,scenarios}
mkdir -p docs/troubleshooting

# Create config and scripts directories (if not present)
mkdir -p config scripts
```

## Step 2: Migrate Core Service

### 2.1 Move Core Source Code

```bash
# Move core commands
mv cmd/worker services/core/cmd/
mv cmd/fc-agent services/core/cmd/
mv cmd/migrate services/core/cmd/
mv cmd/aether-cli services/core/cmd/cli

# Move core packages
mkdir -p services/core/pkg
mv pkg/vmm services/core/pkg/
mv pkg/network services/core/pkg/
mv pkg/tools services/core/pkg/
mv pkg/storage services/core/pkg/
mv pkg/queue services/core/pkg/
mv pkg/types services/core/pkg/

# Move core tests
mkdir -p services/core/tests
# Move any core-specific tests here

# Move migrations
mv migrations services/core/

# Copy config example
cp config/config.yaml.example services/core/config/
```

### 2.2 Create Core Service go.mod

Create `services/core/go.mod`:

```go
module github.com/aetherium/aetherium/services/core

go 1.25

require (
    github.com/creack/pty v1.1.24
    github.com/firecracker-microvm/firecracker-go-sdk v1.0.0
    github.com/go-chi/chi/v5 v5.2.3
    github.com/golang-migrate/migrate/v4 v4.19.0
    github.com/google/uuid v1.6.0
    github.com/hibiken/asynq v0.25.1
    github.com/jmoiron/sqlx v1.4.0
    github.com/lib/pq v1.10.9
    github.com/mdlayher/vsock v1.2.1
    github.com/redis/go-redis/v9 v9.7.0
    github.com/stretchr/testify v1.11.1
    gopkg.in/yaml.v3 v3.0.1
    github.com/aetherium/aetherium/libs/common v0.0.0
    github.com/aetherium/aetherium/libs/types v0.0.0
)

replace (
    github.com/aetherium/aetherium/libs/common => ../../libs/common
    github.com/aetherium/aetherium/libs/types => ../../libs/types
)
```

### 2.3 Create Core Service Makefile

Create `services/core/Makefile`:

```makefile
.PHONY: all build clean test lint fmt help

GO := go
BINARY_DIR := ../../bin
SERVICE_DIR := $(shell pwd)

all: build

help:
	@echo "Core Service Build System"
	@echo ""
	@echo "Usage:"
	@echo "  make build          - Build all core binaries"
	@echo "  make test           - Run tests"
	@echo "  make lint           - Run linters"
	@echo "  make clean          - Clean build artifacts"

build:
	@echo "Building core service binaries..."
	@mkdir -p $(BINARY_DIR)
	$(GO) build -o $(BINARY_DIR)/worker ./cmd/worker
	$(GO) build -o $(BINARY_DIR)/fc-agent ./cmd/fc-agent
	$(GO) build -o $(BINARY_DIR)/aether-cli ./cmd/cli
	$(GO) build -o $(BINARY_DIR)/migrate ./cmd/migrate
	@echo "Build complete!"

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
```

### 2.4 Create Core Service README

Create `services/core/README.md`:

```markdown
# Aetherium Core Service

VM provisioning and lifecycle management using Firecracker.

## Features

- **VM Orchestration**: Firecracker and Docker backends
- **Command Execution**: Execute commands in VMs via vsock
- **Tool Installation**: Automatic tool provisioning with mise
- **Network Management**: TAP devices, bridge setup, NAT
- **Persistent Storage**: PostgreSQL state storage
- **Task Queue**: Redis/Asynq task distribution
- **CLI Tools**: Command-line interface for operations

## Building

```bash
make build
```

Binaries in `../../bin/`:
- `worker` - Main task worker daemon
- `fc-agent` - VM agent (runs inside VM)
- `aether-cli` - CLI tool
- `migrate` - Database migration tool

## Testing

```bash
make test
make test-coverage
```

## Running

### Local Development

```bash
# Start infrastructure
make docker-up

# Run migrations
make migrate-up

# Start worker (requires sudo for Firecracker)
sudo make run-worker
```

### Production

See [../../docs/DEPLOYMENT.md](../../docs/DEPLOYMENT.md)

## API

See [pkg/service/task_service.go](pkg/service/task_service.go) for public API.

## Architecture

See [pkg/README.md](pkg/README.md)
```

## Step 3: Migrate Gateway Service

### 3.1 Move Gateway Source Code

```bash
# Move gateway command
mv cmd/api-gateway services/gateway/cmd/

# Move gateway packages
mkdir -p services/gateway/pkg
mv pkg/api services/gateway/pkg/
# Keep pkg/integrations for now, will be in gateway
mv pkg/integrations services/gateway/pkg/
```

### 3.2 Create Gateway go.mod

Create `services/gateway/go.mod`:

```go
module github.com/aetherium/aetherium/services/gateway

go 1.25

require (
    github.com/go-chi/chi/v5 v5.2.3
    github.com/go-chi/cors v1.2.2
    github.com/google/uuid v1.6.0
    github.com/gorilla/websocket v1.5.3
    github.com/redis/go-redis/v9 v9.7.0
    gopkg.in/yaml.v3 v3.0.1
    github.com/aetherium/aetherium/libs/common v0.0.0
    github.com/aetherium/aetherium/libs/types v0.0.0
    github.com/aetherium/aetherium/services/core v0.0.0
)

replace (
    github.com/aetherium/aetherium/libs/common => ../../libs/common
    github.com/aetherium/aetherium/libs/types => ../../libs/types
    github.com/aetherium/aetherium/services/core => ../core
)
```

### 3.3 Create Gateway Makefile

Create `services/gateway/Makefile`:

```makefile
.PHONY: all build clean test lint fmt help run

GO := go
BINARY_DIR := ../../bin

all: build

help:
	@echo "Gateway Service Build System"
	@echo ""
	@echo "Usage:"
	@echo "  make build    - Build gateway"
	@echo "  make test     - Run tests"
	@echo "  make run      - Run gateway locally"
	@echo "  make lint     - Run linters"
	@echo "  make clean    - Clean build artifacts"

build:
	@echo "Building gateway service..."
	@mkdir -p $(BINARY_DIR)
	$(GO) build -o $(BINARY_DIR)/api-gateway ./cmd/api-gateway
	@echo "Build complete!"

run: build
	./$(BINARY_DIR)/api-gateway

test:
	$(GO) test -v -race ./...

lint:
	golangci-lint run ./...

fmt:
	$(GO) fmt ./...

clean:
	$(GO) clean

deps:
	$(GO) mod download
	$(GO) mod tidy
```

## Step 4: Create K8s Manager Service (Placeholder)

### 4.1 Create K8s Manager Structure

Create `services/k8s-manager/go.mod`:

```go
module github.com/aetherium/aetherium/services/k8s-manager

go 1.25

require (
    k8s.io/client-go v0.29.0
    k8s.io/api v0.29.0
    k8s.io/apimachinery v0.29.0
    gopkg.in/yaml.v3 v3.0.1
    github.com/aetherium/aetherium/libs/common v0.0.0
    github.com/aetherium/aetherium/libs/types v0.0.0
    github.com/aetherium/aetherium/services/core v0.0.0
)

replace (
    github.com/aetherium/aetherium/libs/common => ../../libs/common
    github.com/aetherium/aetherium/libs/types => ../../libs/types
    github.com/aetherium/aetherium/services/core => ../core
)
```

Create `services/k8s-manager/README.md`:

```markdown
# Aetherium K8s Manager Service

Kubernetes pod lifecycle management and orchestration.

## Status

🚧 In Development

## Architecture

- Pod creation and deployment strategies
- Auto-scaling configuration
- Resource management
- Health monitoring
```

## Step 5: Move Dashboard

### 5.1 Move Dashboard to Services

```bash
# The dashboard should already be mostly isolated
# Just ensure it's in services/dashboard/
# Update any API client URLs to use the gateway service
```

Update `services/dashboard/README.md` to reference new structure.

## Step 6: Move Infrastructure Code

### 6.1 Move Helm Charts

```bash
mv helm/aetherium infrastructure/helm/
```

### 6.2 Move Pulumi Code

```bash
# Reorganize pulumi code
mv pulumi/* infrastructure/pulumi/core/
```

Create `infrastructure/pulumi/core/README.md`:

```markdown
# Aetherium Pulumi IaC

Infrastructure as Code for Aetherium deployment.

## Stacks

- **dev**: Development environment
- **prod**: Production environment

## Deployment

```bash
pulumi stack select dev
pulumi up
```

See [../../docs/DEPLOYMENT.md](../../docs/DEPLOYMENT.md) for details.
```

## Step 7: Create Shared Libraries

### 7.1 Create Common Library

Create `libs/common/go.mod`:

```go
module github.com/aetherium/aetherium/libs/common

go 1.25

require (
    gopkg.in/yaml.v3 v3.0.1
)
```

```bash
# Move shared code to libs/common
mkdir -p libs/common/pkg
mv pkg/logging libs/common/pkg/
mv pkg/config libs/common/pkg/
mv pkg/container libs/common/pkg/
mv pkg/events libs/common/pkg/
```

Create `libs/common/README.md`:

```markdown
# Common Libraries

Shared utilities used across services.

## Packages

- **logging**: Logging abstractions (Loki, stdout)
- **config**: YAML configuration management
- **container**: Dependency injection container
- **events**: Event bus abstractions
```

### 7.2 Create Types Library

Create `libs/types/go.mod`:

```go
module github.com/aetherium/aetherium/libs/types

go 1.25

require (
    github.com/google/uuid v1.6.0
    gopkg.in/yaml.v3 v3.0.1
)
```

```bash
# Move shared types
mkdir -p libs/types/pkg/{api,domain,events}

# Move API types
mv pkg/api/models.go libs/types/pkg/api/

# Move domain types
# (Keep in individual services or move to here if truly shared)

# Move event types
mkdir -p libs/types/pkg/events
# Create event type definitions here
```

## Step 8: Update Root go.work

Create `go.work` in repository root:

```
go 1.25

use (
    ./libs/common
    ./libs/types
    ./services/core
    ./services/gateway
    ./services/k8s-manager
)
```

## Step 9: Create Root Makefile

Create comprehensive root `Makefile`:

```makefile
.PHONY: help build test lint fmt clean docker-up docker-down deps install

help:
	@echo "Aetherium Monorepo Build System"
	@echo ""
	@echo "Usage:"
	@echo "  make build              - Build all services"
	@echo "  make build-core         - Build core service only"
	@echo "  make build-gateway      - Build gateway service only"
	@echo "  make build-k8s-manager  - Build k8s-manager service only"
	@echo "  make build-dashboard    - Build dashboard only"
	@echo ""
	@echo "  make test               - Test all services"
	@echo "  make test-core          - Test core service only"
	@echo "  make test-coverage      - Generate coverage reports"
	@echo ""
	@echo "  make lint               - Lint all services"
	@echo "  make fmt                - Format all code"
	@echo ""
	@echo "  make docker-build       - Build all Docker images"
	@echo "  make docker-up          - Start infrastructure"
	@echo "  make docker-down        - Stop infrastructure"
	@echo ""
	@echo "  make deps               - Download all dependencies"
	@echo "  make clean              - Clean all build artifacts"

# Building
build:
	cd services/core && make build
	cd services/gateway && make build
	cd services/k8s-manager && make build
	cd services/dashboard && make build
	@echo "All services built successfully!"

build-core:
	cd services/core && make build

build-gateway:
	cd services/gateway && make build

build-k8s-manager:
	cd services/k8s-manager && make build

build-dashboard:
	cd services/dashboard && make build

# Testing
test:
	cd services/core && make test
	cd services/gateway && make test
	cd services/k8s-manager && make test

test-core:
	cd services/core && make test

test-coverage:
	cd services/core && make test-coverage
	cd services/gateway && make test-coverage
	cd services/k8s-manager && make test-coverage

# Code Quality
lint:
	cd services/core && make lint
	cd services/gateway && make lint
	cd services/k8s-manager && make lint

fmt:
	cd services/core && make fmt
	cd services/gateway && make fmt
	cd services/k8s-manager && make fmt
	cd libs/common && make fmt
	cd libs/types && make fmt

# Dependencies
deps:
	go work sync
	cd services/core && go mod download
	cd services/gateway && go mod download
	cd services/k8s-manager && go mod download
	cd libs/common && go mod download
	cd libs/types && go mod download

# Docker
docker-build:
	docker-compose -f docker-compose.yml build

docker-up:
	docker-compose -f docker-compose.yml up -d

docker-down:
	docker-compose -f docker-compose.yml down

# Cleanup
clean:
	cd services/core && make clean
	cd services/gateway && make clean
	cd services/k8s-manager && make clean
	rm -f coverage.out coverage.html
	go clean
```

## Step 10: Update Imports

Update import paths in all Go files:

```bash
# Old: github.com/aetherium/aetherium/pkg/vmm
# New: github.com/aetherium/aetherium/services/core/pkg/vmm

# Old: github.com/aetherium/aetherium/pkg/logging
# New: github.com/aetherium/aetherium/libs/common/pkg/logging
```

Use `grep -r` to find and update imports systematically.

## Step 11: Update Documentation

1. Update `README.md` to reference new structure
2. Create `docs/SERVICES.md` describing each service
3. Create `docs/DEVELOPMENT.md` with monorepo development guide
4. Create `docs/ARCHITECTURE.md` with updated architecture

## Step 12: Update CI/CD

Create `.github/workflows/`:

1. **core.yml** - Build and test core service
2. **gateway.yml** - Build and test gateway service
3. **k8s-manager.yml** - Build and test k8s-manager service
4. **dashboard.yml** - Build and test dashboard
5. **integration.yml** - Run integration tests across services
6. **infrastructure.yml** - Validate Helm and Pulumi

## Step 13: Verify Structure

```bash
# Verify Go workspaces work
go mod graph

# Verify each service builds independently
cd services/core && go build ./...
cd services/gateway && go build ./...

# Verify imports are correct
go mod tidy
go work sync

# Run tests
make test
```

## Step 14: Cleanup Old Structure

```bash
# After verification, remove old directories
rm -rf pkg cmd  # OLD pkg and cmd at root level
```

## Rollback Plan

If something breaks:

```bash
git reset --hard HEAD~N  # Go back N commits
# or
git reflog  # Find specific commit and reset
```

## Post-Migration

1. Update local development documentation
2. Train team on new structure
3. Update CI/CD pipelines
4. Update deployment scripts
5. Monitor for any issues
6. Document lessons learned
