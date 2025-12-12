# Shared Types Library

Type definitions and models used across all Aetherium services.

## Packages

### pkg/domain/

Core domain models:

```go
import "github.com/aetherium/aetherium/libs/types/pkg/domain"

// VM
type VM struct {
    ID string
    Name string
    Status string
    VCPUCount int
    MemoryMB int
    CreatedAt time.Time
}

// Task
type Task struct {
    ID string
    Type string
    Status string
    Payload map[string]interface{}
    CreatedAt time.Time
}

// Execution
type Execution struct {
    ID string
    VMId string
    Command string
    ExitCode int
    Stdout string
    Stderr string
    StartedAt time.Time
    CompletedAt time.Time
}
```

### pkg/api/

API request and response types:

```go
import "github.com/aetherium/aetherium/libs/types/pkg/api"

type CreateVMRequest struct {
    Name string `json:"name"`
    VCPUs int `json:"vcpus"`
    MemoryMB int `json:"memory_mb"`
}

type VMResponse struct {
    ID string `json:"id"`
    Name string `json:"name"`
    Status string `json:"status"`
}
```

### pkg/events/

Event type definitions:

```go
import "github.com/aetherium/aetherium/libs/types/pkg/events"

type VMCreatedEvent struct {
    VMID string
    Name string
    Timestamp time.Time
}

type CommandExecutedEvent struct {
    VMID string
    Command string
    ExitCode int
    Timestamp time.Time
}
```

## Usage

All services import from this library for type definitions:

```go
import (
    "github.com/aetherium/aetherium/libs/types/pkg/domain"
    "github.com/aetherium/aetherium/libs/types/pkg/api"
)
```

## Building

```bash
cd libs/types
go mod tidy
go build ./...
```

## Testing

```bash
go test ./...
```

## Adding New Types

When adding new type definitions:

1. Determine category: domain, api, or events
2. Add to appropriate package
3. Add JSON tags for API types
4. Update this README
5. Update root go.work
