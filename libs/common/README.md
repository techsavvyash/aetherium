# Common Libraries

Shared utilities used across all Aetherium services.

## Packages

### pkg/logging/

Logging abstractions and implementations.

- Stdout logger for local development
- Loki logger for production

```go
import "github.com/aetherium/aetherium/libs/common/pkg/logging"

logger := logging.NewStdoutLogger()
logger.Log(ctx, "User message")
```

### pkg/config/

Configuration management with YAML support.

```go
import "github.com/aetherium/aetherium/libs/common/pkg/config"

cfg := config.Load("config.yaml")
vmm := cfg.GetString("vmm.provider")
```

### pkg/container/

Dependency injection container.

```go
import "github.com/aetherium/aetherium/libs/common/pkg/container"

container := container.New()
container.Register("service", &MyService{})
service := container.Get("service")
```

### pkg/events/

Event bus abstractions for pub/sub patterns.

```go
import "github.com/aetherium/aetherium/libs/common/pkg/events"

bus.Publish(ctx, "vm.created", vmCreatedEvent)
bus.Subscribe(ctx, "vm.created", handler)
```

## Usage

All services import from this library for shared utilities:

```go
import (
    "github.com/aetherium/aetherium/libs/common/pkg/logging"
    "github.com/aetherium/aetherium/libs/common/pkg/config"
)
```

## Building

```bash
cd libs/common
go mod tidy
go build ./...
```

## Testing

```bash
go test ./...
```

## Adding New Utilities

When adding new shared utilities:

1. Create new package in `pkg/{name}/`
2. Define interface first
3. Implement with multiple backends
4. Add tests
5. Update this README
6. Update root go.work
