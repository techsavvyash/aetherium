# Current vs Proposed Structure Comparison

## Visual Comparison

### CURRENT STATE (Haphazard)

```
aetherium/
├── cmd/                          # Mixed services
│   ├── api-gateway/             # Gateway service
│   ├── worker/                  # Core service
│   ├── fc-agent/                # Core service
│   ├── aether-cli/              # Core CLI
│   ├── migrate/                 # Core DB migrations
│   ├── test-proxy/              # Test utilities
│   └── test-proxy-with-squid/
│
├── pkg/                          # All business logic (entangled)
│   ├── api/                     # Gateway
│   ├── integrations/            # Gateway
│   ├── vmm/                     # Core
│   ├── network/                 # Core
│   ├── storage/                 # Core
│   ├── queue/                   # Core
│   ├── tools/                   # Core
│   ├── service/                 # Core + Gateway
│   ├── worker/                  # Core
│   ├── container/               # Shared
│   ├── config/                  # Shared
│   ├── logging/                 # Shared
│   ├── events/                  # Shared
│   ├── types/                   # Shared
│   ├── mcp/                     # Unclear ownership
│   ├── discovery/               # Unclear ownership
│   ├── websocket/               # Unclear ownership
│   └── ...
│
├── dashboard/                    # Isolated frontend (good)
│   ├── src/
│   ├── package.json
│   └── ...
│
├── helm/                         # Infrastructure code
│   └── aetherium/
│
├── pulumi/                       # Infrastructure code
│   ├── aetherium.ts
│   ├── index.ts
│   └── ...
│
├── migrations/                   # Database migrations
├── internal/                     # Unused?
├── tests/                        # Minimal tests
│
├── go.mod                        # Single large go.mod
├── Makefile                      # Builds everything together
└── README.md
```

**Problems:**
- ❌ All services in single `pkg/` namespace
- ❌ Circular dependencies possible
- ❌ Can't build/test services independently
- ❌ Unclear which code belongs to which service
- ❌ Infrastructure code loosely integrated
- ❌ Hard to add new services (K8s manager)
- ❌ Large monolithic go.mod
- ❌ Difficult to scale team ownership

---

### PROPOSED STATE (Organized Monorepo)

```
aetherium/
│
├── services/                     # ✅ Service tier
│   │
│   ├── core/                     # VM provisioning & lifecycle
│   │   ├── cmd/
│   │   │   ├── worker/          # Task worker daemon
│   │   │   ├── fc-agent/        # VM agent
│   │   │   ├── migrate/         # DB migrations
│   │   │   └── cli/             # CLI tool
│   │   │
│   │   ├── pkg/
│   │   │   ├── vmm/             # VM orchestration
│   │   │   ├── network/         # TAP/bridge setup
│   │   │   ├── storage/         # Database
│   │   │   ├── queue/           # Task queue
│   │   │   ├── tools/           # Tool installation
│   │   │   ├── service/         # Business logic
│   │   │   └── types/           # Core domain types
│   │   │
│   │   ├── migrations/          # DB schema
│   │   ├── tests/               # Core tests
│   │   ├── go.mod               # ✅ Isolated module
│   │   ├── Makefile             # Service-specific build
│   │   └── README.md
│   │
│   ├── gateway/                 # REST API & integrations
│   │   ├── cmd/
│   │   │   └── api-gateway/
│   │   │
│   │   ├── pkg/
│   │   │   ├── api/             # REST handlers
│   │   │   ├── middleware/      # HTTP middleware
│   │   │   ├── auth/            # Authentication
│   │   │   ├── integrations/    # Plugins
│   │   │   └── types/           # API types
│   │   │
│   │   ├── go.mod               # ✅ Isolated module
│   │   ├── Makefile
│   │   └── README.md
│   │
│   ├── k8s-manager/             # Kubernetes orchestration
│   │   ├── cmd/
│   │   │   └── k8s-manager/
│   │   │
│   │   ├── pkg/
│   │   │   ├── orchestrator/
│   │   │   ├── podlifecycle/
│   │   │   └── scaling/
│   │   │
│   │   ├── go.mod
│   │   ├── Makefile
│   │   └── README.md
│   │
│   └── dashboard/               # Frontend (Next.js)
│       ├── app/
│       ├── components/
│       ├── package.json
│       ├── Makefile
│       └── README.md
│
├── libs/                         # ✅ Shared libraries tier
│   │
│   ├── common/                   # Non-business utilities
│   │   ├── pkg/
│   │   │   ├── logging/         # Logging abstractions
│   │   │   ├── config/          # Config management
│   │   │   ├── container/       # DI container
│   │   │   ├── events/          # Event bus abstractions
│   │   │   └── errors/
│   │   │
│   │   ├── go.mod
│   │   └── README.md
│   │
│   └── types/                    # Shared domain types
│       ├── pkg/
│       │   ├── api/             # API types
│       │   ├── domain/          # Domain models
│       │   └── events/          # Event types
│       │
│       ├── go.mod
│       └── README.md
│
├── infrastructure/               # ✅ Infrastructure tier
│   │
│   ├── helm/                     # Kubernetes Helm charts
│   │   ├── aetherium/           # Core chart
│   │   │   ├── templates/
│   │   │   ├── values.yaml
│   │   │   └── Chart.yaml
│   │   ├── gateway/
│   │   ├── k8s-manager/
│   │   ├── dashboard/
│   │   └── README.md
│   │
│   └── pulumi/                   # Pulumi IaC
│       ├── core/
│       │   ├── index.ts
│       │   ├── infrastructure.ts
│       │   ├── Pulumi.yaml
│       │   ├── package.json
│       │   └── tsconfig.json
│       ├── scripts/
│       └── README.md
│
├── tests/                        # ✅ Cross-service tests
│   ├── integration/             # Service integration tests
│   ├── e2e/                     # End-to-end scenarios
│   └── scenarios/               # Common test scenarios
│
├── docs/                         # Global documentation
│   ├── ARCHITECTURE.md
│   ├── SERVICES.md
│   ├── CONTRIBUTING.md
│   ├── DEVELOPMENT.md
│   ├── DEPLOYMENT.md
│   └── troubleshooting/
│
├── config/                       # Configuration templates
│   ├── config.yaml.example
│   ├── docker-compose.yml
│   └── .env.example
│
├── scripts/                      # Global utility scripts
│   ├── setup.sh
│   ├── local-dev.sh
│   └── clean.sh
│
├── go.work                       # ✅ Go workspace
│
├── Makefile                      # ✅ Root orchestration
├── docker-compose.yml            # Infrastructure services
│
├── README.md                     # Root documentation
│
├── MONOREPO_RESTRUCTURE.md       # This plan
├── MONOREPO_MIGRATION_STEPS.md   # Migration guide
├── CURRENT_VS_PROPOSED.md        # This document
│
└── .github/
    └── workflows/
        ├── core.yml              # Core CI/CD
        ├── gateway.yml           # Gateway CI/CD
        ├── k8s-manager.yml       # K8s manager CI/CD
        ├── dashboard.yml         # Dashboard CI/CD
        └── infrastructure.yml    # Helm/Pulumi validation
```

**Benefits:**
- ✅ Clear service boundaries
- ✅ Isolated go.mod per service
- ✅ Independent builds: `cd services/core && go build`
- ✅ Independent tests: `cd services/gateway && go test ./...`
- ✅ Clear shared libraries
- ✅ Easy to add new services
- ✅ No circular dependencies
- ✅ Team scalability
- ✅ Workspace support

---

## Dependency Graph

### CURRENT STATE (Tangled)

```
┌─────────────────────────────────────────────┐
│  Everything imports from pkg/*              │
│  (can create circular dependencies)         │
└─────────────────────────────────────────────┘
              ↑         ↑         ↑
              │         │         │
        api-gateway   worker   dashboard
              │         │
         (both at root level)
```

**Issues:**
- No clear import boundaries
- Possible circular imports
- Everything coupled to everything else
- Hard to understand dependencies

### PROPOSED STATE (Layered)

```
                    services/gateway
                          │
                          ├─→ libs/common
                          ├─→ libs/types
                          └─→ services/core

                     services/k8s-manager
                          │
                          ├─→ libs/common
                          ├─→ libs/types
                          └─→ services/core

                    services/dashboard (JS)
                          │
                          └─→ services/gateway (API)

                       libs/common
                          │ (no dependencies)

                       libs/types
                          │ (minimal dependencies)

                    services/core
                          │
                          └─→ libs/common
                              libs/types
```

**Benefits:**
- Clear unidirectional dependencies
- No circular imports possible
- Easy to trace data flow
- Self-documenting architecture

---

## Build and Deployment Impact

### CURRENT STATE

```bash
# Always builds everything
$ make build
Building all services...

# Can't easily build just gateway
# Can't easily test just core
# Must manage all infrastructure together
```

**Problems:**
- Build times increase with each new service
- Can't do selective deployment
- Hard to parallelize builds

### PROPOSED STATE

```bash
# Build specific services
$ cd services/core && make build
$ cd services/gateway && make build

# Or build all from root
$ make build              # All services
$ make build-core         # Just core
$ make build-gateway      # Just gateway

# Can deploy independently
$ make docker-build       # All images
$ make deploy-core        # Just core image
$ helm upgrade aetherium-core ...

# Parallel CI/CD pipelines
- Pipeline: core → gateway → k8s-manager → dashboard
- Each can run independently after dependencies satisfied
```

**Benefits:**
- Faster iteration (build only what changed)
- Selective deployment
- Parallel CI/CD pipelines
- Better resource utilization

---

## Development Workflow Comparison

### CURRENT WORKFLOW

```bash
# Setup (confusing: which parts are needed?)
git clone ...
npm install  # dashboard
go mod download  # but where? all services?
make build

# Development (hard to isolate)
make run-gateway  # everything else still needed?
make run-worker   # can this run without gateway?

# Testing (large blast radius)
make test  # tests everything even if you only changed dashboard
go test ./pkg/api/...  # still coupled to everything

# Deploy (all or nothing)
docker-compose up  # deploys all services
```

### PROPOSED WORKFLOW

```bash
# Setup (clear: per-service or monorepo)
git clone ...
make deps  # prepare all services, OR

cd services/gateway && go mod download  # just gateway

# Development (isolated)
cd services/core
go run ./cmd/worker          # focused terminal

# Another terminal
cd services/gateway
go run ./cmd/api-gateway     # independent service

# Testing (selective)
make test                    # all services
make test-core              # just core
cd services/gateway && go test ./pkg/api/...  # just gateway API

# Deploy (selective or full)
make docker-build           # all images
docker build services/core  # just core image
make deploy-helm ENV=prod   # deploy all
make deploy-core            # just core service
```

---

## Team Organization Impact

### CURRENT STATE

```
Team Structure: Everyone works on "Aetherium"
├─ Alice (full-stack)
├─ Bob (full-stack)
└─ Charlie (full-stack)

Problem: No clear ownership, everything's interconnected
```

### PROPOSED STATE

```
Team Structure: Service-based ownership

Core Team:
├─ Alice (core service lead)
│  └─ owns: services/core/*, migrations/
│
Gateway Team:
├─ Bob (gateway lead)
│  └─ owns: services/gateway/*, integrations/
│
K8s Team:
├─ Charlie (k8s lead)
│  └─ owns: services/k8s-manager/
│
Frontend Team:
├─ Diana (frontend lead)
│  └─ owns: services/dashboard/
│
Platform Team:
└─ Eve (platform lead)
   └─ owns: libs/*, infrastructure/*, scripts/

Benefit: Clear ownership, reduced merge conflicts, focused expertise
```

---

## Migration Effort Estimation

| Phase | Task | Effort | Notes |
|-------|------|--------|-------|
| 1 | Create directory structure | 30 min | Straightforward |
| 2 | Move core service code | 45 min | Update imports, create go.mod |
| 3 | Move gateway service code | 30 min | Smaller, fewer dependencies |
| 4 | Create K8s manager placeholder | 15 min | Template for future development |
| 5 | Move shared libraries | 30 min | Extract common code |
| 6 | Create go.work | 15 min | Workspace configuration |
| 7 | Update imports across codebase | 1-2 hr | Most time-consuming |
| 8 | Update Makefiles and build | 45 min | Root + service-level builds |
| 9 | Update documentation | 1 hr | READMEs, guides, architecture |
| 10 | Update CI/CD pipelines | 1 hr | GitHub Actions workflows |
| 11 | Verification and testing | 1-2 hr | Ensure everything builds/tests |
| **Total** | | **5-7 hours** | One person can do this |

---

## Risk Assessment

### LOW RISK
- Directory restructuring (reversible with git)
- Documentation updates (no code impact)
- go.work configuration (additive)

### MEDIUM RISK
- Import path changes (large refactor, but mechanical)
- Makefile updates (can break builds, but testable)
- go.mod creation (need to get dependencies right)

### HIGH RISK
- None identified with proper testing

### Mitigation
- Use git branches for migration
- Test each service builds/tests independently
- Keep old structure in git history
- Easy to rollback if issues arise

---

## Success Criteria

✅ All services build independently:
```bash
cd services/core && go build ./...
cd services/gateway && go build ./...
```

✅ All tests pass:
```bash
make test
```

✅ Import paths are correct:
```bash
go mod verify
go work verify
```

✅ Documentation is updated with new structure

✅ CI/CD pipelines work with selective builds

✅ Team understands new structure

---

## Next Steps

1. **Review** this document with team
2. **Approve** the proposed structure
3. **Schedule** migration (suggest: dedicated day/half-day)
4. **Backup** current state
5. **Execute** MONOREPO_MIGRATION_STEPS.md
6. **Verify** all builds and tests pass
7. **Deploy** with new structure
8. **Document** lessons learned
