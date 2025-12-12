# Monorepo Quick Reference

## Quick Start After Migration

```bash
# Clone and setup
git clone https://github.com/aetherium/aetherium.git
cd aetherium

# Install all dependencies
make deps

# Build everything
make build

# Or build specific service
cd services/core
go build ./...
```

## Service Quick Reference

| Service | Path | Language | Purpose | Build |
|---------|------|----------|---------|-------|
| Core | `services/core/` | Go | VM provisioning & lifecycle | `cd services/core && make build` |
| Gateway | `services/gateway/` | Go | REST API & integrations | `cd services/gateway && make build` |
| K8s Manager | `services/k8s-manager/` | Go | Pod lifecycle | `cd services/k8s-manager && make build` |
| Dashboard | `services/dashboard/` | TypeScript/Next.js | Web UI | `cd services/dashboard && make build` |
| Common | `libs/common/` | Go | Shared utilities | (library) |
| Types | `libs/types/` | Go | Shared types | (library) |

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

# From service directory
cd services/core
go build -o ../../bin/worker ./cmd/worker
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

# From service directory
cd services/core
golangci-lint run ./...
```

### Running Locally

```bash
# Terminal 1: Start infrastructure
make docker-up

# Terminal 2: Run core service
cd services/core
sudo ./../../bin/worker

# Terminal 3: Run gateway
cd services/gateway
./../../bin/api-gateway

# Terminal 4: Run dashboard
cd services/dashboard
npm run dev
```

### Managing Dependencies

```bash
# Update all go.mod files
make deps

# Update workspace
go work sync

# Add dependency to specific service
cd services/core
go get github.com/some/package
```

### Docker

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

### Deployment

```bash
# Deploy via Helm
make deploy-helm ENV=dev
make deploy-helm ENV=prod

# Deploy via Pulumi
cd infrastructure/pulumi/core
pulumi up -s dev

# Deploy specific service
helm upgrade aetherium-core ./infrastructure/helm/aetherium -f values.yaml
```

## Directory Lookup Guide

### I need to modify...

**VM provisioning logic?**
→ `services/core/pkg/vmm/` and `services/core/pkg/storage/`

**REST API endpoints?**
→ `services/gateway/pkg/api/`

**Integration plugins (GitHub, Slack)?**
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

**Documentation?**
→ `docs/` and service-specific `README.md` files

## Import Path Changes (Migration Reference)

### Old → New Mappings

```go
// Old: github.com/aetherium/aetherium/pkg/vmm
// New: github.com/aetherium/aetherium/services/core/pkg/vmm
import "github.com/aetherium/aetherium/services/core/pkg/vmm"

// Old: github.com/aetherium/aetherium/pkg/logging
// New: github.com/aetherium/aetherium/libs/common/pkg/logging
import "github.com/aetherium/aetherium/libs/common/pkg/logging"

// Old: github.com/aetherium/aetherium/pkg/api
// New: github.com/aetherium/aetherium/services/gateway/pkg/api
import "github.com/aetherium/aetherium/services/gateway/pkg/api"

// Old: github.com/aetherium/aetherium/pkg/types
// New: github.com/aetherium/aetherium/libs/types/pkg/domain
import "github.com/aetherium/aetherium/libs/types/pkg/domain"
```

## Service Dependencies

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
(only stdlib and third-party)

libs/types
    ↓ imports
(minimal external dependencies)
```

## Environment Variables

### Core Service
```bash
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=aetherium
POSTGRES_PASSWORD=aetherium
POSTGRES_DB=aetherium

REDIS_ADDR=localhost:6379

FIRECRACKER_BIN=/path/to/firecracker
KERNEL_PATH=/path/to/kernel
ROOTFS_PATH=/path/to/rootfs.ext4
```

### Gateway Service
```bash
GATEWAY_PORT=8080
CORE_SERVICE_URL=localhost:50051  # or HTTP API
LOG_LEVEL=info

GITHUB_TOKEN=...
SLACK_BOT_TOKEN=...
```

### Dashboard
```bash
NEXT_PUBLIC_API_URL=http://localhost:8080
NEXT_PUBLIC_WS_URL=ws://localhost:8080
```

## Database Migrations

### Running migrations
```bash
cd services/core
go run ./cmd/migrate/main.go up
```

### Creating new migration
```bash
cd services/core
go run ./cmd/migrate/main.go create add_column_to_vms
```

### Viewing migrations
```bash
ls -la services/core/migrations/
```

## Kubernetes Deployment

### Prerequisites
```bash
kubectl create namespace aetherium
# or
helm repo add aetherium https://charts.aetherium.io
```

### Deploy all services
```bash
helm install aetherium ./infrastructure/helm/aetherium \
  -n aetherium \
  -f infrastructure/helm/aetherium/values.yaml
```

### Deploy specific service
```bash
helm install aetherium-core ./infrastructure/helm/aetherium \
  -n aetherium \
  --set core.enabled=true \
  --set gateway.enabled=false
```

### Upgrade
```bash
helm upgrade aetherium ./infrastructure/helm/aetherium \
  -n aetherium
```

## Troubleshooting

### Service won't build
```bash
# Clean and try again
cd services/{service}
go clean -modcache
go mod tidy
go build ./...
```

### Import errors after migration
```bash
# Update all imports to match new structure
go work sync
cd services/{service}
go mod tidy
```

### Tests failing
```bash
# Run with verbose output
go test -v ./...

# Run specific test
go test -v -run TestFunctionName ./...
```

### Docker issues
```bash
# Start fresh
make docker-down
docker system prune
make docker-up
```

### Database migrations failing
```bash
# Check status
cd services/core
go run ./cmd/migrate/main.go version

# Rollback one
go run ./cmd/migrate/main.go down

# See all migrations
ls migrations/
```

## CI/CD Pipeline

### What gets tested
- On PR: Each service that changed
- On merge to main: All services
- Nightly: Integration tests across services

### View pipeline status
- GitHub: https://github.com/aetherium/aetherium/actions
- Logs: Click on specific workflow run

### Triggering pipeline manually
```bash
git tag release/v1.0.0
git push --tags
```

## Performance Tips

### Local development
```bash
# Use docker-compose for dependencies
make docker-up

# Build only what you changed
cd services/core && make build

# Use hot reload where available
cd services/dashboard && npm run dev
```

### Production deployment
```bash
# Use prebuilt images
docker pull aetherium/core:latest
docker pull aetherium/gateway:latest

# Or build from source
make docker-build
docker push aetherium/core:$(git rev-parse --short HEAD)
```

## Common Issues & Solutions

| Issue | Solution |
|-------|----------|
| `module not found` error | Run `go work sync` then `go mod tidy` in service dir |
| Service can't connect to DB | Check `POSTGRES_*` env vars and `make docker-up` |
| Import cycle detected | Check dependency direction (should be one-way) |
| Tests timeout | Increase timeout: `go test -timeout 10m ./...` |
| Docker can't reach host services | Use `host.docker.internal` on macOS/Windows |
| Helm install fails | Check namespace exists: `kubectl create namespace aetherium` |
| Firecracker permission denied | Run with sudo or grant capabilities |

## Useful Links

- **Documentation**: [docs/README.md](docs/README.md)
- **Architecture**: [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)
- **Development**: [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md)
- **Deployment**: [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md)
- **Contributing**: [docs/CONTRIBUTING.md](docs/CONTRIBUTING.md)

## Getting Help

- Check [docs/troubleshooting/](docs/troubleshooting/)
- Ask in #aetherium Slack channel
- Open issue on GitHub: https://github.com/aetherium/aetherium/issues
