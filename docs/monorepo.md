# Monorepo Guide

Project structure and conventions for the Aetherium monorepo.

## Quick Start

```bash
# Clone and setup
git clone https://github.com/aetherium/aetherium.git
cd aetherium

# Install dependencies
make deps

# Build all services
make build

# Or build specific service
cd services/core && make build
```

## Service Structure

| Service | Path | Language | Purpose |
|---------|------|----------|---------|
| Core | `services/core/` | Go | VM provisioning & lifecycle |
| Gateway | `services/gateway/` | Go | REST API & integrations |
| K8s Manager | `services/k8s-manager/` | Go | Pod lifecycle |
| Dashboard | `services/dashboard/` | TypeScript/Next.js | Web UI |
| Common | `libs/common/` | Go | Shared utilities |
| Types | `libs/types/` | Go | Shared types |

## Common Tasks

### Building

```bash
# All services
make build

# Specific service
make build-core
make build-gateway
make build-k8s-manager
make build-dashboard
```

### Testing

```bash
# All services
make test

# Specific service
make test-core
cd services/gateway && go test ./...

# With coverage
make test-coverage
```

### Code Quality

```bash
# Format all code
make fmt

# Lint all services
make lint
```

### Running Locally

```bash
# Terminal 1: Infrastructure
make docker-up

# Terminal 2: Core service
cd services/core && sudo ./../../bin/worker

# Terminal 3: Gateway
cd services/gateway && ./../../bin/api-gateway

# Terminal 4: Dashboard
cd services/dashboard && npm run dev
```

## Directory Lookup

**VM provisioning logic?**
→ `services/core/pkg/vmm/` and `services/core/pkg/storage/`

**REST API endpoints?**
→ `services/gateway/pkg/api/`

**Integration plugins?**
→ `services/gateway/pkg/integrations/`

**Database schema?**
→ `services/core/migrations/`

**Frontend/Dashboard?**
→ `services/dashboard/src/`

**Kubernetes orchestration?**
→ `services/k8s-manager/pkg/orchestrator/`

**Logging configuration?**
→ `libs/common/pkg/logging/`

**Shared domain types?**
→ `libs/types/pkg/domain/`

**Helm charts?**
→ `infrastructure/helm/`

**Pulumi infrastructure?**
→ `infrastructure/pulumi/core/`

## Import Paths

```go
// Core service
import "github.com/aetherium/aetherium/services/core/pkg/vmm"

// Shared logging
import "github.com/aetherium/aetherium/libs/common/pkg/logging"

// Shared types
import "github.com/aetherium/aetherium/libs/types/pkg/domain"

// Gateway API
import "github.com/aetherium/aetherium/services/gateway/pkg/api"
```

## Dependencies

```
services/gateway
    ↓ imports
services/core, libs/common, libs/types

services/k8s-manager
    ↓ imports
services/core, libs/common, libs/types

services/core
    ↓ imports
libs/common, libs/types

libs/common
    ↓ imports
(stdlib and third-party only)

libs/types
    ↓ imports
(minimal external dependencies)
```

## Managing Dependencies

```bash
# Update all go.mod files
make deps

# Update workspace
go work sync

# Add dependency to service
cd services/core
go get github.com/some/package
```

## Database Migrations

```bash
# Run migrations
cd services/core
go run ./cmd/migrate/main.go up

# Create new migration
go run ./cmd/migrate/main.go create add_column_to_vms

# View migrations
ls -la services/core/migrations/
```

## Docker Operations

```bash
# Build all images
make docker-build

# Start infrastructure
make docker-up

# Stop infrastructure
make docker-down

# View logs
docker-compose logs -f
```

## Deployment

```bash
# Deploy via Helm
make deploy-helm ENV=dev
make deploy-helm ENV=prod

# Deploy via Pulumi
cd infrastructure/pulumi/core
pulumi up -s dev

# Upgrade specific service
helm upgrade aetherium-core ./infrastructure/helm/aetherium -f values.yaml
```

## Troubleshooting

**Service won't build:**
```bash
cd services/{service}
go clean -modcache
go mod tidy
go build ./...
```

**Import errors:**
```bash
go work sync
cd services/{service}
go mod tidy
```

**Tests failing:**
```bash
go test -v ./...
go test -v -run TestFunctionName ./...
```

**Database migrations failing:**
```bash
cd services/core
go run ./cmd/migrate/main.go version
go run ./cmd/migrate/main.go down  # Rollback one
ls migrations/  # See all migrations
```

## CI/CD

- **On PR:** Test services that changed
- **On merge to main:** Test all services
- **Nightly:** Integration tests across services

View status: https://github.com/aetherium/aetherium/actions
