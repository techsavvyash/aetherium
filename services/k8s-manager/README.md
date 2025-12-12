# Aetherium K8s Manager Service

**Kubernetes pod lifecycle management and orchestration.**

K8s Manager handles the integration between Aetherium and Kubernetes, managing pod creation, deployment strategies, auto-scaling, and health monitoring.

## Status

🚧 **In Development** - This service is a placeholder for future development.

## Vision

The K8s Manager will:
- Create and manage Kubernetes pods from Aetherium tasks
- Define deployment strategies (rolling, blue-green, etc.)
- Manage resource scaling based on demand
- Monitor pod health and lifecycle
- Bridge between VM-level and Kubernetes orchestration

## Architecture (Planned)

### Packages

- **pkg/orchestrator/** - Kubernetes client wrappers
- **pkg/podlifecycle/** - Pod CRUD operations
- **pkg/deployment/** - Deployment strategies
- **pkg/scaling/** - Auto-scaling logic
- **pkg/monitoring/** - Health monitoring

### Integration Points

```
Gateway API
    ↓
K8s Manager
    ├─ Core Service (for VM fallback)
    ├─ Kubernetes Cluster (for pod management)
    └─ Prometheus (for metrics)
```

## Planned API

```go
type K8sOrchestrator interface {
    // Pod lifecycle
    CreatePod(ctx, spec) (*Pod, error)
    GetPod(ctx, namespace, name) (*Pod, error)
    ListPods(ctx, namespace) ([]*Pod, error)
    DeletePod(ctx, namespace, name) error
    
    // Deployments
    CreateDeployment(ctx, spec) (*Deployment, error)
    UpdateDeployment(ctx, spec) error
    DeleteDeployment(ctx, namespace, name) error
    
    // Scaling
    ScalePod(ctx, podID, replicas) error
    
    // Health
    GetPodLogs(ctx, namespace, pod) (string, error)
    GetPodEvents(ctx, namespace, pod) ([]*Event, error)
}
```

## Building

```bash
# Currently just a placeholder
cd services/k8s-manager
make build

# Will eventually produce k8s-manager binary
```

## Testing

```bash
# Currently no tests
make test
```

## Configuration (Planned)

```yaml
k8s_manager:
  enabled: false  # Enable when implemented
  
kubernetes:
  config_path: ~/.kube/config
  namespace: aetherium
  
deployment:
  strategy: rolling  # or: blue-green, canary
  max_surge: 1
  max_unavailable: 0

scaling:
  enabled: false
  metrics: prometheus
  min_replicas: 1
  max_replicas: 10
  target_cpu: 70
```

## Imports from Core

Will import from core service:

```go
import "github.com/aetherium/aetherium/services/core/pkg/service"
```

To provide fallback to VM-based execution if Kubernetes is unavailable.

## Next Steps for Implementation

1. **Phase 1**: Basic pod creation/deletion
   - Kubernetes client setup
   - Pod CRUD operations
   - Basic monitoring

2. **Phase 2**: Deployment strategies
   - Rolling deployments
   - Blue-green deployments
   - Canary deployments

3. **Phase 3**: Auto-scaling
   - Metrics collection
   - Scaling policies
   - Load testing

4. **Phase 4**: Advanced features
   - Service mesh integration
   - Custom resource definitions
   - Multi-cluster support

## Related Documentation

- See [../../docs/KUBERNETES.md](../../docs/KUBERNETES.md) (planned)
- See [../../MONOREPO_QUICK_REFERENCE.md](../../MONOREPO_QUICK_REFERENCE.md) for quick lookup
- See services/core/README.md for Core service details

## Development

To start implementing:

```bash
# Add Kubernetes dependencies
cd services/k8s-manager
go get k8s.io/client-go k8s.io/api

# Create first implementation
mkdir -p pkg/{orchestrator,podlifecycle,deployment,scaling,monitoring}
touch pkg/orchestrator/interface.go
```

## Contributing

When implementing features:
1. Follow existing code style in services/core/
2. Create interfaces in `pkg/*/interface.go`
3. Implement in separate files
4. Add comprehensive tests
5. Update this README

