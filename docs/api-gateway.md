## API Gateway Documentation

The Aetherium API Gateway provides a RESTful HTTP interface for all platform operations.

## Base URL

```
http://localhost:8080/api/v1
```

---

## Authentication

All endpoints require authentication via:
- **JWT Token**: `Authorization: Bearer <token>`
- **API Key**: `X-API-Key: <key>`

---

## Endpoints

### Virtual Machines

#### Create VM

```http
POST /vms
```

**Request:**
```json
{
  "name": "my-vm",
  "vcpus": 2,
  "memory_mb": 1024,
  "additional_tools": ["go", "python"],
  "tool_versions": {
    "go": "1.23.0",
    "python": "3.11"
  }
}
```

**Response:** `202 Accepted`
```json
{
  "task_id": "uuid",
  "status": "pending"
}
```

**Default Tools (installed automatically):**
- git
- nodejs@20
- bun@latest
- claude-code@latest

#### List VMs

```http
GET /vms
```

**Response:** `200 OK`
```json
{
  "vms": [
    {
      "id": "uuid",
      "name": "my-vm",
      "status": "running",
      "vcpu_count": 2,
      "memory_mb": 1024,
      "created_at": "2025-10-05T10:00:00Z",
      "metadata": {}
    }
  ],
  "total": 1
}
```

#### Get VM

```http
GET /vms/{id}
```

**Response:** `200 OK`
```json
{
  "id": "uuid",
  "name": "my-vm",
  "status": "running",
  "vcpu_count": 2,
  "memory_mb": 1024,
  "kernel_path": "/var/firecracker/vmlinux",
  "rootfs_path": "/var/firecracker/rootfs.ext4",
  "socket_path": "/tmp/aetherium-vm-uuid.sock",
  "created_at": "2025-10-05T10:00:00Z",
  "started_at": "2025-10-05T10:01:00Z",
  "metadata": {}
}
```

#### Delete VM

```http
DELETE /vms/{id}
```

**Response:** `202 Accepted`
```json
{
  "id": "task-uuid",
  "type": "vm:delete",
  "status": "pending"
}
```

### Command Execution

#### Execute Command

```http
POST /vms/{id}/execute
```

**Request:**
```json
{
  "command": "git",
  "args": ["clone", "https://github.com/user/repo"]
}
```

**Response:** `202 Accepted`
```json
{
  "task_id": "uuid",
  "vm_id": "vm-uuid",
  "status": "pending"
}
```

#### List Executions

```http
GET /vms/{id}/executions
```

**Response:** `200 OK`
```json
{
  "executions": [
    {
      "id": "uuid",
      "vm_id": "vm-uuid",
      "command": "git",
      "args": ["clone", "https://github.com/user/repo"],
      "exit_code": 0,
      "stdout": "Cloning into 'repo'...",
      "stderr": "",
      "started_at": "2025-10-05T10:05:00Z",
      "completed_at": "2025-10-05T10:05:30Z",
      "duration_ms": 30000
    }
  ],
  "total": 1
}
```

### Tasks

#### Get Task Status

```http
GET /tasks/{id}
```

**Response:** `200 OK`
```json
{
  "id": "uuid",
  "type": "vm:create",
  "status": "completed",
  "result": {
    "vm_id": "uuid",
    "name": "my-vm",
    "status": "running"
  },
  "created_at": "2025-10-05T10:00:00Z"
}
```

### Logs

#### Query Logs

```http
POST /logs/query
```

**Request:**
```json
{
  "vm_id": "uuid",
  "level": "error",
  "search_text": "failed",
  "start_time": 1696512000000,
  "end_time": 1696598400000,
  "limit": 100
}
```

**Response:** `200 OK`
```json
{
  "logs": [
    {
      "timestamp": "2025-10-05T10:00:00Z",
      "level": "error",
      "message": "Task failed",
      "task_id": "uuid",
      "vm_id": "uuid",
      "fields": {
        "error": "connection timeout"
      }
    }
  ],
  "total": 1
}
```

#### Stream Logs (WebSocket)

```http
GET /logs/stream?vm_id={id}&level={level}
```

Upgrade to WebSocket connection for real-time log streaming.

### Integrations

#### Webhook Handler

```http
POST /webhooks/{integration}
```

Supported integrations: `github`, `slack`

**GitHub Webhook:**
```json
{
  "action": "opened",
  "pull_request": {
    "number": 123,
    "title": "Fix bug"
  }
}
```

**Slack Webhook:**
```json
{
  "type": "slash_command",
  "command": "/aetherium-status",
  "user_id": "U123456",
  "channel_id": "C123456"
}
```

### Health

#### Health Check

```http
GET /health
```

**Response:** `200 OK`
```json
{
  "status": "ok",
  "components": {
    "github": "healthy",
    "slack": "healthy",
    "loki": "healthy",
    "event_bus": "healthy"
  },
  "timestamp": "2025-10-05T10:00:00Z"
}
```

---

## Error Responses

All errors follow this format:

```json
{
  "error": "Bad Request",
  "message": "Invalid VM ID: not a valid UUID",
  "code": 400
}
```

**Common Status Codes:**
- `200 OK` - Success
- `202 Accepted` - Async operation started
- `400 Bad Request` - Invalid input
- `401 Unauthorized` - Missing/invalid auth
- `404 Not Found` - Resource not found
- `500 Internal Server Error` - Server error

---

## Examples

### Create VM with Claude Code Support

```bash
curl -X POST http://localhost:8080/api/v1/vms \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d '{
    "name": "claude-code-vm",
    "vcpus": 2,
    "memory_mb": 2048,
    "additional_tools": ["go"],
    "tool_versions": {
      "nodejs": "20",
      "go": "1.23.0"
    }
  }'
```

**Default tools installed:**
- nodejs@20
- bun@latest
- claude-code@latest (automatically)
- git

**Additional tools installed:**
- go@1.23.0

### Execute Command in VM

```bash
# Get VM ID from create response
VM_ID="uuid-here"

# Clone a repository
curl -X POST http://localhost:8080/api/v1/vms/$VM_ID/execute \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d '{
    "command": "git",
    "args": ["clone", "https://github.com/user/repo"]
  }'
```

### Query Logs

```bash
curl -X POST http://localhost:8080/api/v1/logs/query \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d '{
    "vm_id": "uuid",
    "level": "error",
    "limit": 50
  }'
```

---

## Rate Limiting

- Default: 100 requests/minute per API key
- Burst: 20 requests
- Headers:
  - `X-RateLimit-Limit`: Total limit
  - `X-RateLimit-Remaining`: Requests remaining
  - `X-RateLimit-Reset`: Reset timestamp

---

## WebSocket Log Streaming

Connect to real-time log stream:

```javascript
const ws = new WebSocket('ws://localhost:8080/api/v1/logs/stream?vm_id=uuid');

ws.onmessage = (event) => {
  const log = JSON.parse(event.data);
  console.log(log.timestamp, log.level, log.message);
};
```

---

## Environment Variables

```bash
# API Gateway
PORT=8080

# Database
POSTGRES_HOST=localhost
POSTGRES_USER=aetherium
POSTGRES_PASSWORD=secret
POSTGRES_DB=aetherium

# Redis
REDIS_ADDR=localhost:6379

# Logging
LOKI_URL=http://localhost:3100

# Integrations
GITHUB_TOKEN=ghp_xxx
GITHUB_WEBHOOK_SECRET=xxx
SLACK_BOT_TOKEN=xoxb-xxx
SLACK_SIGNING_SECRET=xxx
```

---

## Production Deployment

### Docker Compose

```yaml
version: '3.8'

services:
  api-gateway:
    image: aetherium/api-gateway:latest
    ports:
      - "8080:8080"
    environment:
      - POSTGRES_HOST=postgres
      - REDIS_ADDR=redis:6379
      - LOKI_URL=http://loki:3100
    depends_on:
      - postgres
      - redis
      - loki

  worker:
    image: aetherium/worker:latest
    privileged: true
    volumes:
      - /var/firecracker:/var/firecracker
    environment:
      - POSTGRES_HOST=postgres
      - REDIS_ADDR=redis:6379

  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: aetherium
      POSTGRES_USER: aetherium
      POSTGRES_PASSWORD: secret
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine

  loki:
    image: grafana/loki:latest
    ports:
      - "3100:3100"

volumes:
  postgres_data:
```

### Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api-gateway
spec:
  replicas: 3
  selector:
    matchLabels:
      app: api-gateway
  template:
    metadata:
      labels:
        app: api-gateway
    spec:
      containers:
      - name: api-gateway
        image: aetherium/api-gateway:latest
        ports:
        - containerPort: 8080
        env:
        - name: POSTGRES_HOST
          value: postgres-service
        - name: REDIS_ADDR
          value: redis-service:6379
        - name: LOKI_URL
          value: http://loki-service:3100
```
