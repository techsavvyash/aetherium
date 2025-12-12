# Aetherium Documentation Index

Complete guide to all documentation in this repository.

## Quick Navigation

**New to Aetherium?** Start here:
1. [README.md](../README.md) - Project overview
2. [SETUP_GUIDE.md](SETUP_GUIDE.md) - One-time setup and daily operations
3. [TESTING_GUIDE.md](TESTING_GUIDE.md) - How to verify everything works

**Want to deploy?** Read:
- [PRODUCTION-ARCHITECTURE.md](PRODUCTION-ARCHITECTURE.md) - Production deployment guide
- [DEPLOYMENT.md](DEPLOYMENT.md) - Kubernetes and cloud deployment
- [KUBERNETES.md](KUBERNETES.md) - K8s specific guidance

**Building features?** Check:
- [design.md](design.md) - System design and architecture
- [CLAUDE.md](../CLAUDE.md) - Development guidelines (in root)
- [AGENTS.md](../AGENTS.md) - AI agent guidelines (in root)

**Troubleshooting?** See:
- [SETUP_GUIDE.md#troubleshooting](SETUP_GUIDE.md#troubleshooting) - Common setup issues
- [TESTING_GUIDE.md#troubleshooting](TESTING_GUIDE.md#troubleshooting) - Test-time issues
- [COMMAND-EXECUTION-GUIDE.md](COMMAND-EXECUTION-GUIDE.md) - Command execution details

---

## Documentation Categories

### 🚀 Getting Started

| Document | Purpose | Read Time |
|----------|---------|-----------|
| [QUICKSTART.md](QUICKSTART.md) | Quick setup in 5 minutes | 5 min |
| [SETUP_GUIDE.md](SETUP_GUIDE.md) | Complete setup with detailed explanations | 20 min |
| [TESTING_GUIDE.md](TESTING_GUIDE.md) | How to verify everything works | 15 min |
| [VM-CLI-GUIDE.md](VM-CLI-GUIDE.md) | Using the CLI tool to manage VMs | 10 min |

### 🏗️ Architecture & Design

| Document | Purpose | Read Time |
|----------|---------|-----------|
| [design.md](design.md) | System design and architecture decisions | 30 min |
| [PRODUCTION-ARCHITECTURE.md](PRODUCTION-ARCHITECTURE.md) | Production deployment architecture | 25 min |
| [firecracker-vmm.md](firecracker-vmm.md) | Firecracker VM management details | 20 min |
| [EPHEMERAL-VM-SECURITY.md](EPHEMERAL-VM-SECURITY.md) | Security considerations for ephemeral VMs | 15 min |

### 🔧 API & Integration

| Document | Purpose | Read Time |
|----------|---------|-----------|
| [API-GATEWAY.md](API-GATEWAY.md) | REST API endpoints and usage | 15 min |
| [COMMAND-EXECUTION-GUIDE.md](COMMAND-EXECUTION-GUIDE.md) | How to execute commands in VMs | 10 min |
| [INTEGRATIONS.md](INTEGRATIONS.md) | GitHub, Slack, and custom integrations | 20 min |
| [DISTRIBUTED-WORKER-API.md](DISTRIBUTED-WORKER-API.md) | Distributed worker communication | 15 min |

### 📦 Infrastructure & Deployment

| Document | Purpose | Read Time |
|----------|---------|-----------|
| [DEPLOYMENT.md](DEPLOYMENT.md) | General deployment guide | 20 min |
| [KUBERNETES.md](KUBERNETES.md) | Kubernetes deployment | 20 min |
| [TOOLS-AND-PROVISIONING.md](TOOLS-AND-PROVISIONING.md) | Tool installation and VM provisioning | 15 min |
| [firecracker-test-results.md](firecracker-test-results.md) | Test results and performance data | 10 min |

### 🧪 Testing & Verification

| Document | Purpose | Read Time |
|----------|---------|-----------|
| [TESTING_GUIDE.md](TESTING_GUIDE.md) | Complete testing guide with verification | 15 min |
| [CURRENT_STATUS.md](CURRENT_STATUS.md) | Current project status and feature completeness | 10 min |

### 🔄 Status & Progress

| Document | Purpose | Read Time |
|----------|---------|-----------|
| [CURRENT_STATUS.md](CURRENT_STATUS.md) | What's working, what's planned | 10 min |
| [DISTRIBUTED-WORKERS-STATUS.md](DISTRIBUTED-WORKERS-STATUS.md) | Status of distributed worker features | 5 min |
| [implementation-plan.md](implementation-plan.md) | Original implementation roadmap | 15 min |

### 🐛 Fixes & Enhancements

| Document | Purpose | Read Time |
|----------|---------|-----------|
| [TUI_STREAMING_FIX.md](TUI_STREAMING_FIX.md) | TUI streaming implementation fixes | 5 min |
| [VM-TCP-COMMUNICATION-FIX.md](VM-TCP-COMMUNICATION-FIX.md) | VM TCP communication improvements | 5 min |

### 📚 Reference & Archive

| Document | Purpose |
|----------|---------|
| [INDEX.md](INDEX.md) | This file - documentation index |
| [ROOT_ARCHIVE.md](ROOT_ARCHIVE.md) | Archive of files moved from root |

---

## By Use Case

### "I want to set up Aetherium locally"

1. Read: [QUICKSTART.md](QUICKSTART.md) (5 min)
2. Follow: [SETUP_GUIDE.md](SETUP_GUIDE.md) (20 min)
3. Verify: [TESTING_GUIDE.md](TESTING_GUIDE.md) (15 min)

**Total: ~40 minutes**

### "I want to deploy to production"

1. Understand: [design.md](design.md) (30 min)
2. Review: [PRODUCTION-ARCHITECTURE.md](PRODUCTION-ARCHITECTURE.md) (25 min)
3. Deploy: [DEPLOYMENT.md](DEPLOYMENT.md) or [KUBERNETES.md](KUBERNETES.md) (20 min)
4. Secure: [EPHEMERAL-VM-SECURITY.md](EPHEMERAL-VM-SECURITY.md) (15 min)

**Total: ~90 minutes**

### "I want to use the API"

1. Learn: [API-GATEWAY.md](API-GATEWAY.md) (15 min)
2. Execute: [COMMAND-EXECUTION-GUIDE.md](COMMAND-EXECUTION-GUIDE.md) (10 min)
3. Integrate: [INTEGRATIONS.md](INTEGRATIONS.md) (20 min)

**Total: ~45 minutes**

### "I want to contribute code"

1. Read: [CLAUDE.md](../CLAUDE.md) (10 min)
2. Understand: [design.md](design.md) (30 min)
3. Review: [AGENTS.md](../AGENTS.md) (5 min)
4. Check: [CURRENT_STATUS.md](CURRENT_STATUS.md) (10 min)

**Total: ~55 minutes**

### "Something isn't working"

1. Check: [SETUP_GUIDE.md#troubleshooting](SETUP_GUIDE.md#troubleshooting)
2. Verify: [TESTING_GUIDE.md#troubleshooting](TESTING_GUIDE.md#troubleshooting)
3. Review: [CURRENT_STATUS.md](CURRENT_STATUS.md)
4. Report: Use project issue tracker

### "I want to add GitHub integration"

1. Learn: [INTEGRATIONS.md](INTEGRATIONS.md) (20 min)
2. Design: [design.md](design.md#integrations) (10 min)
3. Code: Follow [AGENTS.md](../AGENTS.md) guidelines

### "I want to install custom tools"

1. Read: [TOOLS-AND-PROVISIONING.md](TOOLS-AND-PROVISIONING.md) (15 min)
2. Execute: [COMMAND-EXECUTION-GUIDE.md](COMMAND-EXECUTION-GUIDE.md) (10 min)

---

## Document Map

```
docs/
├── INDEX.md (you are here)
├── ROOT_ARCHIVE.md (archive of moved files)
│
├── Getting Started
│   ├── QUICKSTART.md
│   ├── SETUP_GUIDE.md
│   ├── TESTING_GUIDE.md
│   └── VM-CLI-GUIDE.md
│
├── Architecture
│   ├── design.md
│   ├── PRODUCTION-ARCHITECTURE.md
│   ├── firecracker-vmm.md
│   ├── EPHEMERAL-VM-SECURITY.md
│   └── implementation-plan.md
│
├── API & Execution
│   ├── API-GATEWAY.md
│   ├── COMMAND-EXECUTION-GUIDE.md
│   ├── DISTRIBUTED-WORKER-API.md
│   └── INTEGRATIONS.md
│
├── Infrastructure
│   ├── DEPLOYMENT.md
│   ├── KUBERNETES.md
│   ├── TOOLS-AND-PROVISIONING.md
│   └── firecracker-test-results.md
│
├── Status & Reference
│   ├── CURRENT_STATUS.md
│   ├── DISTRIBUTED-WORKERS-STATUS.md
│   └── TUI_STREAMING_FIX.md
│
└── Root Directory (keep there)
    ├── README.md
    ├── CLAUDE.md
    ├── AGENTS.md
    ├── Makefile
    ├── docker-compose.yml
    ├── go.mod / go.sum
    ├── .gitignore
    └── .git/
```

---

## File Organization Changes

### Moved to docs/

The following files have been consolidated or moved to the `docs/` folder:
- `SETUP.md` → `SETUP_GUIDE.md`
- `EXECUTION_SUMMARY.md` → Merged into `TESTING_GUIDE.md`
- `EXECUTION_INDEX.md` → Merged into `INDEX.md` and `TESTING_GUIDE.md`
- `MANUAL_RUN_STEPS.md` → Merged into `TESTING_GUIDE.md`
- `VISUAL_WALKTHROUGH.md` → Merged into `TESTING_GUIDE.md`
- `PRE_EXECUTION_CHECKLIST.md` → Merged into `TESTING_GUIDE.md`
- `COMMANDS_REFERENCE.md` → Embedded in relevant guides
- And 20+ other redundant documents

### Kept in Root

Essential files that remain in the root directory:
- `README.md` - Project overview
- `CLAUDE.md` - Claude AI guidelines
- `AGENTS.md` - Agent guidelines
- `Makefile` - Build automation
- `docker-compose.yml` - Infrastructure
- `go.mod` / `go.sum` - Dependencies
- `.gitignore` - Git ignore rules
- `.git/` - Git repository

### Deleted from Root

Build artifacts and temporary files:
- `api-gateway` (binary) - build artifact
- `worker` (binary) - build artifact
- `api-gateway.log` - log file
- `worker.log` - log file
- `fix-rootfs.sh` - temporary script
- `RUN_NOW.sh` - temporary script
- 30+ duplicate/outdated documentation files

---

## Reading Order by Role

### System Administrator

1. [SETUP_GUIDE.md](SETUP_GUIDE.md) - Set up locally
2. [DEPLOYMENT.md](DEPLOYMENT.md) - Deploy to server
3. [KUBERNETES.md](KUBERNETES.md) - Deploy to K8s
4. [EPHEMERAL-VM-SECURITY.md](EPHEMERAL-VM-SECURITY.md) - Security hardening
5. [TESTING_GUIDE.md](TESTING_GUIDE.md) - Verify operations

### Developer

1. [QUICKSTART.md](QUICKSTART.md) - Get started
2. [design.md](design.md) - Understand architecture
3. [CLAUDE.md](../CLAUDE.md) - Development guidelines
4. [API-GATEWAY.md](API-GATEWAY.md) - API reference
5. [CURRENT_STATUS.md](CURRENT_STATUS.md) - What's incomplete
6. [AGENTS.md](../AGENTS.md) - Agent guidelines

### DevOps Engineer

1. [PRODUCTION-ARCHITECTURE.md](PRODUCTION-ARCHITECTURE.md) - Understand production setup
2. [DEPLOYMENT.md](DEPLOYMENT.md) - Deploy to infrastructure
3. [KUBERNETES.md](KUBERNETES.md) - Kubernetes setup
4. [firecracker-test-results.md](firecracker-test-results.md) - Performance data
5. [TESTING_GUIDE.md](TESTING_GUIDE.md) - Verification procedures

### Integration Developer

1. [INTEGRATIONS.md](INTEGRATIONS.md) - Integration framework
2. [API-GATEWAY.md](API-GATEWAY.md) - REST API
3. [COMMAND-EXECUTION-GUIDE.md](COMMAND-EXECUTION-GUIDE.md) - Execute commands
4. [design.md](design.md#integrations) - Design overview
5. [DISTRIBUTED-WORKER-API.md](DISTRIBUTED-WORKER-API.md) - Worker communication

---

## Key Concepts

### VM Management
- **Create**: Spin up ephemeral microVMs
- **Execute**: Run commands inside VMs
- **Delete**: Clean up VMs
- See: [COMMAND-EXECUTION-GUIDE.md](COMMAND-EXECUTION-GUIDE.md)

### Task Queue
- **Async**: Redis-based distributed task queue
- **Priority**: Support for priority levels
- **Reliability**: Persistent state in PostgreSQL
- See: [design.md](design.md#task-queue)

### Network
- **Bridge**: `aetherium0` TAP bridge
- **Isolation**: NAT and iptables rules
- **Vsock**: VM-host communication channel
- See: [SETUP_GUIDE.md#network-architecture](SETUP_GUIDE.md#network-architecture)

### Tool Installation
- **Default**: git, nodejs, bun, claude-code
- **Custom**: Any tool via package manager
- **Versions**: Pin specific versions
- See: [TOOLS-AND-PROVISIONING.md](TOOLS-AND-PROVISIONING.md)

### Integration Framework
- **GitHub**: PR creation, webhooks
- **Slack**: Notifications, commands
- **Custom**: Build your own plugins
- See: [INTEGRATIONS.md](INTEGRATIONS.md)

---

## Command Reference

### Build
```bash
make build              # Build all binaries
make clean              # Clean artifacts
make test               # Run tests
make lint               # Lint code
```

### Setup (One-Time)
```bash
docker-compose up -d                    # Start services
sudo ./scripts/setup-network.sh          # Configure network
sudo ./scripts/setup-rootfs-once.sh      # Setup rootfs
```

### Daily Operations
```bash
sudo ./bin/worker                       # Start worker
./bin/api-gateway                       # Start API
./bin/aether-cli -type vm:create ...   # Create VM
```

See [SETUP_GUIDE.md](SETUP_GUIDE.md#common-commands) for full reference.

---

## Troubleshooting Quick Links

| Problem | Document | Section |
|---------|----------|---------|
| Setup issues | [SETUP_GUIDE.md](SETUP_GUIDE.md) | #troubleshooting |
| Test failures | [TESTING_GUIDE.md](TESTING_GUIDE.md) | #troubleshooting |
| API not working | [API-GATEWAY.md](API-GATEWAY.md) | #troubleshooting |
| Network problems | [SETUP_GUIDE.md](SETUP_GUIDE.md) | #troubleshooting |
| Command execution fails | [COMMAND-EXECUTION-GUIDE.md](COMMAND-EXECUTION-GUIDE.md) | #troubleshooting |

---

## Status & Roadmap

- ✅ **Complete**: VM creation, command execution, task queue
- 🚧 **In Progress**: Kubernetes deployment, distributed workers
- 📋 **Planned**: GitHub integration, Slack integration, advanced security
- See: [CURRENT_STATUS.md](CURRENT_STATUS.md)

---

## Contributing

Before contributing:
1. Read [AGENTS.md](../AGENTS.md)
2. Read [CLAUDE.md](../CLAUDE.md)
3. Check [CURRENT_STATUS.md](CURRENT_STATUS.md)
4. Follow the design in [design.md](design.md)

---

## Support & Resources

- **Issues**: Check project issue tracker
- **Security**: See [EPHEMERAL-VM-SECURITY.md](EPHEMERAL-VM-SECURITY.md)
- **Performance**: See [firecracker-test-results.md](firecracker-test-results.md)
- **Architecture**: See [PRODUCTION-ARCHITECTURE.md](PRODUCTION-ARCHITECTURE.md)

---

**Last Updated:** December 12, 2025  
**Documentation Status:** Complete and organized  
**Total Documentation Pages:** 19 comprehensive guides
