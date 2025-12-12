# Ephemeral VM Security Architecture

## Overview

This document describes the implementation of ephemeral VMs with secure secret injection for Aetherium. The goal is to eliminate the security risk of secrets being accessible to untrusted user code while maintaining functionality.

## Current Security Issues

### Critical Vulnerabilities

1. **Secrets Written to Filesystem**
   - Location: `pkg/worker/workspace_handlers.go:401`
   - Current behavior: `echo 'export KEY="value"' >> ~/.bashrc`
   - Risk: Secrets persist on rootfs, accessible to all code

2. **No Isolation Between User Code and Secrets**
   - User code has full access to environment variables
   - Malicious code can exfiltrate secrets
   - No network egress filtering

3. **Long-Lived VMs**
   - VMs stay alive indefinitely
   - Increases attack surface
   - Secrets remain accessible long after needed

## New Architecture: Ephemeral VMs

### Key Principles

1. **Ephemeral Lifecycle**
   - VMs are short-lived
   - Auto-shutdown after:
     - Task completion
     - 30 minutes of idle time

2. **Secure Secret Injection**
   - Secrets injected at boot time via vsock
   - Stored in fc-agent memory only
   - Never written to filesystem
   - Cleared on VM shutdown

3. **Time-Limited Exposure**
   - Secrets only available during active task execution
   - Automatic cleanup
   - Reduced attack window

### Architecture Diagram

```
┌──────────────────────────────────────────────────────────────┐
│                    Worker Process (Host)                      │
│  ┌────────────────────────────────────────────────────────┐  │
│  │ WorkspaceService                                       │  │
│  │ - Decrypts secrets from PostgreSQL                    │  │
│  │ - Holds secrets in memory                             │  │
│  └──────────────┬─────────────────────────────────────────┘  │
│                 │                                             │
│                 │ (1) VM Boot + vsock connection              │
│                 ▼                                             │
│  ┌────────────────────────────────────────────────────────┐  │
│  │ Firecracker VMM                                        │  │
│  │ - Starts VM                                            │  │
│  │ - Provides vsock device                                │  │
│  └──────────────┬─────────────────────────────────────────┘  │
└─────────────────┼──────────────────────────────────────────────┘
                  │
                  │ vsock connection
                  ▼
┌──────────────────────────────────────────────────────────────┐
│                    Guest VM (Firecracker)                     │
│  ┌────────────────────────────────────────────────────────┐  │
│  │ fc-agent (On Startup)                                  │  │
│  │                                                        │  │
│  │ (2) Send GET_SECRETS request via vsock ───────────────┼──┐│
│  │                                                        │  ││
│  │ (3) Receive secrets ◄──────────────────────────────────┼──┘│
│  │     { "ANTHROPIC_API_KEY": "sk-...", ... }            │  │
│  │                                                        │  │
│  │ (4) Store in memory (map[string]string)               │  │
│  │     ❌ NOT written to ~/.bashrc                       │  │
│  │     ❌ NOT written to any file                        │  │
│  │     ✅ Only in fc-agent process memory                │  │
│  │                                                        │  │
│  │ (5) Inject into command environment on-demand         │  │
│  │     When executing user code:                         │  │
│  │     cmd.Env = append(baseEnv, secrets...)             │  │
│  │                                                        │  │
│  │ (6) Track idle time                                   │  │
│  │     - lastActivity timestamp                          │  │
│  │     - Goroutine checks every 5 minutes                │  │
│  │     - If idle > 30 min → SHUTDOWN                     │  │
│  │                                                        │  │
│  │ (7) On task completion → SHUTDOWN                     │  │
│  └────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────┘
```

## Implementation Details

### 1. fc-agent Enhancements

**File**: `cmd/fc-agent/main.go`

**New Message Types**:

```go
// Request types
const (
    RequestTypeCommand    = "execute"
    RequestTypeGetSecrets = "get_secrets"
    RequestTypeShutdown   = "shutdown"
)

// Response types
const (
    ResponseTypeSuccess = "success"
    ResponseTypeError   = "error"
)

type Request struct {
    Type    string          `json:"type"`
    Payload json.RawMessage `json:"payload,omitempty"`
}

type Response struct {
    Type    string          `json:"type"`
    Payload json.RawMessage `json:"payload,omitempty"`
    Error   string          `json:"error,omitempty"`
}

// Secrets stored in memory
type SecretStore struct {
    mu      sync.RWMutex
    secrets map[string]string
}

// Idle tracking
type IdleTracker struct {
    mu           sync.RWMutex
    lastActivity time.Time
    idleTimeout  time.Duration // 30 minutes
}
```

**Boot Sequence**:

1. Agent starts
2. Connects to host via vsock (reverse connection to port 9998)
3. Sends `GET_SECRETS` request
4. Receives secrets, stores in memory map
5. Closes connection
6. Starts listening on port 9999 for commands
7. Starts idle timeout goroutine

**Secret Injection**:

```go
func executeCommand(req *CommandRequest, secrets *SecretStore) CommandResponse {
    cmd := exec.Command(req.Cmd, req.Args...)

    // Inject secrets into command environment
    env := os.Environ() // Base environment

    secrets.mu.RLock()
    for key, value := range secrets.secrets {
        env = append(env, fmt.Sprintf("%s=%s", key, value))
    }
    secrets.mu.RUnlock()

    cmd.Env = env

    // Execute command...
}
```

**Idle Timeout**:

```go
func (t *IdleTracker) startMonitoring(shutdownFn func()) {
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()

    for range ticker.C {
        t.mu.RLock()
        elapsed := time.Since(t.lastActivity)
        t.mu.RUnlock()

        if elapsed > t.idleTimeout {
            log.Println("Idle timeout reached, shutting down VM")
            shutdownFn()
            return
        }
    }
}

func shutdownVM() {
    // Signal systemd to shut down
    cmd := exec.Command("systemctl", "poweroff")
    cmd.Run()
}
```

### 2. Worker-Side Changes

**File**: `pkg/vmm/firecracker/firecracker.go`

**New Method**: Provide secrets to VM on boot

```go
// ProvideSecretsOnBoot establishes a vsock connection and sends secrets to the VM
func (f *FirecrackerOrchestrator) ProvideSecretsOnBoot(ctx context.Context, vmID string, secrets map[string]string) error {
    // Wait for VM to connect (with timeout)
    conn, err := f.acceptSecretConnection(ctx, vmID, 30*time.Second)
    if err != nil {
        return fmt.Errorf("failed to accept secret connection: %w", err)
    }
    defer conn.Close()

    // Send secrets
    secretsPayload, _ := json.Marshal(secrets)
    response := Response{
        Type:    ResponseTypeSuccess,
        Payload: secretsPayload,
    }

    data, _ := json.Marshal(response)
    conn.Write(append(data, '\n'))

    log.Printf("Secrets provided to VM %s", vmID)
    return nil
}
```

**File**: `pkg/worker/workspace_handlers.go`

**Modified Workspace Creation**:

```go
func (w *Worker) HandleWorkspaceCreate(ctx context.Context, task *queue.Task) (*queue.TaskResult, error) {
    // ... existing VM creation ...

    // Get workspace secrets (if any)
    var secrets map[string]string
    if w.workspaceService != nil {
        workspaceSecrets, err := w.store.Secrets().ListByWorkspace(ctx, workspaceID)
        if err == nil && len(workspaceSecrets) > 0 {
            secrets = make(map[string]string)
            for _, secret := range workspaceSecrets {
                decrypted, err := w.workspaceService.GetDecryptedSecret(ctx, secret.ID)
                if err != nil {
                    log.Printf("Warning: failed to decrypt secret %s: %v", secret.Name, err)
                    continue
                }
                secrets[secret.Name] = decrypted
            }
        }
    }

    // Start VM
    if err := w.orchestrator.StartVM(ctx, vm.ID); err != nil {
        return nil, fmt.Errorf("failed to start VM: %w", err)
    }

    // Provide secrets to VM on boot (if any)
    if len(secrets) > 0 {
        if err := w.orchestrator.ProvideSecretsOnBoot(ctx, vm.ID, secrets); err != nil {
            log.Printf("Warning: failed to provide secrets to VM: %v", err)
            // Don't fail the entire workspace creation
        }
    }

    // ... rest of workspace creation ...
}
```

**Remove Old Secret Injection**:

```go
// DELETE THIS FUNCTION ENTIRELY
func (w *Worker) executeEnvVar(...) {
    // OLD CODE: exportCmd := fmt.Sprintf("echo 'export %s=\"%s\"' >> ~/.bashrc", key, value)
    // REMOVED
}
```

### 3. Idle Timeout Management

**Database Schema Addition**:

```sql
-- Add to workspaces table
ALTER TABLE workspaces ADD COLUMN last_activity_at TIMESTAMP WITH TIME ZONE DEFAULT NOW();
ALTER TABLE workspaces ADD COLUMN idle_timeout_minutes INTEGER DEFAULT 30;

CREATE INDEX idx_workspaces_last_activity ON workspaces(last_activity_at);
```

**Worker Idle Cleanup** (already exists):

```go
// This already exists in cmd/worker/main.go
func (w *Worker) StartIdleCleanup(ctx context.Context, checkInterval time.Duration) {
    // Check for idle workspaces and delete them
}
```

### 4. Task Completion Shutdown

**File**: `pkg/worker/workspace_handlers.go`

```go
// After prompt execution completes
func (w *Worker) HandlePromptExecute(ctx context.Context, task *queue.Task) (*queue.TaskResult, error) {
    // ... execute prompt ...

    // Check if this is a one-off task (not an interactive session)
    if shouldShutdownAfterTask(workspace) {
        // Signal VM to shutdown
        shutdownCmd := &vmm.Command{
            Cmd:  "systemctl",
            Args: []string{"poweroff"},
        }
        w.orchestrator.ExecuteCommand(ctx, workspace.VMID, shutdownCmd)

        // Mark workspace as stopped
        workspace.Status = "stopped"
        w.store.Workspaces().Update(ctx, workspace)
    }

    // ... return result ...
}
```

## Security Benefits

### ✅ Improvements

1. **No Filesystem Persistence**
   - Secrets never written to disk
   - Only in fc-agent process memory
   - Cleared on VM shutdown

2. **Time-Limited Exposure**
   - 30-minute idle timeout
   - Auto-shutdown on task completion
   - Reduced attack window

3. **Defense in Depth**
   - Even if user code reads env vars, secrets cleared after task
   - VM ephemeral → rootfs destroyed → no persistent access
   - Per-VM rootfs isolation (already implemented)

4. **Audit Trail**
   - Log secret access requests
   - Track VM lifecycle
   - Monitor idle timeouts

### ⚠️ Remaining Risks

1. **User Code Can Still Access Secrets During Execution**
   - Mitigation: Time-limited, ephemeral VMs
   - Future: Proxy-based secret injection (phase 2)

2. **Network Exfiltration**
   - User code can send secrets to external servers during execution
   - Mitigation: Network policies (future enhancement)

3. **Encryption Key Management**
   - Still needs Kubernetes Secret or external KMS
   - Recommendation: Implement in production

## Deployment

### Configuration

**Kubernetes ConfigMap**:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: aetherium-worker-config
data:
  idle_timeout_minutes: "30"
  shutdown_on_task_completion: "true"
```

**Kubernetes Secret**:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: aetherium-encryption-key
type: Opaque
data:
  encryption-key: <base64-encoded-32-byte-key>
```

**Worker Environment**:

```yaml
env:
  - name: WORKSPACE_ENCRYPTION_KEY
    valueFrom:
      secretKeyRef:
        name: aetherium-encryption-key
        key: encryption-key
  - name: IDLE_TIMEOUT_MINUTES
    valueFrom:
      configMapKeyRef:
        name: aetherium-worker-config
        key: idle_timeout_minutes
```

## Testing Plan

1. **Unit Tests**
   - Test secret injection mechanism
   - Test idle timeout logic
   - Test task completion shutdown

2. **Integration Tests**
   - Create workspace with secrets
   - Execute task
   - Verify secrets available during execution
   - Verify VM shuts down after task
   - Verify secrets not accessible after shutdown

3. **Security Tests**
   - Attempt to read secrets from filesystem (should fail)
   - Verify secrets cleared on shutdown
   - Test idle timeout enforcement

## Migration Path

### Phase 1: Ephemeral VMs (This Implementation)
- ✅ Secrets in memory only
- ✅ Auto-shutdown on idle/completion
- ⚠️ User code can still access secrets during execution

### Phase 2: Proxy-Based Secret Injection (Future)
- Secret proxy service
- User code never has direct access to secrets
- Proxy intercepts API calls and injects credentials
- Full isolation

### Phase 3: Network Isolation (Future)
- Egress filtering
- Allow only specific external endpoints
- Block exfiltration attempts

## Success Criteria

- ✅ Secrets never written to filesystem
- ✅ VMs auto-shutdown after 30 minutes idle
- ✅ VMs auto-shutdown after task completion
- ✅ Zero secrets persist after VM shutdown
- ✅ Backward compatible with existing workspaces
- ✅ No performance degradation

## Monitoring & Alerts

**Metrics to Track**:
- VM lifetime distribution
- Idle timeout events
- Secret access requests
- Task completion rates

**Alerts**:
- VMs running longer than expected
- Failed secret injection
- Excessive idle VMs

---

**Implementation Status**: Ready for development
**Est. Complexity**: Medium
**Est. Time**: 2-3 days
**Security Impact**: High (significant improvement)
