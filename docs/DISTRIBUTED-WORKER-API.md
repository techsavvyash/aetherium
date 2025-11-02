# Distributed Worker Orchestration API

This document describes the REST API endpoints for managing distributed workers and cluster operations in Aetherium.

## Base URL

```
http://localhost:8080/api/v1
```

## Worker Management Endpoints

### List All Workers

Returns all registered workers in the cluster.

**Endpoint:** `GET /workers`

**Query Parameters:**
- `zone` (optional): Filter workers by geographic zone

**Example Request:**
```bash
# List all workers
curl http://localhost:8080/api/v1/workers

# List workers in specific zone
curl http://localhost:8080/api/v1/workers?zone=us-west-1a
```

**Example Response:**
```json
{
  "workers": [
    {
      "id": "worker-01",
      "hostname": "node1.example.com",
      "address": "192.168.1.10:8081",
      "zone": "us-west-1a",
      "status": "active",
      "labels": {
        "env": "production",
        "tier": "compute"
      },
      "capabilities": ["firecracker", "docker"],
      "cpu_cores": 16,
      "memory_mb": 32768,
      "disk_gb": 500,
      "used_cpu_cores": 4,
      "used_memory_mb": 8192,
      "used_disk_gb": 50,
      "cpu_usage_percent": 25.0,
      "memory_usage_percent": 25.0,
      "vm_count": 3,
      "max_vms": 100,
      "started_at": "2025-11-02T10:00:00Z",
      "last_seen": "2025-11-02T12:30:45Z",
      "uptime": "2h30m45s",
      "is_healthy": true
    }
  ],
  "total": 1
}
```

### Get Worker Details

Get detailed information about a specific worker.

**Endpoint:** `GET /workers/{id}`

**Example Request:**
```bash
curl http://localhost:8080/api/v1/workers/worker-01
```

**Example Response:**
```json
{
  "id": "worker-01",
  "hostname": "node1.example.com",
  "address": "192.168.1.10:8081",
  "zone": "us-west-1a",
  "status": "active",
  "labels": {
    "env": "production",
    "tier": "compute"
  },
  "capabilities": ["firecracker", "docker"],
  "cpu_cores": 16,
  "memory_mb": 32768,
  "disk_gb": 500,
  "used_cpu_cores": 4,
  "used_memory_mb": 8192,
  "used_disk_gb": 50,
  "cpu_usage_percent": 25.0,
  "memory_usage_percent": 25.0,
  "vm_count": 3,
  "max_vms": 100,
  "started_at": "2025-11-02T10:00:00Z",
  "last_seen": "2025-11-02T12:30:45Z",
  "uptime": "2h30m45s",
  "is_healthy": true
}
```

### Get Worker VMs

Get all VMs running on a specific worker.

**Endpoint:** `GET /workers/{id}/vms`

**Example Request:**
```bash
curl http://localhost:8080/api/v1/workers/worker-01/vms
```

**Example Response:**
```json
{
  "worker_id": "worker-01",
  "vms": [
    {
      "id": "vm-123",
      "name": "dev-vm-1",
      "status": "running",
      "vcpus": 2,
      "memory_mb": 2048
    },
    {
      "id": "vm-456",
      "name": "dev-vm-2",
      "status": "running",
      "vcpus": 1,
      "memory_mb": 512
    }
  ],
  "total": 2
}
```

### Drain Worker

Mark a worker as draining. It will stop accepting new tasks but continue processing existing ones.

**Endpoint:** `POST /workers/{id}/drain`

**Example Request:**
```bash
curl -X POST http://localhost:8080/api/v1/workers/worker-01/drain
```

**Example Response:**
```json
{
  "worker_id": "worker-01",
  "status": "draining",
  "message": "Worker marked as draining. It will stop accepting new tasks."
}
```

### Activate Worker

Reactivate a draining worker to resume accepting tasks.

**Endpoint:** `POST /workers/{id}/activate`

**Example Request:**
```bash
curl -X POST http://localhost:8080/api/v1/workers/worker-01/activate
```

**Example Response:**
```json
{
  "worker_id": "worker-01",
  "status": "active",
  "message": "Worker activated. It will resume accepting tasks."
}
```

## Cluster Management Endpoints

### Get Cluster Statistics

Get aggregate statistics for the entire cluster.

**Endpoint:** `GET /cluster/stats`

**Example Request:**
```bash
curl http://localhost:8080/api/v1/cluster/stats
```

**Example Response:**
```json
{
  "total_workers": 5,
  "active_workers": 4,
  "draining_workers": 1,
  "offline_workers": 0,
  "total_cpu_cores": 80,
  "total_memory_mb": 163840,
  "used_cpu_cores": 20,
  "used_memory_mb": 40960,
  "available_cpu_cores": 60,
  "available_memory_mb": 122880,
  "cluster_cpu_usage_percent": 25.0,
  "cluster_memory_usage_percent": 25.0,
  "total_vms": 15,
  "max_vms": 500,
  "available_vm_slots": 485,
  "zones": {
    "us-west-1a": 2,
    "us-west-1b": 2,
    "us-east-1a": 1
  }
}
```

### Get VM Distribution

Get VM distribution across all workers.

**Endpoint:** `GET /cluster/distribution`

**Example Request:**
```bash
curl http://localhost:8080/api/v1/cluster/distribution
```

**Example Response:**
```json
{
  "distribution": [
    {
      "worker_id": "worker-01",
      "hostname": "node1.example.com",
      "zone": "us-west-1a",
      "vm_count": 5,
      "vms": [
        {
          "id": "vm-001",
          "name": "dev-vm-1",
          "status": "running",
          "vcpus": 2,
          "memory_mb": 2048
        },
        {
          "id": "vm-002",
          "name": "dev-vm-2",
          "status": "running",
          "vcpus": 1,
          "memory_mb": 1024
        }
      ]
    },
    {
      "worker_id": "worker-02",
      "hostname": "node2.example.com",
      "zone": "us-west-1a",
      "vm_count": 3,
      "vms": [...]
    }
  ],
  "total_workers": 2
}
```

## Worker Status Values

Workers can be in one of the following states:

- **active**: Worker is accepting and processing tasks
- **draining**: Worker is processing existing tasks but not accepting new ones
- **offline**: Worker is not responding to heartbeats

## Health Indicators

Workers are considered healthy if:
- Last heartbeat was received within the last 60 seconds
- `is_healthy: true` in the response

Workers are considered unhealthy if:
- Last heartbeat was > 60 seconds ago
- `is_healthy: false` in the response

## Resource Metrics

### Per-Worker Metrics

- **cpu_usage_percent**: Percentage of CPU cores in use
- **memory_usage_percent**: Percentage of memory in use
- **vm_count**: Number of VMs currently running
- **used_cpu_cores**: Number of CPU cores allocated to VMs
- **used_memory_mb**: Amount of memory allocated to VMs

### Cluster-Wide Metrics

- **cluster_cpu_usage_percent**: Average CPU usage across all workers
- **cluster_memory_usage_percent**: Average memory usage across all workers
- **available_cpu_cores**: Total unallocated CPU cores
- **available_memory_mb**: Total unallocated memory
- **available_vm_slots**: Remaining VM capacity

## Integration with Existing VM Endpoints

The existing VM endpoints have been enhanced to include worker information:

### List VMs

**Endpoint:** `GET /vms`

Now includes `worker_id` in the response:

```json
{
  "vms": [
    {
      "id": "vm-123",
      "name": "dev-vm",
      "worker_id": "worker-01",  // NEW FIELD
      "status": "running",
      ...
    }
  ]
}
```

### Get VM

**Endpoint:** `GET /vms/{id}`

Response includes worker assignment:

```json
{
  "id": "vm-123",
  "name": "dev-vm",
  "worker_id": "worker-01",  // NEW FIELD
  "status": "running",
  ...
}
```

## Use Cases

### 1. Monitoring Dashboard

```bash
# Get cluster overview
curl http://localhost:8080/api/v1/cluster/stats

# Get worker list with health status
curl http://localhost:8080/api/v1/workers

# Get VM distribution for visualization
curl http://localhost:8080/api/v1/cluster/distribution
```

### 2. Worker Maintenance

```bash
# Drain worker before maintenance
curl -X POST http://localhost:8080/api/v1/workers/worker-01/drain

# Wait for VMs to complete or migrate them
# ... perform maintenance ...

# Reactivate worker
curl -X POST http://localhost:8080/api/v1/workers/worker-01/activate
```

### 3. Capacity Planning

```bash
# Check available resources
curl http://localhost:8080/api/v1/cluster/stats | jq '{
  available_cpu: .available_cpu_cores,
  available_memory_gb: (.available_memory_mb / 1024),
  available_vm_slots: .available_vm_slots
}'
```

### 4. Zone-Based Filtering

```bash
# List workers in specific zone
curl http://localhost:8080/api/v1/workers?zone=us-west-1a

# Get VMs in a zone (via worker filtering)
for worker in $(curl -s http://localhost:8080/api/v1/workers?zone=us-west-1a | jq -r '.workers[].id'); do
  curl -s http://localhost:8080/api/v1/workers/$worker/vms
done
```

## Error Responses

All endpoints follow the standard error format:

```json
{
  "error": "Not Found",
  "message": "Worker not found: worker-99",
  "code": 404
}
```

Common error codes:
- `400` - Bad Request (invalid parameters)
- `404` - Not Found (worker/VM doesn't exist)
- `500` - Internal Server Error

## Next Steps

To enable full distributed worker functionality:

1. **Run Database Migration**:
   ```bash
   cd migrations
   migrate -database "postgres://..." -path . up
   ```

2. **Start Consul** (optional, for service discovery):
   ```bash
   consul agent -dev
   ```

3. **Start API Gateway**:
   ```bash
   ./bin/api-gateway
   ```

4. **Start Workers** (on multiple machines):
   ```bash
   # See worker configuration guide
   ./bin/worker
   ```

See [WORKER-SETUP.md](./WORKER-SETUP.md) for complete worker configuration instructions.
