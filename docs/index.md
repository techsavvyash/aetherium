# Aetherium Documentation

Distributed task execution platform for running isolated workloads in Firecracker microVMs.

## Getting Started

- [Quick Start](quickstart.md) - Get up and running in 5 minutes
- [Setup Guide](setup.md) - Detailed installation and configuration
- [Architecture Overview](architecture.md) - System design and components

## Core Features

- **VM Orchestration** - Firecracker microVMs with Docker support
- **Command Execution** - Execute commands in VMs via vsock or TCP
- **REST API** - Full API gateway with JSON endpoints
- **Task Queue** - Redis-backed async task processing
- **Database** - PostgreSQL state persistence
- **Logging** - Loki centralized logging
- **Integrations** - GitHub, Slack, Discord plugins

## Development

- [Monorepo Guide](monorepo.md) - Project structure and conventions
- [API Reference](api-gateway.md) - REST API documentation
- [Tool Installation](tools-provisioning.md) - How tools are installed
- [Integration Framework](integrations.md) - Building plugins
- [Production Deployment](deployment.md) - Docker/Kubernetes setup

## Common Issues

- [Vsock Connection](troubleshooting/vsock-connection.md) - Fixing vsock timeouts
- [VM Creation](troubleshooting/vm-creation.md) - VM creation failures
- [Tool Installation](troubleshooting/tools.md) - Tool installation issues

## Architecture

```
API Gateway (port 8080)
    ↓
Redis Queue (Asynq)
    ↓
Worker Process
    ↓
Firecracker Orchestrator
    ↓
microVMs (isolated execution)
```

For detailed documentation on each component, see the individual files in this directory.
