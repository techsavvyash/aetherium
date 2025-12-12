# Quick Navigation Guide

**Unsure where to start?** This page will guide you to the right documentation.

## First Time Setup?

Follow this path:
1. **[SETUP_GUIDE.md](SETUP_GUIDE.md)** - Step-by-step setup instructions
2. **[TESTING_GUIDE.md](TESTING_GUIDE.md)** - Verify everything works
3. **[QUICKSTART.md](QUICKSTART.md)** - Try it out quickly

**Time estimate:** 40 minutes total

---

## Want to Get Started Fast?

Read **[QUICKSTART.md](QUICKSTART.md)** in 5 minutes, then dive in.

---

## Looking for Something Specific?

### Installation & Setup
- **I want to set up Aetherium locally** → [SETUP_GUIDE.md](SETUP_GUIDE.md)
- **I want to deploy to production** → [PRODUCTION-ARCHITECTURE.md](PRODUCTION-ARCHITECTURE.md) + [DEPLOYMENT.md](DEPLOYMENT.md)
- **I want to deploy to Kubernetes** → [KUBERNETES.md](KUBERNETES.md)
- **I want to verify my setup** → [TESTING_GUIDE.md](TESTING_GUIDE.md)

### Using Aetherium
- **I want to create and manage VMs** → [VM-CLI-GUIDE.md](VM-CLI-GUIDE.md)
- **I want to execute commands in VMs** → [COMMAND-EXECUTION-GUIDE.md](COMMAND-EXECUTION-GUIDE.md)
- **I want to use the REST API** → [API-GATEWAY.md](API-GATEWAY.md)
- **I want to install custom tools** → [TOOLS-AND-PROVISIONING.md](TOOLS-AND-PROVISIONING.md)

### Integration & Extension
- **I want to integrate with GitHub/Slack** → [INTEGRATIONS.md](INTEGRATIONS.md)
- **I want to build a custom integration** → [INTEGRATIONS.md](INTEGRATIONS.md)
- **I want to understand the worker API** → [DISTRIBUTED-WORKER-API.md](DISTRIBUTED-WORKER-API.md)

### Architecture & Design
- **I want to understand the system design** → [design.md](design.md)
- **I want to see the production architecture** → [PRODUCTION-ARCHITECTURE.md](PRODUCTION-ARCHITECTURE.md)
- **I want to understand Firecracker integration** → [firecracker-vmm.md](firecracker-vmm.md)
- **I want to understand the implementation plan** → [implementation-plan.md](implementation-plan.md)

### Troubleshooting
- **Something isn't working during setup** → [SETUP_GUIDE.md#troubleshooting](SETUP_GUIDE.md#troubleshooting)
- **Something isn't working during tests** → [TESTING_GUIDE.md#troubleshooting](TESTING_GUIDE.md#troubleshooting)
- **My API calls are failing** → [API-GATEWAY.md](API-GATEWAY.md#troubleshooting)
- **I'm getting network errors** → [SETUP_GUIDE.md#troubleshooting](SETUP_GUIDE.md#troubleshooting)

### Project Status
- **What's working now?** → [CURRENT_STATUS.md](CURRENT_STATUS.md)
- **What's being worked on?** → [CURRENT_STATUS.md](CURRENT_STATUS.md)
- **What's planned next?** → [CURRENT_STATUS.md](CURRENT_STATUS.md)

### Security
- **I want to secure my deployment** → [EPHEMERAL-VM-SECURITY.md](EPHEMERAL-VM-SECURITY.md)
- **I want to understand isolation** → [EPHEMERAL-VM-SECURITY.md](EPHEMERAL-VM-SECURITY.md) + [design.md](design.md)

### Performance & Testing
- **I want to see performance benchmarks** → [firecracker-test-results.md](firecracker-test-results.md)
- **I want to run tests** → [TESTING_GUIDE.md](TESTING_GUIDE.md)

---

## By Role

### System Administrator

**Reading order:**
1. [QUICKSTART.md](QUICKSTART.md) - 5 min overview
2. [SETUP_GUIDE.md](SETUP_GUIDE.md) - Installation guide
3. [TESTING_GUIDE.md](TESTING_GUIDE.md) - Verification procedures
4. [DEPLOYMENT.md](DEPLOYMENT.md) - Deployment guide
5. [KUBERNETES.md](KUBERNETES.md) - If using Kubernetes
6. [PRODUCTION-ARCHITECTURE.md](PRODUCTION-ARCHITECTURE.md) - Production setup
7. [EPHEMERAL-VM-SECURITY.md](EPHEMERAL-VM-SECURITY.md) - Security hardening

### Developer

**Reading order:**
1. [QUICKSTART.md](QUICKSTART.md) - 5 min overview
2. [CLAUDE.md](../CLAUDE.md) - Development guidelines (root directory)
3. [design.md](design.md) - System architecture
4. [API-GATEWAY.md](API-GATEWAY.md) - API reference
5. [COMMAND-EXECUTION-GUIDE.md](COMMAND-EXECUTION-GUIDE.md) - Execution details
6. [AGENTS.md](../AGENTS.md) - Agent guidelines (root directory)

### DevOps Engineer

**Reading order:**
1. [QUICKSTART.md](QUICKSTART.md) - 5 min overview
2. [PRODUCTION-ARCHITECTURE.md](PRODUCTION-ARCHITECTURE.md) - Architecture
3. [DEPLOYMENT.md](DEPLOYMENT.md) - General deployment
4. [KUBERNETES.md](KUBERNETES.md) - K8s deployment
5. [firecracker-test-results.md](firecracker-test-results.md) - Performance data
6. [EPHEMERAL-VM-SECURITY.md](EPHEMERAL-VM-SECURITY.md) - Security setup
7. [TESTING_GUIDE.md](TESTING_GUIDE.md) - Verification

### Integration Developer

**Reading order:**
1. [QUICKSTART.md](QUICKSTART.md) - 5 min overview
2. [INTEGRATIONS.md](INTEGRATIONS.md) - Integration framework
3. [API-GATEWAY.md](API-GATEWAY.md) - REST API
4. [COMMAND-EXECUTION-GUIDE.md](COMMAND-EXECUTION-GUIDE.md) - Execution API
5. [DISTRIBUTED-WORKER-API.md](DISTRIBUTED-WORKER-API.md) - Worker API
6. [design.md](design.md) - System design

---

## File Organization

```
Root Directory (Essential only):
  ├── README.md                           Project overview
  ├── CLAUDE.md                          Developer guidelines
  ├── AGENTS.md                          Agent guidelines
  ├── Makefile                           Build automation
  ├── docker-compose.yml                 Infrastructure
  ├── go.mod / go.sum                   Dependencies
  └── .gitignore / .git                 Git files

Docs Directory (23 comprehensive guides):
  ├── INDEX.md                          Master index (you should read this first!)
  ├── NAVIGATION.md                     This file
  │
  ├── Getting Started:
  │   ├── QUICKSTART.md
  │   ├── SETUP_GUIDE.md
  │   ├── TESTING_GUIDE.md
  │   └── VM-CLI-GUIDE.md
  │
  ├── Architecture:
  │   ├── design.md
  │   ├── PRODUCTION-ARCHITECTURE.md
  │   ├── firecracker-vmm.md
  │   ├── EPHEMERAL-VM-SECURITY.md
  │   └── implementation-plan.md
  │
  ├── API & Execution:
  │   ├── API-GATEWAY.md
  │   ├── COMMAND-EXECUTION-GUIDE.md
  │   ├── DISTRIBUTED-WORKER-API.md
  │   └── INTEGRATIONS.md
  │
  ├── Deployment:
  │   ├── DEPLOYMENT.md
  │   ├── KUBERNETES.md
  │   ├── TOOLS-AND-PROVISIONING.md
  │   └── firecracker-test-results.md
  │
  └── Status & Reference:
      ├── CURRENT_STATUS.md
      ├── DISTRIBUTED-WORKERS-STATUS.md
      ├── TUI_STREAMING_FIX.md
      ├── VM-TCP-COMMUNICATION-FIX.md
      └── ROOT_ARCHIVE.md
```

---

## Quick Commands

### Build
```bash
make build              # Build all binaries
make test               # Run tests
make lint               # Lint code
```

### Setup (One-Time)
```bash
docker-compose up -d                    # Start infrastructure
sudo ./scripts/setup-network.sh          # Configure network
sudo ./scripts/setup-rootfs-once.sh      # Setup rootfs template
```

### Daily Operations
```bash
sudo ./bin/worker                       # Start worker
./bin/api-gateway                       # Start API (optional)
./bin/aether-cli -type vm:create ...   # Create VM
```

See [SETUP_GUIDE.md](SETUP_GUIDE.md) for detailed commands.

---

## Need More Help?

1. **Search** the documentation using Ctrl+F
2. **Browse** [INDEX.md](INDEX.md) for complete navigation
3. **Check** [CURRENT_STATUS.md](CURRENT_STATUS.md) for what's implemented
4. **Review** [SETUP_GUIDE.md#troubleshooting](SETUP_GUIDE.md#troubleshooting) for common issues
5. **Report** issues on the project tracker

---

## Document Versions

- **Last Updated:** December 12, 2025
- **Status:** Complete and organized
- **Total Guides:** 23 comprehensive documents
- **Total Lines:** 12,000+ lines of documentation

---

**👉 Next Step:** Start with [INDEX.md](INDEX.md) for the master index, or go directly to the guide that matches your needs above.
