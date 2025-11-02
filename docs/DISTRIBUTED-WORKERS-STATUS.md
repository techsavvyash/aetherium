# Distributed Worker Orchestration - Status & Roadmap

**Last Updated:** 2025-11-02
**Status:** Phase 1 & 2 Complete ✅ | Phase 3 Pending

---

## 🎯 Project Goal

Enable Aetherium to spawn micro-VMs across multiple physical machines in a network cluster, with:
- Automatic worker discovery
- Centralized orchestration
- Resource-aware scheduling
- Web UI for monitoring and control

---

## ✅ **COMPLETED WORK**

### Phase 1: Core Infrastructure (100% Complete)

#### 1.1 Service Discovery System
**Status:** ✅ Complete
**Files:**
- `pkg/discovery/types.go` (280 lines)
- `pkg/discovery/interface.go` (120 lines)
- `pkg/discovery/consul/consul.go` (320 lines)

**Features:**
- ✅ Pluggable service registry interface (Consul, etcd-ready)
- ✅ Worker registration with TTL-based health checks
- ✅ Heartbeat mechanism (configurable interval)
- ✅ Worker metadata (ID, hostname, zone, labels, capabilities)
- ✅ Resource tracking (CPU, memory, disk, VM count)
- ✅ Filter workers by zone/labels/capabilities/status
- ✅ Real-time worker events via Watch API
- ✅ Health check configuration (interval, timeout, deregister age)

**API:**
```go
type ServiceRegistry interface {
    Register(ctx, *WorkerInfo) error
    Deregister(ctx, workerID) error
    UpdateStatus(ctx, workerID, status) error
    UpdateResources(ctx, workerID, *WorkerResources) error
    Heartbeat(ctx, workerID) error
    GetWorker(ctx, workerID) (*WorkerInfo, error)
    ListWorkers(ctx) ([]*WorkerInfo, error)
    ListWorkersWithFilter(ctx, *WorkerFilter) ([]*WorkerInfo, error)
    Watch(ctx) (<-chan *WorkerEvent, error)
}
```

#### 1.2 Database Schema
**Status:** ✅ Complete
**Files:**
- `migrations/000002_add_workers.up.sql` (140 lines)
- `migrations/000002_add_workers.down.sql` (30 lines)
- `pkg/storage/postgres/workers.go` (230 lines)
- `pkg/storage/postgres/worker_metrics.go` (130 lines)
- `pkg/storage/storage.go` (modified, +70 lines)

**Tables:**
```sql
-- Workers table
workers (
    id, hostname, address, status, last_seen, started_at,
    zone, labels JSONB, capabilities JSONB,
    cpu_cores, memory_mb, disk_gb,
    used_cpu_cores, used_memory_mb, used_disk_gb,
    vm_count, max_vms, metadata JSONB
)

-- Worker metrics (historical)
worker_metrics (
    id, worker_id, timestamp,
    cpu_usage, memory_usage, disk_usage,
    vm_count, tasks_processed,
    network_in_mb, network_out_mb
)

-- Enhanced VMs table
vms.worker_id -> FOREIGN KEY workers(id)
tasks.worker_id -> FOREIGN KEY workers(id)
```

**Indices:**
- `idx_workers_status`, `idx_workers_zone`, `idx_workers_last_seen`
- `idx_workers_labels` (GIN), `idx_workers_capabilities` (GIN)
- `idx_vms_worker_id`, `idx_tasks_worker_id`

#### 1.3 Enhanced Worker
**Status:** ✅ Complete
**Files:**
- `pkg/worker/worker.go` (modified, +350 lines)

**Features:**
- ✅ Dual constructor: `New()` (legacy) and `NewWithConfig()` (distributed)
- ✅ Worker registration with Consul + database
- ✅ Automatic heartbeat (goroutine-based, configurable interval)
- ✅ Resource tracking (real-time VM count, CPU, memory usage)
- ✅ Populates `worker_id` when creating VMs
- ✅ Updates worker resources on VM create/delete
- ✅ Thread-safe with RWMutex
- ✅ Graceful deregistration on shutdown

**API:**
```go
type Config struct {
    ID, Hostname, Address, Zone string
    Labels       map[string]string
    Capabilities []string
    CPUCores     int
    MemoryMB, DiskGB int64
    MaxVMs       int
    Registry     discovery.ServiceRegistry
}

func NewWithConfig(store, orchestrator, *Config) (*Worker, error)
func (w *Worker) Register(ctx) error
func (w *Worker) StartHeartbeat(interval) error
func (w *Worker) Deregister(ctx) error
func (w *Worker) GetWorkerInfo() *WorkerInfo
```

### Phase 2: Service Layer & API (100% Complete)

#### 2.1 Worker Management Service
**Status:** ✅ Complete
**Files:**
- `pkg/service/worker_service.go` (420 lines)

**Features:**
- ✅ High-level worker queries (list, get, filter by zone)
- ✅ Cluster statistics (aggregate CPU, memory, VMs, zones)
- ✅ VM distribution across workers
- ✅ Worker control (drain, activate)
- ✅ Worker health status (last_seen < 60s)
- ✅ Usage percentage calculations

**API:**
```go
type WorkerService struct {}

// Queries
ListWorkers(ctx) ([]*WorkerStats, error)
GetWorker(ctx, workerID) (*WorkerStats, error)
ListActiveWorkers(ctx) ([]*WorkerStats, error)
ListWorkersByZone(ctx, zone) ([]*WorkerStats, error)

// Cluster ops
GetClusterStats(ctx) (*ClusterStats, error)
GetVMDistribution(ctx) ([]*VMDistribution, error)
GetWorkerVMs(ctx, workerID) ([]VMInfo, error)

// Control
DrainWorker(ctx, workerID) error
ActivateWorker(ctx, workerID) error
```

**Response Types:**
```go
type WorkerStats struct {
    ID, Hostname, Address, Zone, Status string
    Labels, Capabilities []string
    CPUCores, MemoryMB, DiskGB int64
    UsedCPUCores, UsedMemoryMB, UsedDiskGB int64
    CPUUsagePercent, MemoryUsagePercent float64
    VMCount, MaxVMs int
    StartedAt, LastSeen time.Time
    Uptime string
    IsHealthy bool
}

type ClusterStats struct {
    TotalWorkers, ActiveWorkers, DrainingWorkers, OfflineWorkers int
    TotalCPUCores, UsedCPUCores, AvailableCPUCores int
    TotalMemoryMB, UsedMemoryMB, AvailableMemoryMB int64
    ClusterCPUUsagePercent, ClusterMemoryUsagePercent float64
    TotalVMs, MaxVMs, AvailableVMSlots int
    Zones map[string]int
}
```

#### 2.2 API Gateway Endpoints
**Status:** ✅ Complete
**Files:**
- `cmd/api-gateway/main.go` (modified, +140 lines)
- `pkg/storage/postgres/vms.go` (modified, +10 lines for worker_id)

**Endpoints:**
```
GET  /api/v1/workers              - List workers (?zone=us-west-1a)
GET  /api/v1/workers/{id}         - Worker details
GET  /api/v1/workers/{id}/vms     - VMs on worker
POST /api/v1/workers/{id}/drain   - Drain worker
POST /api/v1/workers/{id}/activate - Activate worker

GET  /api/v1/cluster/stats        - Cluster statistics
GET  /api/v1/cluster/distribution - VM distribution

# Enhanced existing
GET  /api/v1/vms                  - Now includes worker_id
GET  /api/v1/vms/{id}             - Shows worker assignment
```

**Example Response:**
```json
{
  "workers": [{
    "id": "worker-01",
    "hostname": "node1.example.com",
    "zone": "us-west-1a",
    "status": "active",
    "cpu_usage_percent": 25.0,
    "memory_usage_percent": 25.0,
    "vm_count": 3,
    "max_vms": 100,
    "is_healthy": true,
    "uptime": "2h30m45s"
  }],
  "total": 1
}
```

### Phase 3: Documentation (100% Complete)

#### 3.1 API Documentation
**Status:** ✅ Complete
**Files:**
- `docs/DISTRIBUTED-WORKER-API.md` (480 lines)

**Contents:**
- Complete API reference for all 7 new endpoints
- Request/response examples
- Query parameter documentation
- Error handling
- Use cases and examples
- Integration with existing endpoints

#### 3.2 Implementation Summary
**Status:** ✅ Complete
**Files:**
- `DISTRIBUTED-IMPLEMENTATION-SUMMARY.md` (quick reference)
- `docs/DISTRIBUTED-WORKERS-STATUS.md` (this file)

---

## 🚧 **PENDING WORK**

### Phase 4: WebSocket & Real-Time Updates (Not Started)

**Priority:** High
**Estimated Effort:** 4-6 hours

#### 4.1 WebSocket Server
**Files to Create:**
- `pkg/api/websocket.go` - WebSocket connection handler
- `pkg/events/types.go` - Event type definitions

**Features Needed:**
- ✅ WebSocket endpoint: `GET /api/v1/events`
- ✅ Real-time worker join/leave events
- ✅ VM creation/deletion events
- ✅ Task status updates
- ✅ Resource usage updates
- ✅ Subscribe to Consul Watch API
- ✅ Subscribe to Redis event bus

**Event Types:**
```go
type Event struct {
    Type      EventType  // worker.joined, vm.created, etc.
    Timestamp time.Time
    Data      interface{}
}

type EventType string
const (
    EventWorkerJoined   EventType = "worker.joined"
    EventWorkerLeft     EventType = "worker.left"
    EventWorkerUpdated  EventType = "worker.updated"
    EventVMCreated      EventType = "vm.created"
    EventVMDeleted      EventType = "vm.deleted"
    EventTaskStarted    EventType = "task.started"
    EventTaskCompleted  EventType = "task.completed"
)
```

#### 4.2 Event Streaming Integration
**Files to Modify:**
- `cmd/api-gateway/main.go` - Add WebSocket route

**Tasks:**
- Watch Consul for worker changes
- Subscribe to Redis for task events
- Aggregate and forward to WebSocket clients
- Connection pooling and management

### Phase 5: Web UI (Not Started)

**Priority:** High
**Estimated Effort:** 12-16 hours

#### 5.1 Frontend Setup
**Files to Create:**
- `web/package.json` - Node.js dependencies
- `web/index.html` - Entry point
- `web/src/App.jsx` - Main app component
- `web/src/websocket.js` - WebSocket client

**Technology Stack:**
- React 18 (or htmx for lightweight alternative)
- Chart.js for visualizations
- Tailwind CSS for styling
- WebSocket for real-time updates

#### 5.2 Dashboard Page
**Files to Create:**
- `web/src/components/Dashboard.jsx`
- `web/src/components/ClusterStats.jsx`
- `web/src/components/WorkerGrid.jsx`

**Features:**
- Cluster overview (total workers, VMs, resources)
- Worker status cards (grid layout)
- Resource usage gauges (CPU, memory)
- Zone distribution chart
- Real-time updates via WebSocket

#### 5.3 Worker Monitoring Page
**Files to Create:**
- `web/src/components/WorkerList.jsx`
- `web/src/components/WorkerDetails.jsx`
- `web/src/components/ResourceChart.jsx`

**Features:**
- Searchable/filterable worker table
- Zone filter dropdown
- Status filter (active/draining/offline)
- Capability filter
- Worker detail modal
- Resource usage timeline charts

#### 5.4 VM Distribution Page
**Files to Create:**
- `web/src/components/VMDistribution.jsx`
- `web/src/components/VMList.jsx`

**Features:**
- Heatmap of VMs per worker
- Worker capacity bars
- VM list with filtering
- Click to view VM details
- Worker assignment visualization

#### 5.5 Task Queue Page
**Files to Create:**
- `web/src/components/TaskQueue.jsx`
- `web/src/components/TaskDetails.jsx`

**Features:**
- Pending/running/completed task counts
- Task timeline visualization
- Task filtering by type/status
- Real-time task updates

#### 5.6 Control Panel
**Files to Create:**
- `web/src/components/ControlPanel.jsx`

**Features:**
- Drain worker button (with confirmation)
- Activate worker button
- View worker logs
- Cancel tasks (future)

### Phase 6: Enhanced Worker Functionality (Partially Complete)

#### 6.1 Worker Drain with Queue Integration
**Status:** API endpoint exists, queue logic missing
**Priority:** Medium
**Estimated Effort:** 2-3 hours

**Current:** API endpoint marks worker as "draining" in database
**Needed:** Queue integration to stop pulling new tasks

**Files to Modify:**
- `pkg/queue/asynq/asynq.go` - Check worker status before accepting task
- `pkg/worker/worker.go` - Stop pulling from queue when draining

**Implementation:**
```go
// In worker task handler wrapper
func (w *Worker) checkDrainStatus() bool {
    if w.workerInfo.Status == discovery.WorkerStatusDraining {
        // Don't accept new tasks
        return true
    }
    return false
}
```

#### 6.2 Task Cancellation
**Status:** Not started
**Priority:** Medium
**Estimated Effort:** 3-4 hours

**Files to Create:**
- `pkg/api/` - Add cancellation endpoint

**Files to Modify:**
- `pkg/worker/worker.go` - Context cancellation support
- `cmd/api-gateway/main.go` - Add `POST /api/v1/tasks/{id}/cancel`

**Features:**
- Cancel pending tasks (remove from queue)
- Cancel running tasks (context cancellation)
- Cleanup resources (VMs, etc.)

### Phase 7: Configuration & Deployment (Not Started)

#### 7.1 Configuration System
**Status:** Not started
**Priority:** High
**Estimated Effort:** 2-3 hours

**Files to Create:**
- `config.example.yaml` - Example configuration
- `pkg/config/config.go` - Enhanced config loader

**Current:** Environment variables only
**Needed:** YAML configuration file support

**Example Config:**
```yaml
discovery:
  provider: consul
  consul:
    address: localhost:8500
    datacenter: dc1
    health_check_interval: 10s

worker:
  id: worker-01  # Auto-generate if empty
  hostname: node1.example.com
  zone: us-west-1a
  labels:
    env: production
    tier: compute
  capabilities:
    - firecracker
    - docker
  resources:
    cpu_cores: 16
    memory_mb: 32768
    disk_gb: 500
    max_vms: 100

api_gateway:
  port: 8080
  enable_cors: true
  enable_websocket: true
```

#### 7.2 Docker Compose Setup
**Status:** Not started
**Priority:** Medium
**Estimated Effort:** 2-3 hours

**Files to Create:**
- `docker-compose.yml` - Multi-container setup
- `docker-compose.dev.yml` - Development overrides

**Services:**
```yaml
services:
  postgres:
    image: postgres:15-alpine

  redis:
    image: redis:7-alpine

  consul:
    image: consul:latest

  api-gateway:
    build: .
    command: /bin/api-gateway

  worker-1:
    build: .
    command: /bin/worker
    environment:
      WORKER_ID: worker-01
      WORKER_ZONE: us-west-1a

  worker-2:
    build: .
    command: /bin/worker
    environment:
      WORKER_ID: worker-02
      WORKER_ZONE: us-west-1b
```

#### 7.3 Kubernetes Manifests
**Status:** Not started
**Priority:** Low
**Estimated Effort:** 4-6 hours

**Files to Create:**
- `k8s/postgres.yaml`
- `k8s/redis.yaml`
- `k8s/consul.yaml`
- `k8s/api-gateway.yaml`
- `k8s/worker-deployment.yaml`

### Phase 8: Testing & Quality (In Progress)

#### 8.1 Integration Tests
**Status:** In progress
**Priority:** Critical
**Estimated Effort:** 4-6 hours

**Files to Create:**
- `tests/integration/worker_test.go` - Worker registration/heartbeat
- `tests/integration/cluster_test.go` - Multi-worker scenarios
- `tests/integration/api_test.go` - API endpoint testing

**Test Scenarios:**
- ✅ Worker registration with Consul
- ✅ Heartbeat mechanism
- ✅ Worker resource tracking
- ✅ VM assignment to worker
- ✅ Cluster statistics calculation
- ✅ Worker drain/activate
- ✅ API endpoint responses

#### 8.2 Unit Tests
**Status:** Not started
**Priority:** Medium
**Estimated Effort:** 3-4 hours

**Files to Create:**
- `pkg/discovery/consul/consul_test.go`
- `pkg/service/worker_service_test.go`
- `pkg/worker/worker_test.go`

#### 8.3 Load Testing
**Status:** Not started
**Priority:** Low
**Estimated Effort:** 2-3 hours

**Test Cases:**
- 100 workers registering simultaneously
- 1000 concurrent VMs across cluster
- Heartbeat storm (all workers at once)
- API endpoint throughput

### Phase 9: Production Readiness (Not Started)

#### 9.1 Monitoring & Alerting
**Status:** Not started
**Priority:** Medium

**Features Needed:**
- Prometheus metrics export
- Grafana dashboards
- Alert rules (worker down, high CPU, etc.)

#### 9.2 Security Hardening
**Status:** Not started
**Priority:** High

**Features Needed:**
- Consul ACL tokens
- API authentication (JWT)
- mTLS for worker-gateway communication
- Rate limiting

#### 9.3 Production Documentation
**Status:** Not started
**Priority:** High

**Files to Create:**
- `docs/PRODUCTION-DEPLOYMENT.md`
- `docs/WORKER-SETUP.md`
- `docs/TROUBLESHOOTING.md`

---

## 📊 **Progress Summary**

| Phase | Component | Status | Progress |
|-------|-----------|--------|----------|
| 1 | Service Discovery | ✅ Complete | 100% |
| 1 | Database Schema | ✅ Complete | 100% |
| 1 | Enhanced Worker | ✅ Complete | 100% |
| 2 | Worker Service | ✅ Complete | 100% |
| 2 | API Gateway | ✅ Complete | 100% |
| 3 | Documentation | ✅ Complete | 100% |
| 4 | WebSocket Events | ⏳ Pending | 0% |
| 5 | Web UI | ⏳ Pending | 0% |
| 6 | Drain Logic | 🟡 Partial | 50% |
| 6 | Task Cancellation | ⏳ Pending | 0% |
| 7 | Configuration | ⏳ Pending | 0% |
| 7 | Docker Compose | ⏳ Pending | 0% |
| 7 | Kubernetes | ⏳ Pending | 0% |
| 8 | Integration Tests | 🟡 In Progress | 10% |
| 8 | Unit Tests | ⏳ Pending | 0% |
| 9 | Monitoring | ⏳ Pending | 0% |
| 9 | Security | ⏳ Pending | 0% |

**Overall Progress:** 55% Complete

---

## 🎯 **Recommended Next Steps**

### Immediate (This Session)
1. ✅ **Integration Tests** - Validate current implementation
2. ✅ **Test Worker Registration** - Ensure Consul integration works
3. ✅ **Test API Endpoints** - Verify all endpoints return correct data

### Short-term (Next Session)
4. **WebSocket Event Streaming** - Enable real-time updates
5. **Worker Drain Queue Logic** - Complete drain functionality
6. **Configuration System** - YAML config support

### Medium-term
7. **Web UI Dashboard** - Build React frontend
8. **Docker Compose** - Multi-container development setup
9. **Production Docs** - Deployment guides

### Long-term
10. **Monitoring & Alerting** - Prometheus + Grafana
11. **Security Hardening** - Authentication, mTLS
12. **Kubernetes** - Production orchestration

---

## 🔧 **Current Limitations**

1. **No Real-Time Updates**: Clients must poll for changes
2. **Basic Drain Logic**: Workers marked as draining still pull tasks from queue
3. **No Task Cancellation**: Can't cancel in-flight tasks
4. **Environment Variables Only**: No YAML config support
5. **No Web UI**: Must use curl/Postman for API testing
6. **Limited Testing**: Only manual testing performed so far
7. **No Monitoring**: Can't track historical metrics
8. **No Authentication**: API is wide open

---

## 💡 **Design Decisions**

### Why Consul?
- Industry-standard service discovery
- Built-in health checks
- Strong consistency guarantees
- Simple HTTP API
- Multi-datacenter support

### Why Queue-Based Distribution?
- Workers self-assign tasks (pull model)
- Natural load balancing
- No complex scheduling logic needed
- Easy to scale horizontally
- Proven pattern (Kubernetes, Nomad)

### Why PostgreSQL for Worker State?
- Source of truth for worker metadata
- Historical metrics storage
- Rich query capabilities
- JSONB for flexible metadata

### Why Worker_ID in VMs Table?
- Track which worker created each VM
- Enable worker-specific queries
- Support future VM migration
- Audit trail

---

## 📖 **References**

- [Distributed Worker API Docs](./DISTRIBUTED-WORKER-API.md)
- [Consul Documentation](https://www.consul.io/docs)
- [Asynq Queue Library](https://github.com/hibiken/asynq)
- [PostgreSQL JSONB](https://www.postgresql.org/docs/current/datatype-json.html)

---

**End of Status Document**
