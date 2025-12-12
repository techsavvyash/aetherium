# Aetherium Monorepo Restructuring Plan

## Executive Summary

The current project structure is scattered across Go packages (`/pkg`, `/cmd`), separate frontend (`/dashboard`), and infrastructure code (`/helm`, `/pulumi`). This document outlines a reorganization into a proper monorepo with clear boundaries and ownership.

## Current Problems

1. **No clear separation of concerns**: Core VM provisioning mixed with gateway and dashboard code
2. **Tangled dependencies**: All services use the same `pkg/` namespace
3. **Deployment complexity**: Unclear which services need to be built/deployed together
4. **Scalability issues**: Adding Kubernetes lifecycle management is unclear where it belongs
5. **Build/Test isolation**: Can't selectively build/test individual services
6. **Infrastructure code**: Helm/Pulumi scripts are loosely integrated

## Proposed Monorepo Structure

```
aetherium/
│
├── README.md                           # Root monorepo documentation
├── Makefile                            # Root build orchestration
├── go.work                             # Go workspaces for selective builds
├── workspace.json                      # Workspace configuration
│
├── .github/
│   └── workflows/
│       ├── core.yml                   # Core service CI/CD
│       ├── gateway.yml                # Gateway service CI/CD
│       ├── dashboard.yml              # Dashboard CI/CD
│       ├── k8s.yml                    # K8s manager CI/CD
│       └── infrastructure.yml         # Helm/Pulumi CI/CD
│
├── services/                          # Microservices layer
│   │
│   ├── core/                          # Core VM provisioning (Firecracker + management)
│   │   ├── cmd/
│   │   │   ├── worker/               # Worker daemon
│   │   │   ├── fc-agent/             # Firecracker agent
│   │   │   ├── migrate/              # DB migrations
│   │   │   └── cli/                  # CLI tool
│   │   ├── pkg/
│   │   │   ├── vmm/                  # VM orchestration
│   │   │   │   ├── interface.go
│   │   │   │   ├── firecracker/
│   │   │   │   └── docker/
│   │   │   ├── network/              # Network management (TAP, bridge)
│   │   │   ├── tools/                # Tool installation
│   │   │   ├── storage/              # Database layer
│   │   │   ├── queue/                # Task queue abstractions
│   │   │   └── types/                # Domain types
│   │   ├── internal/
│   │   ├── migrations/
│   │   ├── tests/
│   │   ├── Makefile
│   │   ├── go.mod
│   │   └── README.md
│   │
│   ├── gateway/                      # REST API Gateway
│   │   ├── cmd/
│   │   │   └── api-gateway/
│   │   ├── pkg/
│   │   │   ├── api/                  # REST handlers
│   │   │   ├── middleware/           # HTTP middleware
│   │   │   ├── auth/                 # Authentication
│   │   │   ├── integrations/         # GitHub, Slack plugins
│   │   │   └── types/                # API types
│   │   ├── tests/
│   │   ├── Makefile
│   │   ├── go.mod
│   │   └── README.md
│   │
│   ├── k8s-manager/                  # Kubernetes pod lifecycle manager
│   │   ├── cmd/
│   │   │   └── k8s-manager/
│   │   ├── pkg/
│   │   │   ├── orchestrator/         # K8s orchestration
│   │   │   ├── podlifecycle/         # Pod management
│   │   │   ├── deployment/           # Deployment strategies
│   │   │   └── scaling/              # Auto-scaling logic
│   │   ├── tests/
│   │   ├── Makefile
│   │   ├── go.mod
│   │   └── README.md
│   │
│   └── dashboard/                    # Frontend (Next.js)
│       ├── app/
│       ├── components/
│       ├── lib/
│       ├── public/
│       ├── src/
│       ├── package.json
│       ├── next.config.ts
│       ├── Makefile
│       └── README.md
│
├── infrastructure/                   # Infrastructure as Code
│   │
│   ├── helm/                        # Helm charts
│   │   ├── aetherium/              # Core chart
│   │   │   ├── templates/
│   │   │   ├── values.yaml
│   │   │   ├── Chart.yaml
│   │   │   └── README.md
│   │   ├── gateway/                # Gateway chart
│   │   ├── k8s-manager/            # K8s manager chart
│   │   ├── dashboard/              # Dashboard chart
│   │   └── README.md
│   │
│   └── pulumi/                      # Pulumi IaC
│       ├── core/                   # Core Pulumi program
│       │   ├── index.ts
│       │   ├── infrastructure.ts
│       │   ├── k8s.ts
│       │   ├── namespace.ts
│       │   ├── node-pools.ts
│       │   ├── bare-metal.ts
│       │   ├── Pulumi.yaml
│       │   ├── Pulumi.dev.yaml
│       │   ├── Pulumi.prod.yaml
│       │   ├── package.json
│       │   ├── tsconfig.json
│       │   └── README.md
│       └── scripts/
│           └── deploy.sh
│
├── libs/                            # Shared libraries (internal)
│   │
│   ├── common/                      # Shared utilities
│   │   ├── pkg/
│   │   │   ├── logging/            # Logging abstractions
│   │   │   ├── config/             # Config management
│   │   │   ├── container/          # DI container
│   │   │   ├── events/             # Event bus abstractions
│   │   │   └── errors/             # Error handling
│   │   ├── Makefile
│   │   ├── go.mod
│   │   └── README.md
│   │
│   └── types/                       # Shared type definitions
│       ├── pkg/
│       │   ├── api/                # API request/response types
│       │   ├── domain/             # Domain models
│       │   └── events/             # Event types
│       ├── Makefile
│       ├── go.mod
│       └── README.md
│
├── config/                          # Global configuration templates
│   ├── config.yaml.example
│   ├── docker-compose.yml
│   └── .env.example
│
├── scripts/                         # Global utility scripts
│   ├── setup.sh                     # One-time setup
│   ├── local-dev.sh                 # Local development setup
│   ├── docker-setup.sh              # Docker environment
│   ├── firecracker-setup.sh         # Firecracker setup
│   └── clean.sh                     # Cleanup
│
├── docs/                            # Global documentation
│   ├── ARCHITECTURE.md              # Overall architecture
│   ├── SERVICES.md                  # Service descriptions
│   ├── CONTRIBUTING.md              # Contribution guide
│   ├── DEPLOYMENT.md                # Deployment guide
│   ├── DEVELOPMENT.md               # Development guide
│   └── troubleshooting/             # Troubleshooting guides
│
├── tests/
│   ├── integration/                 # Cross-service integration tests
│   ├── e2e/                         # End-to-end tests
│   └── scenarios/                   # Test scenarios
│
└── Makefile.root                    # Root Makefile
```

## Service Responsibilities

### 1. **Core Service** (`services/core/`)
**Purpose**: Firecracker VM provisioning and lifecycle management

**Owns**:
- VM creation, starting, stopping, deletion
- Firecracker orchestration
- Docker orchestration (for testing)
- Command execution via vsock
- Tool installation (mise integration)
- Network setup (TAP devices, bridge)
- Database persistence (PostgreSQL)
- Task queue management (Redis/Asynq)
- Worker daemon

**Exports**:
- `TaskService` interface
- `VMOrchestrator` interface
- Domain types (VM, Task, Execution)

**Dependencies**:
- Firecracker SDK
- PostgreSQL driver
- Redis client
- Asynq queue library

### 2. **Gateway Service** (`services/gateway/`)
**Purpose**: REST API and integration framework

**Owns**:
- REST API endpoints (`/api/v1/*`)
- Request routing and validation
- Authentication/Authorization
- Integration plugins (GitHub, Slack, Discord)
- Event publishing
- WebSocket support
- API middleware (CORS, logging, rate limiting)

**Imports**:
- Core service (via shared libs)
- Shared types library

**Dependencies**:
- Chi router
- Gorilla WebSocket
- Integration SDKs (GitHub, Slack, etc.)

### 3. **K8s Manager Service** (`services/k8s-manager/`)
**Purpose**: Kubernetes pod lifecycle management

**Owns**:
- Pod creation and deployment
- Deployment strategies
- Auto-scaling logic
- Resource management
- Pod health monitoring
- Pod-to-VM lifecycle bridging

**Imports**:
- Core service (via shared libs)
- Shared types library

**Dependencies**:
- Kubernetes Go client
- Kubebuilder (for controllers)

### 4. **Dashboard** (`services/dashboard/`)
**Purpose**: Web UI for system management

**Owns**:
- Frontend UI (Next.js/React)
- API client for Gateway service
- Real-time updates (WebSocket)
- User authentication UI
- VM/Pod management UI
- Logs/metrics visualization

**Dependencies**:
- Next.js
- React
- WebSocket client
- API client (fetch/axios)

## Shared Libraries

### `libs/common/`
**Non-service-specific utilities**:
- Logging (stdout, Loki integrations)
- Configuration management (YAML loader)
- DI Container
- Event bus abstractions
- Error handling utilities
- Constants and helpers

### `libs/types/`
**Shared type definitions**:
- API request/response types
- Domain models (used across services)
- Event types

## Build and Dependency Management

### Go Workspaces (`go.work`)
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

This allows:
- Building individual services: `cd services/core && go build`
- Selective testing: `cd services/gateway && go test ./...`
- Shared library updates: Changes propagate automatically

### Root Makefile Targets
```bash
make build              # Build all services
make build-core        # Build core only
make build-gateway     # Build gateway only
make build-k8s-manager # Build k8s-manager only
make build-dashboard   # Build dashboard only

make test              # Test all services
make test-core        # Test core only
make test-integration # Run integration tests

make lint              # Lint all services
make fmt              # Format all code

make docker-build     # Build all Docker images
make docker-up        # Start infrastructure
make docker-down      # Stop infrastructure

make deploy-helm      # Deploy via Helm
make deploy-pulumi    # Deploy via Pulumi
```

## Migration Path

### Phase 1: Preparation
1. Create new directory structure
2. Move services to correct locations
3. Update `go.mod` files for each service
4. Create shared library modules

### Phase 2: Go Refactoring
1. Create `go.work` workspace
2. Update import paths in all services
3. Move shared code to `libs/`
4. Update Makefiles

### Phase 3: CI/CD
1. Create separate GitHub Actions workflows for each service
2. Update deployment pipelines
3. Test selective builds

### Phase 4: Documentation
1. Update README with new structure
2. Create service-specific documentation
3. Update CONTRIBUTING.md with monorepo guidelines

## Benefits of This Structure

✅ **Clear Ownership**: Each service has clear responsibilities  
✅ **Scalability**: Easy to add new services (e.g., metrics, logging)  
✅ **Independent Testing**: Test services in isolation  
✅ **Selective Building**: Build only what changed  
✅ **Deployment Flexibility**: Deploy services independently or together  
✅ **Team Scaling**: Teams can own specific services  
✅ **Shared Resources**: Libraries provide common functionality  
✅ **Infrastructure as Code**: Clear separation of deployment config  

## Development Workflow

### Local Development
```bash
# Install all dependencies
make install

# Start infrastructure (Docker, PostgreSQL, Redis)
make infrastructure-up

# Build and run specific service
cd services/core
go run ./cmd/worker

# In another terminal
cd services/gateway
go run ./cmd/api-gateway

# In another terminal
cd services/dashboard
npm run dev
```

### Testing
```bash
# Test everything
make test

# Test specific service
make test-core

# Test with coverage
make test-coverage

# Integration tests (across services)
make test-integration
```

### Building for Production
```bash
# Build all binaries
make build

# Build all Docker images
make docker-build

# Push images
make docker-push

# Deploy
make deploy-helm ENV=prod
```

## Constraints & Guidelines

1. **No circular dependencies**: Core ↔ Gateway ✗, but Gateway → Core ✓
2. **Shared libs only for common code**: Don't move business logic there
3. **Service independence**: Each service should build/test standalone
4. **API contracts**: Services communicate via well-defined APIs
5. **Configuration**: Service-specific config in `services/*/config/`

## Next Steps

1. Review and approve this structure
2. Create new directory structure incrementally
3. Migrate one service at a time (start with Core)
4. Update CI/CD pipelines
5. Update documentation
