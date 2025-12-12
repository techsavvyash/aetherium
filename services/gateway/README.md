# Aetherium Gateway Service

**REST API Gateway and Integration Framework.**

Gateway service provides the HTTP API interface for Aetherium, handling client requests, integration plugins, authentication, and WebSocket connections.

## Features

- **REST API**: Full HTTP API for VM and task operations
- **Integration Plugins**: GitHub, Slack, Discord, and extensible framework
- **Authentication**: JWT and API key support
- **WebSocket**: Real-time updates and log streaming
- **CORS**: Cross-origin resource sharing
- **Middleware**: Logging, rate limiting, request validation

## Building

```bash
# From root
cd services/gateway
make build

# Output to ../../bin/api-gateway
./../../bin/api-gateway
```

## Testing

```bash
# Run all tests
make test

# With coverage
make test-coverage
```

## Running

```bash
# From service directory
make run

# Or manually
go run ./cmd/api-gateway
```

### Connecting to Core Service

Gateway imports the Core service and uses its `TaskService` to perform operations:

```go
import "github.com/aetherium/aetherium/services/core/pkg/service"
```

See `cmd/api-gateway/main.go` for initialization.

## REST API

### VMs

```
POST   /api/v1/vms                 Create VM
GET    /api/v1/vms                 List VMs
GET    /api/v1/vms/:id             Get VM
DELETE /api/v1/vms/:id             Delete VM
```

### Commands

```
POST   /api/v1/vms/:id/execute     Execute command
GET    /api/v1/vms/:id/executions  List execution history
```

### System

```
GET    /health                     Health check
POST   /api/v1/logs/query          Query logs (Loki)
```

## Integrations

### Architecture

```
Integration Plugin
  ├─ Name() string
  ├─ Initialize(ctx, config) error
  ├─ SendNotification(ctx, notif) error
  ├─ HandleEvent(ctx, event) error
  └─ Health(ctx) error
```

### Available Integrations

- **GitHub**: PR creation, webhook listeners
- **Slack**: Message posting, slash commands
- **Discord**: (Planned)
- **Custom**: Implement `integrations.Integration` interface

### Using Integrations

```go
// Register
integration := github.NewGitHubIntegration(config)
registry.Register(integration)

// Use
integration.SendNotification(ctx, notification)
```

## WebSocket

Real-time updates via WebSocket connections:

```
WS     /api/v1/ws                  WebSocket endpoint
```

Receive:
- Log streams
- Task updates
- VM status changes
- Integration notifications

## Authentication

### API Keys

```
Authorization: Bearer YOUR_API_KEY
```

### JWT Tokens

```
Authorization: Bearer JWT_TOKEN
```

## Configuration

```yaml
gateway:
  port: 8080
  log_level: info

core_service:
  url: http://localhost:50051  # or gRPC endpoint

integrations:
  github:
    token: ${GITHUB_TOKEN}
    webhook_secret: ${GITHUB_WEBHOOK_SECRET}
  
  slack:
    bot_token: ${SLACK_BOT_TOKEN}
    signing_secret: ${SLACK_SIGNING_SECRET}
```

## Architecture

### Packages

- **pkg/api/** - REST handlers and routing
- **pkg/middleware/** - HTTP middleware (CORS, logging, auth)
- **pkg/auth/** - Authentication (JWT, API keys)
- **pkg/integrations/** - Integration plugins
- **pkg/websocket/** - WebSocket handling

### Request Flow

```
HTTP Request
    ↓
Chi Router
    ↓
Middleware (auth, CORS, logging)
    ↓
API Handler
    ↓
TaskService (from Core)
    ↓
Response
```

## Imports from Core

Gateway imports from core service:

```go
import (
    "github.com/aetherium/aetherium/services/core/pkg/service"
    "github.com/aetherium/aetherium/services/core/pkg/types"
)
```

Use these to:
- Create/manage VMs
- Execute commands
- Get status
- Store results

## Exports to Clients

This service exports:
- REST API (HTTP)
- WebSocket connections
- Integration webhooks

See `services/dashboard/` for frontend that uses this API.

## Development

### Adding a New Endpoint

1. Create handler in `pkg/api/handlers.go` or new file
2. Register route in `cmd/api-gateway/main.go`
3. Add middleware if needed
4. Write tests in `pkg/api/handlers_test.go`

### Adding a New Integration

1. Create file in `pkg/integrations/{name}/`
2. Implement `Integration` interface
3. Register in integration registry
4. Add configuration

## Troubleshooting

### Can't connect to core service

```
Error: failed to connect to core service
```

Check:
1. Core service is running
2. Connection URL is correct in config
3. Network connectivity: `curl http://core:8000/health`

### API returns 500 errors

Check:
1. Core service is healthy
2. PostgreSQL is running
3. Redis is running
4. Check logs: `docker logs aetherium-gateway`

### WebSocket connections drop

Check:
1. Client is sending pings
2. Server is configured for proper timeouts
3. Network is stable

## Next Steps

- See [../../docs/ARCHITECTURE.md](../../docs/ARCHITECTURE.md) for system design
- See [CURRENT_VS_PROPOSED.md](../../CURRENT_VS_PROPOSED.md) for API documentation
- See [../../MONOREPO_QUICK_REFERENCE.md](../../MONOREPO_QUICK_REFERENCE.md) for quick lookup
