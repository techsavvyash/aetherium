# Aetherium TODO List

## ✅ Completed

- [x] **Debug and fix vsock connection timeout issue**
  - Fixed by using Firecracker's native vsock library
  - Added comprehensive diagnostic tools
  - Vsock communication working between host and guest

- [x] **Create Redis task queue implementation with Asynq**
  - Implemented Asynq-based distributed task queue
  - Defined task types (VM operations, job execution, integrations)
  - Created task handlers with retry logic and priority queues
  - Supports concurrent processing with configurable workers

- [x] **Create PostgreSQL state store with migrations**
  - Designed complete database schema (VMs, Tasks, Jobs, Executions)
  - Implemented repository pattern with interfaces
  - Created PostgreSQL implementation with transactions
  - Migration system with up/down support
  - Migration CLI tool for database setup

## 📋 Infrastructure

- [ ] Implement Loki logging backend
  - Set up Loki client and configuration
  - Integrate structured logging across services
  - Create log aggregation and querying interface

## 🔌 Integration Framework

- [ ] Create integration framework (plugin registry, event bus, SDK)
  - Design plugin architecture
  - Implement event bus for inter-component communication
  - Create SDK for custom plugin development
  - Document plugin development guide

- [ ] Implement GitHub integration (PR creation, webhook handling)
  - GitHub PR creation from task results
  - Webhook handlers for repository events
  - Issue tracking integration
  - Status updates and notifications

- [ ] Implement Slack integration (notifications, slash commands)
  - Slack bot for notifications
  - Slash commands for task management
  - Status updates and alerts
  - Interactive message components

## 🏗️ Core Services

- [ ] Build API Gateway service
  - REST API endpoints for VM management
  - gRPC endpoints for inter-service communication
  - Authentication and authorization
  - Rate limiting and request validation
  - API documentation (OpenAPI/Swagger)

- [ ] Build Task Orchestrator service
  - Task queue consumer
  - Task distribution logic
  - State management integration
  - Task lifecycle management
  - Dead letter queue handling

- [ ] Build Agent Worker service
  - Worker pool management
  - Task execution coordination
  - VM lifecycle management (create, start, execute, stop)
  - Result reporting and error handling
  - Health checking and auto-recovery

## 🎯 Immediate Next Steps

1. **Fix Vsock Issue**
   ```bash
   sudo ./scripts/setup-and-test.sh
   ./scripts/test-and-diagnose.sh
   cat /tmp/firecracker-test-vm.sock.log
   ```

2. **Alternative Path (if vsock debugging takes too long)**
   - Implement TAP device networking for Firecracker VMs
   - Use TCP fallback in agent
   - OR focus on Docker orchestrator first (networking works out of the box)

3. **Once VMM working**
   - Start with Redis + PostgreSQL infrastructure
   - Build minimal API Gateway
   - Implement basic Task Orchestrator

## 📝 Documentation Tasks

- [ ] Add API documentation
- [ ] Create architecture diagrams
- [ ] Write deployment guide
- [ ] Create development setup guide
- [ ] Document configuration options

## 🧪 Testing Tasks

- [ ] Add unit tests for orchestrators
- [ ] Add integration tests for task queue
- [ ] Create end-to-end test suite
- [ ] Set up CI/CD pipeline
- [ ] Add performance benchmarks

## 🔒 Security Tasks

- [ ] Implement VM isolation and security policies
- [ ] Add authentication system
- [ ] Set up secrets management
- [ ] Network security configuration
- [ ] Audit logging

---

**Last Updated:** 2025-10-04
**Priority:** Fix vsock issue → Infrastructure → Core Services → Integrations
