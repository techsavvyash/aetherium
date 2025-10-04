# Aetherium

**Remote VMM Orchestration Engine for Autonomous AI Agents**

Aetherium is a secure, scalable platform for running AI agents in isolated microVMs with command execution capabilities and a plugin-based architecture for maximum flexibility.

## Current Status

✅ **Phase 1 Complete** - VM Management & Command Execution Working
📅 **Last Updated:** 2025-10-04

[See detailed status →](docs/CURRENT_STATUS.md)

## Architecture

- **Control Plane (Go)**: API Gateway, Task Orchestrator, GitHub Integration *(Planned)*
- **Execution Plane (Zig + Go)**: Firecracker VMM Manager + Docker Runtime **✅ Working**
- **Command Execution**: Run arbitrary commands in VMs with stdout/stderr capture **✅ Working**
- **Plugin System**: Extensible integrations (GitHub, Slack, Discord, etc.) *(Planned)*
- **Interface-Driven**: Swap components via configuration **✅ Implemented**

## Features

### ✅ Implemented

- **VMM Orchestration**: Docker (working), Firecracker (core implementation complete)
- **Command Execution**: Execute commands in VMs, capture output, handle exit codes
- **CLI Tools**: Interactive demo mode, programmatic API
- **Interface Design**: All major abstractions defined
- **Configuration System**: YAML-based with env var support

### 🚧 In Progress

- **Pluggable Components**: Task queues (Redis/RabbitMQ), Logging (Loki/ELK), Storage (Postgres/MySQL)
- **Integration Framework**: GitHub, Slack, Discord with extensible plugin SDK
- **Event-Driven**: Pub/sub architecture for decoupled integrations
- **Security**: Hardware-level isolation via Firecracker microVMs

## Quick Start

### Try the Demo

The fastest way to see Aetherium in action:

```bash
# Build the project
make build

# Run the interactive demo
./bin/vm-cli demo
```

This will walk you through:
1. Initializing the Docker orchestrator
2. Creating a VM
3. Starting the VM
4. Executing multiple commands
5. Checking VM status
6. Cleanup

**Press Enter after each step to continue.**

### Manual Usage

```bash
# Initialize orchestrator
./bin/vm-cli init docker

# Create and start a VM (in separate terminals or use the demo)
docker run -d --name my-vm ubuntu:22.04 sleep infinity
docker exec my-vm echo "Hello from VM!"
docker exec my-vm ls -la /
docker exec my-vm pwd

# Or use the working docker-demo
./bin/docker-demo
```

### Prerequisites

**Current Phase:**
- Go 1.23+
- Zig 0.15+ (for Firecracker)
- Docker (for testing)

**Future Phases:**
- PostgreSQL 15+ (state persistence)
- Redis 7+ (task queue)

## Configuration

Edit `config/config.yaml`:

```yaml
task_queue:
  provider: redis  # Switch to: rabbitmq, kafka, memory

logging:
  provider: loki   # Switch to: elasticsearch, cloudwatch, stdout

vmm:
  provider: firecracker  # Switch to: kata, docker, qemu

integrations:
  enabled:
    - github
    - slack
    - discord
```

## Project Structure

```
aetherium/
├── cmd/                # Service entry points
├── pkg/                # Public libraries
│   ├── queue/         # Task queue abstraction
│   ├── storage/       # State store abstraction
│   ├── logging/       # Logging abstraction
│   ├── vmm/           # VMM orchestrator abstraction
│   ├── events/        # Event bus abstraction
│   └── integrations/  # Integration plugins
├── internal/          # Private packages
│   └── firecracker/   # Zig VMM implementation
└── config/            # Configuration files
```

## Development

```bash
# Build all binaries
make build

# Run tests
make test

# Clean build artifacts
make clean
```

### Running Tests

```bash
# All tests
go test ./...

# Specific package
go test ./pkg/vmm/firecracker

# With coverage
go test -cover ./...
```

## Documentation

- **[Current Status](docs/CURRENT_STATUS.md)** - What's working now
- **[VM CLI Guide](docs/VM-CLI-GUIDE.md)** - CLI usage and examples
- **[Command Execution Guide](docs/COMMAND-EXECUTION-GUIDE.md)** - API usage
- **[Implementation Plan](docs/implementation-plan.md)** - Full project roadmap
- **[Firecracker VMM](docs/firecracker-vmm.md)** - Firecracker API reference

## Programmatic Usage

```go
package main

import (
    "context"
    "fmt"
    "github.com/aetherium/aetherium/pkg/types"
    "github.com/aetherium/aetherium/pkg/vmm"
    "github.com/aetherium/aetherium/pkg/vmm/docker"
)

func main() {
    ctx := context.Background()

    // Create orchestrator
    orch, _ := docker.NewDockerOrchestrator(map[string]interface{}{
        "network": "bridge",
        "image":   "ubuntu:22.04",
    })

    // Create and start VM
    vm, _ := orch.CreateVM(ctx, &types.VMConfig{
        ID:       "my-agent",
        VCPUCount: 2,
        MemoryMB: 512,
    })
    orch.StartVM(ctx, vm.ID)

    // Execute command
    result, _ := orch.ExecuteCommand(ctx, vm.ID, &vmm.Command{
        Cmd:  "echo",
        Args: []string{"Hello from VM!"},
    })

    fmt.Printf("Output: %s\n", result.Stdout)
    fmt.Printf("Exit Code: %d\n", result.ExitCode)

    // Cleanup
    orch.StopVM(ctx, vm.ID, false)
    orch.DeleteVM(ctx, vm.ID)
}
```

## Next Steps

See the [todo list](docs/CURRENT_STATUS.md#-pending-components) for upcoming features:

1. DI Container & Factory Pattern
2. In-Memory Providers (for testing)
3. Redis Task Queue
4. PostgreSQL State Store
5. Integration Framework

## License

MIT
