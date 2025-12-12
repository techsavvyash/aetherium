# Monorepo Structure - Visual Guide

## Directory Tree (Proposed)

```
aetherium/
│
├─ 📦 Services (Independently Deployable Microservices)
│  ├─ core/                        [Core VM Orchestration]
│  │  ├─ cmd/
│  │  │  ├─ worker/               Task worker daemon
│  │  │  ├─ fc-agent/             VM agent (runs inside VM)
│  │  │  ├─ migrate/              Database migrations
│  │  │  └─ cli/                  CLI tool
│  │  ├─ pkg/
│  │  │  ├─ vmm/                  VM orchestration (Firecracker/Docker)
│  │  │  ├─ network/              Network setup (TAP, bridge, NAT)
│  │  │  ├─ storage/              PostgreSQL data layer
│  │  │  ├─ queue/                Task queue abstractions
│  │  │  ├─ tools/                Tool installation (mise)
│  │  │  ├─ service/              Business logic (TaskService)
│  │  │  └─ types/                Core domain types
│  │  ├─ migrations/              SQL schemas
│  │  ├─ tests/
│  │  ├─ go.mod                   Isolated Go module
│  │  ├─ Makefile
│  │  └─ README.md
│  │
│  ├─ gateway/                    [REST API & Integrations]
│  │  ├─ cmd/
│  │  │  └─ api-gateway/          HTTP server (port 8080)
│  │  ├─ pkg/
│  │  │  ├─ api/                  REST handlers (/api/v1/*)
│  │  │  ├─ middleware/           HTTP middleware (CORS, auth, logging)
│  │  │  ├─ auth/                 Authentication/authorization
│  │  │  ├─ integrations/         Plugin system (GitHub, Slack, etc.)
│  │  │  └─ types/                API request/response types
│  │  ├─ tests/
│  │  ├─ go.mod                   Imports: core, libs/common, libs/types
│  │  ├─ Makefile
│  │  └─ README.md
│  │
│  ├─ k8s-manager/                [Kubernetes Pod Lifecycle]
│  │  ├─ cmd/
│  │  │  └─ k8s-manager/          K8s controller
│  │  ├─ pkg/
│  │  │  ├─ orchestrator/         K8s orchestration
│  │  │  ├─ podlifecycle/         Pod creation/deletion/monitoring
│  │  │  ├─ deployment/           Deployment strategies
│  │  │  └─ scaling/              Auto-scaling logic
│  │  ├─ tests/
│  │  ├─ go.mod
│  │  ├─ Makefile
│  │  └─ README.md
│  │
│  └─ dashboard/                  [Frontend Web UI]
│     ├─ app/                     Next.js app router
│     ├─ components/              React components
│     ├─ lib/                     Utilities
│     ├─ public/                  Static assets
│     ├─ src/                     Source code
│     ├─ package.json
│     ├─ next.config.ts
│     ├─ tsconfig.json
│     ├─ Makefile
│     └─ README.md
│
├─ 📚 Shared Libraries (Non-Service-Specific)
│  ├─ common/                     [Shared Utilities]
│  │  ├─ pkg/
│  │  │  ├─ logging/              Logging abstractions (Loki, stdout)
│  │  │  ├─ config/               YAML config management
│  │  │  ├─ container/            Dependency injection
│  │  │  ├─ events/               Event bus abstractions
│  │  │  ├─ errors/               Error handling utilities
│  │  │  └─ constants/            Shared constants
│  │  ├─ go.mod
│  │  ├─ Makefile
│  │  └─ README.md
│  │
│  └─ types/                      [Shared Type Definitions]
│     ├─ pkg/
│     │  ├─ api/                  API request/response types
│     │  ├─ domain/               Domain models (VM, Task, Pod, etc.)
│     │  └─ events/               Event type definitions
│     ├─ go.mod
│     ├─ Makefile
│     └─ README.md
│
├─ 🏗️  Infrastructure (Declarative Deployment)
│  ├─ helm/                       [Kubernetes Helm Charts]
│  │  ├─ aetherium/              Core Helm chart
│  │  │  ├─ templates/           K8s resource templates
│  │  │  ├─ values.yaml          Default values
│  │  │  ├─ values-dev.yaml      Dev environment
│  │  │  ├─ values-prod.yaml     Prod environment
│  │  │  ├─ Chart.yaml           Chart metadata
│  │  │  └─ README.md
│  │  ├─ gateway/
│  │  ├─ k8s-manager/
│  │  ├─ dashboard/
│  │  └─ README.md
│  │
│  └─ pulumi/                     [Infrastructure as Code]
│     ├─ core/
│     │  ├─ index.ts              Main entry point
│     │  ├─ infrastructure.ts     Network, storage, compute
│     │  ├─ k8s.ts                Kubernetes cluster setup
│     │  ├─ namespace.ts          K8s namespace configuration
│     │  ├─ node-pools.ts         Node pool definitions
│     │  ├─ bare-metal.ts         Physical infrastructure
│     │  ├─ Pulumi.yaml           Stack config
│     │  ├─ Pulumi.dev.yaml       Dev settings
│     │  ├─ Pulumi.prod.yaml      Prod settings
│     │  ├─ package.json
│     │  ├─ tsconfig.json
│     │  └─ README.md
│     └─ scripts/
│        └─ deploy.sh             Deployment helper
│
├─ 🧪 Test Suite (Cross-Service)
│  ├─ integration/                Service integration tests
│  │  └─ (test files for service interactions)
│  ├─ e2e/                        End-to-end scenarios
│  │  └─ (test files for full workflows)
│  └─ scenarios/                  Reusable test scenarios
│     └─ (shared test utilities)
│
├─ 📖 Documentation
│  ├─ ARCHITECTURE.md             Overall system design
│  ├─ SERVICES.md                 Service descriptions
│  ├─ DEVELOPMENT.md              Developer guide
│  ├─ DEPLOYMENT.md               Deployment procedures
│  ├─ CONTRIBUTING.md             Contribution guidelines
│  └─ troubleshooting/
│     ├─ common-issues.md
│     ├─ firecracker.md
│     ├─ database.md
│     └─ kubernetes.md
│
├─ ⚙️  Configuration (Templates)
│  ├─ config.yaml.example         Config template
│  ├─ docker-compose.yml          Local infrastructure
│  ├─ .env.example                Environment variables
│  └─ .github/
│     └─ workflows/               CI/CD pipelines
│        ├─ core.yml
│        ├─ gateway.yml
│        ├─ k8s-manager.yml
│        ├─ dashboard.yml
│        ├─ integration.yml
│        └─ infrastructure.yml
│
├─ 🛠️  Scripts
│  ├─ setup.sh                    One-time setup
│  ├─ local-dev.sh                Local development
│  ├─ docker-setup.sh             Docker environment
│  ├─ firecracker-setup.sh        Firecracker setup
│  ├─ clean.sh                    Cleanup
│  └─ deploy.sh                   Deployment helper
│
├─ 📄 Root Configuration
│  ├─ go.work                     Go workspace (enables selective builds)
│  ├── workspace.json             Workspace metadata
│  ├─ Makefile                    Root build orchestration
│  ├─ docker-compose.yml          Local infrastructure services
│  ├─ go.mod                      (ROOT - for workspace only)
│  ├─ go.sum
│  ├─ .gitignore
│  ├─ README.md                   Main project documentation
│  │
│  ├─ 📋 Documentation Files
│  ├─ EVALUATION_SUMMARY.md       This evaluation
│  ├─ MONOREPO_RESTRUCTURE.md    Complete restructuring plan
│  ├─ MONOREPO_MIGRATION_STEPS.md Step-by-step migration
│  ├─ CURRENT_VS_PROPOSED.md      Comparison analysis
│  ├─ MONOREPO_QUICK_REFERENCE.md Quick lookup guide
│  └─ MONOREPO_STRUCTURE_VISUAL.md This document
│
└─ 📊 Meta Files
   ├─ .git/                       Version control
   ├─ .claude/                    Claude configuration
   ├─ .playwright-mcp/            Browser automation
   └─ LICENSE
```

---

## File Count Overview

```
CURRENT STRUCTURE:
├─ Root: 2 modules (go.mod, npm packages)
├─ cmd/: 7 binaries
├─ pkg/: 17 packages (entangled)
├─ dashboard/: 1 Next.js app
├─ helm/: 1 chart
└─ pulumi/: 1 IaC stack
TOTAL: ~150 files, unclear dependencies

PROPOSED STRUCTURE:
├─ services/: 4 modules (core, gateway, k8s, dashboard)
├─ libs/: 2 modules (common, types)
├─ infrastructure/: 2 modules (helm, pulumi)
├─ tests/: Cross-service tests
└─ docs/, config/, scripts/: Support files
TOTAL: ~180 files, crystal clear dependencies
```

---

## Dependency Relationships

### Current (Tangled)
```
    ┌─────────────────┐
    │ Every file can  │
    │ import from     │
    │ ANY pkg/*       │
    │ folder          │
    └──────┬──────────┘
           │
      ┌────┴────┬────────┬──────────┐
      ▼         ▼        ▼          ▼
    cmd/     pkg/     helm/      pulumi/
   (mixing)  (chaos)  (loose)    (loose)
```

### Proposed (Clear)
```
services/dashboard
      │
      └──→ services/gateway
            │
            ├──→ services/core
            │
            ├──→ libs/common
            │
            └──→ libs/types

services/k8s-manager
      │
      ├──→ services/core
      │
      ├──→ libs/common
      │
      └──→ libs/types

services/core
      │
      ├──→ libs/common
      │
      └──→ libs/types

libs/common ──→ (stdlib + third-party only)

libs/types ──→ (stdlib + minimal dependencies)
```

---

## Package Organization

### By Service

```
services/core/                   Build: go build ./...
├─ cmd/worker/                  Runs: ./bin/worker
├─ cmd/fc-agent/                Runs: ./bin/fc-agent
├─ cmd/migrate/                 Runs: ./bin/migrate
├─ cmd/cli/                     Runs: ./bin/aether-cli
├─ pkg/vmm/                     Firecracker/Docker orchestration
├─ pkg/network/                 TAP devices, bridge, NAT
├─ pkg/storage/                 PostgreSQL repository pattern
├─ pkg/queue/                   Task queue abstractions
├─ pkg/tools/                   Tool installation
└─ pkg/service/                 Public API (TaskService)

services/gateway/                Build: go build ./cmd/api-gateway
├─ cmd/api-gateway/             Runs: ./bin/api-gateway (port 8080)
├─ pkg/api/                     REST handlers
├─ pkg/middleware/              HTTP middleware
├─ pkg/auth/                    JWT/OAuth
├─ pkg/integrations/            GitHub, Slack, Discord plugins
└─ pkg/types/                   API models

services/k8s-manager/            Build: go build ./...
├─ cmd/k8s-manager/             Runs: ./bin/k8s-manager
├─ pkg/orchestrator/            K8s client wrappers
├─ pkg/podlifecycle/            Pod CRUD operations
├─ pkg/deployment/              Deployment patterns
└─ pkg/scaling/                 Auto-scaling algorithms

services/dashboard/              Build: npm run build
├─ app/                         Next.js App Router
├─ components/                  React components
├─ lib/                         Utilities, API client
└─ public/                      Assets
```

### By Shared Functionality

```
libs/common/                     No external imports (clean)
├─ pkg/logging/                 Loki, stdout, structured
├─ pkg/config/                  YAML loading, validation
├─ pkg/container/               DI container
├─ pkg/events/                  Event bus interface
└─ pkg/errors/                  Error handling

libs/types/                      Minimal external deps
├─ pkg/api/                     Request/response DTOs
├─ pkg/domain/                  VM, Task, Pod, Execution
└─ pkg/events/                  Event payloads
```

---

## Build Output Locations

```
After: make build

bin/
├─ worker                (services/core/cmd/worker)
├─ fc-agent             (services/core/cmd/fc-agent)
├─ aether-cli           (services/core/cmd/cli)
├─ migrate              (services/core/cmd/migrate)
└─ api-gateway          (services/gateway/cmd/api-gateway)

services/dashboard/
├─ .next/               (built Next.js)
├─ out/                 (static export if configured)
└─ node_modules/        (npm dependencies)
```

---

## Import Statement Examples

### From Gateway Service
```go
// Before (Current)
import (
    "github.com/aetherium/aetherium/pkg/vmm"
    "github.com/aetherium/aetherium/pkg/api"
)

// After (Proposed)
import (
    "github.com/aetherium/aetherium/services/gateway/pkg/api"
    "github.com/aetherium/aetherium/services/core/pkg/vmm"
    "github.com/aetherium/aetherium/libs/common/pkg/logging"
    "github.com/aetherium/aetherium/libs/types/pkg/domain"
)
```

### From Core Service
```go
// Before (Current)
import (
    "github.com/aetherium/aetherium/pkg/logging"
    "github.com/aetherium/aetherium/pkg/types"
)

// After (Proposed)
import (
    "github.com/aetherium/aetherium/services/core/pkg/vmm"
    "github.com/aetherium/aetherium/libs/common/pkg/logging"
    "github.com/aetherium/aetherium/libs/types/pkg/domain"
)
```

---

## Development Workflow Visual

```
Developer Workflow: "I want to add a new API endpoint"

1. Start here
   └─ services/gateway/README.md
      └─ "API endpoints are in pkg/api/"

2. Navigate
   └─ services/gateway/pkg/api/
      └─ Open handlers.go

3. Check dependencies
   └─ "I need a VM operation"
   └─ Import from services/core/pkg/vmm
   └─ Check services/core/go.mod for versions

4. Test locally
   └─ cd services/gateway
   └─ go test ./pkg/api/...
   └─ go run ./cmd/api-gateway

5. Verify whole chain
   └─ cd .. && make test-gateway

Clear, focused, no confusion about what belongs where.
```

---

## Deployment Architecture

### Current (Monolithic)
```
docker-compose.yml
└─ Single deployment config
   ├─ postgres
   ├─ redis
   ├─ api-gateway (mixed services)
   ├─ worker (mixed services)
   └─ dashboard
   
Issue: Must deploy all together, unclear boundaries
```

### Proposed (Service-Based)
```
infrastructure/helm/
├─ values.yaml                    Shared defaults
├─ aetherium/
│  ├─ templates/
│  │  ├─ core-worker.yaml        Core service pod
│  │  ├─ core-migrations.yaml    Database setup
│  │  ├─ gateway.yaml            Gateway service
│  │  ├─ k8s-manager.yaml        K8s manager
│  │  ├─ dashboard.yaml          Frontend service
│  │  └─ infrastructure.yaml     PostgreSQL, Redis
│  └─ Chart.yaml
│
├─ core/values-prod.yaml          Core production config
├─ gateway/values-prod.yaml       Gateway production config
└─ dashboard/values-prod.yaml     Dashboard production config

Benefits:
- helm install aetherium ... (all)
- helm install aetherium-core ... (just core)
- helm upgrade only what changed
- Scale services independently
```

---

## Quick Navigation Reference

### "I need to modify..."

```
VM provisioning logic
  → services/core/pkg/vmm/
     ├─ firecracker/
     ├─ docker/
     └─ interface.go

Network setup
  → services/core/pkg/network/

Database schema
  → services/core/migrations/

REST API
  → services/gateway/pkg/api/
     └─ handlers.go

GitHub integration
  → services/gateway/pkg/integrations/
     └─ github/

Kubernetes orchestration
  → services/k8s-manager/pkg/

Frontend UI
  → services/dashboard/src/

Logging system
  → libs/common/pkg/logging/

Domain types
  → libs/types/pkg/domain/

Helm charts
  → infrastructure/helm/
     ├─ aetherium/
     ├─ core/
     └─ gateway/

Infrastructure code
  → infrastructure/pulumi/core/
```

---

## Summary Table

| Aspect | Current | Proposed |
|--------|---------|----------|
| **Service Clarity** | ❌ Blurred | ✅ Crystal Clear |
| **Module Boundaries** | ❌ Single go.mod | ✅ go.work + 6 modules |
| **Dependency Tracking** | ❌ Implicit | ✅ Explicit in go.mod |
| **Build Time** | ⚠️ Full rebuild always | ✅ Selective builds |
| **Test Isolation** | ❌ All tests run | ✅ Per-service tests |
| **Team Ownership** | ❌ Unclear | ✅ Clear service teams |
| **Onboarding** | ❌ Complex | ✅ Service-focused |
| **New Service Addition** | ❌ Where to put it? | ✅ services/{name}/ |
| **Deployment Flexibility** | ❌ All or nothing | ✅ Selective deployment |
| **Documentation** | ❌ Fragmented | ✅ Service-specific |

---

This visual structure makes it immediately clear:
- What goes where
- How services relate
- Where to find code
- How to extend the system
- How to deploy pieces independently
