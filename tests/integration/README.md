# Integration Tests

This directory contains integration tests for Aetherium's distributed task execution system.

## Prerequisites

Before running integration tests, ensure:

1. **Infrastructure is running:**
   ```bash
   # Start PostgreSQL
   docker run -d \
     --name aetherium-postgres \
     -e POSTGRES_USER=aetherium \
     -e POSTGRES_PASSWORD=aetherium \
     -e POSTGRES_DB=aetherium \
     -p 5432:5432 \
     postgres:15-alpine

   # Start Redis
   docker run -d \
     --name aetherium-redis \
     -p 6379:6379 \
     redis:7-alpine
   ```

2. **Database migrations are run:**
   ```bash
   # From project root
   cd migrations
   migrate -database "postgres://aetherium:aetherium@localhost:5432/aetherium?sslmode=disable" -path . up
   ```

3. **Firecracker setup:**
   - Firecracker installed and in PATH
   - Kernel at `/var/firecracker/vmlinux`
   - Rootfs at `/var/firecracker/rootfs.ext4`
   - fc-agent deployed in rootfs
   - vhost-vsock module loaded: `sudo modprobe vhost_vsock`

4. **Rootfs has git installed** (or test will attempt to install it)

## Running Tests

### Three VM Clone Test

This test demonstrates distributed task execution by:
- Creating 3 VMs in parallel
- Cloning 3 different GitHub repositories:
  - https://github.com/techsavvyash/vync
  - https://github.com/try-veil/veil
  - https://github.com/try-veil/web
- Verifying the clones
- Cleaning up VMs

Run the test:

```bash
cd tests/integration
sudo go test -v -run TestThreeVMClone -timeout 30m
```

**Note:** `sudo` is required for Firecracker operations.

Expected output:
```
=== Aetherium Integration Test: 3 VM Clone ===
Connecting to PostgreSQL...
Connecting to Redis...
Initializing Firecracker orchestrator...
Starting worker...
Worker started successfully

=== Step 1: Creating 3 VMs ===
Created VM task ... for clone-vync
Created VM task ... for clone-veil
Created VM task ... for clone-web
...
✓ VM created: clone-vync (ID: ...)
✓ VM created: clone-veil (ID: ...)
✓ VM created: clone-web (ID: ...)

=== Step 2: Installing git in VMs ===
...

=== Step 3: Cloning repositories ===
Submitted clone task ... for https://github.com/techsavvyash/vync
Submitted clone task ... for https://github.com/try-veil/veil
Submitted clone task ... for https://github.com/try-veil/web
...

=== Step 4: Verifying clones ===
...

=== Step 5: Execution Results ===
VM 1 (vync) - X executions:
  ✓ git [clone https://github.com/techsavvyash/vync] (exit: 0)
  ✓ ls [-la vync] (exit: 0)
...

=== Test Completed Successfully ===
PASS
```

## Test Structure

```
tests/
├── integration/
│   ├── README.md              # This file
│   └── three_vm_clone_test.go # 3 VM clone test
└── (future test directories)
```

## Troubleshooting

### Test hangs or times out

- Check if PostgreSQL and Redis are running:
  ```bash
  docker ps | grep aetherium
  ```
- Check if worker is processing tasks (enable verbose logging)
- Verify Firecracker can start VMs manually

### VM creation fails

- Check Firecracker installation: `which firecracker`
- Check kernel exists: `ls -lh /var/firecracker/vmlinux`
- Check rootfs exists: `ls -lh /var/firecracker/rootfs.ext4`
- Check vsock module: `lsmod | grep vhost_vsock`

### Git clone fails in VM

- Verify network connectivity in VM
- Check if git is installed in rootfs
- Check execution logs in database

### Database connection errors

- Verify PostgreSQL is accessible: `psql -h localhost -U aetherium -d aetherium`
- Check migrations are applied
- Verify connection string in test

## Writing New Integration Tests

Follow this pattern:

```go
func TestMyFeature(t *testing.T) {
    // 1. Setup infrastructure connections
    store, _ := postgres.NewStore(...)
    queue, _ := asynq.NewQueue(...)
    orchestrator := firecracker.New()

    // 2. Start worker
    w := worker.New(store, orchestrator)
    w.RegisterHandlers(queue)
    queue.Start(ctx)
    defer queue.Stop()

    // 3. Create service
    taskService := service.NewTaskService(queue, store)

    // 4. Execute test scenario
    // - Create VMs
    // - Submit tasks
    // - Wait for completion
    // - Verify results

    // 5. Cleanup
    // - Delete VMs
    // - Clean up resources
}
```

## Environment Variables

Optional environment variables for test configuration:

- `POSTGRES_HOST` - PostgreSQL host (default: localhost)
- `POSTGRES_PORT` - PostgreSQL port (default: 5432)
- `REDIS_ADDR` - Redis address (default: localhost:6379)
- `FC_KERNEL` - Firecracker kernel path (default: /var/firecracker/vmlinux)
- `FC_ROOTFS` - Firecracker rootfs path (default: /var/firecracker/rootfs.ext4)
