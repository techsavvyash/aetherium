# TUI Streaming Fix Plan

**Issue:** Terminal output from Claude Code running in Firecracker VMs only shows the final message instead of streaming the full TUI in real-time.

**Created:** 2025-12-07  
**Status:** ✅ IMPLEMENTED  
**Priority:** High  
**Estimated Effort:** 1-2 days (L-sized task)

## Implementation Status

All tasks have been completed and the streaming functionality is now working:

- ✅ fc-agent: Added PTY-based streaming executor with `execute_stream` request type
- ✅ VMOrchestrator: Added `ExecuteCommandStream()` interface method
- ✅ Firecracker: Implemented streaming vsock client with `sendCommandAndStream()`
- ✅ Docker: Added stub/fallback streaming implementation  
- ✅ WebSocket: Updated `handlePrompt()` to use streaming with fallback
- ✅ Frontend: Verified `terminal-view.tsx` handles incremental updates correctly

The streaming now works end-to-end:
1. Prompts trigger `execute_stream` in fc-agent
2. PTY output is streamed as JSON chunks over vsock
3. WebSocket forwards chunks to the dashboard
4. Terminal view displays output incrementally

---

## Executive Summary

The current architecture uses a **blocking/synchronous execution path** where commands run to completion and output is returned as a single response. To show Claude Code's TUI in real-time, we need to implement a **streaming execution path** using PTY (pseudo-terminal) and incremental WebSocket messages.

### Root Cause Analysis

| Layer | Current Behavior | Problem |
|-------|-----------------|---------|
| **fc-agent** | `exec.Command()` with `cmd.Stdout = &stdout` buffer | Waits for command completion, returns all output at once |
| **firecracker/exec.go** | `sendCommandAndWait()` reads single JSON response | Blocks until command finishes |
| **websocket/session.go** | `ExecuteCommand()` called synchronously | Single response message with complete output |
| **terminal-view.tsx** | Supports incremental updates | Never receives incremental data |

### Solution Overview

Add a **parallel streaming path** (alongside existing sync path for backward compatibility):

```
Client → WebSocket (prompt)
       → Session.handlePrompt()
       → VMOrchestrator.ExecuteCommandStream() [NEW]
       → Firecracker.sendCommandAndStream() [NEW]
       → vsock → fc-agent "execute_stream" request [NEW]
       → fc-agent runs command under PTY, emits JSON stream events [NEW]
       → Back up through WebSocket as incremental messages
       → Frontend receives updates in real-time
```

---

## Implementation Tasks

### Task 1: Add PTY Dependency to fc-agent

**Description:** Add the `creack/pty` library for PTY support inside the VM.

**Files to Modify:**
- `go.mod`

**Changes:**
```bash
go get github.com/creack/pty
```

**Verification:**
- [ ] Run `go mod tidy` successfully
- [ ] Verify no import conflicts

---

### Task 2: Extend fc-agent Protocol Types

**Description:** Add new request/response types for streaming.

**File:** `cmd/fc-agent/main.go`

**Changes:**
1. Add new request type:
   ```go
   RequestTypeCommandStream = "execute_stream"
   ```

2. Add new response types:
   ```go
   ResponseTypeStreamData = "stream_data"
   ResponseTypeStreamExit = "stream_exit"
   ```

3. Add payload structs:
   ```go
   type StreamDataPayload struct {
       Stdout string `json:"stdout,omitempty"`
       Stderr string `json:"stderr,omitempty"`
   }
   
   type StreamExitPayload struct {
       ExitCode int    `json:"exit_code"`
       Error    string `json:"error,omitempty"`
   }
   ```

**Verification:**
- [ ] Code compiles without errors
- [ ] New types are exported/accessible

---

### Task 3: Implement PTY-based Streaming Executor in fc-agent

**Description:** Add `executeCommandStream()` function that runs commands under PTY and streams output.

**File:** `cmd/fc-agent/main.go`

**Changes:**
1. Add import: `"github.com/creack/pty"`

2. Create helper function `buildCommand()` to extract env/secrets logic

3. Implement `executeCommandStream()`:
   - Use `pty.Start(cmd)` to run command in PTY
   - Read from PTY in 4KB chunks
   - Send each chunk as `stream_data` JSON response
   - Send `stream_exit` when command completes
   - Update idle tracker on each chunk

4. Wire new request type in `handleRequest()` switch statement

**Key Code Pattern (from GoTTY/ttyd):**
```go
ptmx, err := pty.Start(cmd)
buf := make([]byte, 4096)
for {
    n, readErr := ptmx.Read(buf)
    if n > 0 {
        // Send chunk immediately
        sendResponse(conn, ResponseTypeStreamData, payload, "")
    }
    if readErr != nil {
        break
    }
}
```

**Verification:**
- [ ] Can compile fc-agent with PTY support
- [ ] Manual test: `echo "test"` streams correctly
- [ ] Manual test: Long-running command like `top` streams updates

---

### Task 4: Add VMOrchestrator Streaming Interface

**Description:** Extend VMOrchestrator interface with streaming method.

**File:** `pkg/vmm/interface.go`

**Changes:**
1. Add new types:
   ```go
   type ExecStreamChunk struct {
       Stdout   string
       Stderr   string
       ExitCode *int   // nil until final chunk
       Error    string
   }
   
   type StreamHandler func(chunk *ExecStreamChunk)
   ```

2. Extend interface:
   ```go
   type VMOrchestrator interface {
       // ... existing methods
       ExecuteCommandStream(ctx context.Context, vmID string, cmd *Command, handler StreamHandler) error
   }
   ```

**Verification:**
- [ ] Interface compiles
- [ ] All orchestrator implementations updated (compile check)

---

### Task 5: Implement Firecracker Streaming Executor

**Description:** Add streaming vsock client in Firecracker orchestrator.

**File:** `pkg/vmm/firecracker/exec.go`

**Changes:**
1. Add agent request/response structs for streaming protocol

2. Implement `ExecuteCommandStream()`:
   - Connect via vsock/TCP (same as existing)
   - Call `sendCommandAndStream()`

3. Implement `sendCommandAndStream()`:
   - Send `execute_stream` request
   - Read JSON lines in loop
   - Parse `stream_data` → call handler with stdout/stderr
   - Parse `stream_exit` → call handler with exit code, return
   - Handle `error` response → fall back to sync path

**Key Consideration:** No fixed 30s timeout for streaming path; use context cancellation.

**Verification:**
- [ ] Can connect to updated fc-agent
- [ ] Receives stream_data events
- [ ] Receives stream_exit event
- [ ] Falls back gracefully on old fc-agent

---

### Task 6: Implement Docker Streaming (Stub/Fallback)

**Description:** Add streaming support to Docker orchestrator (can be a wrapper).

**File:** `pkg/vmm/docker/docker.go`

**Changes:**
- Implement `ExecuteCommandStream()` as a wrapper around `ExecuteCommand()` that emits one big chunk

**Verification:**
- [ ] Docker orchestrator compiles
- [ ] Fallback behavior works correctly

---

### Task 7: Update WebSocket Session Handler

**Description:** Use streaming path in `handlePrompt()`.

**File:** `pkg/websocket/session.go`

**Changes:**
1. Import `strings` (if not already)

2. In `handlePrompt()`:
   - Create `streamHandler` callback that:
     - Accumulates stdout/stderr in `strings.Builder`
     - Sends `MessageTypeResponse` with cumulative content after each chunk
   - Call `ExecuteCommandStream()` first
   - If error (e.g., old agent), fall back to `ExecuteCommand()`
   - Send final message with exit code

3. Store final response in database (once at end)

**Key Pattern:**
```go
streamHandler := func(chunk *vmm.ExecStreamChunk) {
    fullStdout.WriteString(chunk.Stdout)
    s.sendMessage(&OutgoingMessage{
        Type:      MessageTypeResponse,
        MessageID: messageID,
        Content:   fullStdout.String(),  // Cumulative
        Timestamp: time.Now(),
    })
}
```

**Verification:**
- [ ] WebSocket messages stream incrementally
- [ ] Falls back to sync on error
- [ ] Database stores final result correctly

---

### Task 8: Verify Frontend Compatibility

**Description:** Ensure `terminal-view.tsx` handles streaming correctly.

**File:** `dashboard/src/components/terminal-view.tsx`

**Analysis:** The current component already supports incremental updates:
```typescript
if (output.startsWith(lastOutputRef.current)) {
    const newContent = output.slice(lastOutputRef.current.length);
    terminalRef.current.write(newContent);
}
```

**Verification:**
- [ ] Multiple rapid updates render correctly
- [ ] ANSI escape codes preserved
- [ ] TUI elements (colors, cursor positioning) work
- [ ] No flickering or redraws

---

## Testing Guide

### Unit Tests

#### Test 1: fc-agent Streaming Protocol
```bash
# In VM or with TCP fallback
echo '{"type":"execute_stream","payload":{"cmd":"echo","args":["hello"]}}' | nc localhost 9999
```
**Expected:** Multiple JSON lines: `stream_data` with "hello\n", then `stream_exit` with exit_code 0.

#### Test 2: PTY Behavior
```bash
# Test ANSI sequences preserved
echo '{"type":"execute_stream","payload":{"cmd":"bash","args":["-c","echo -e \"\\033[31mred\\033[0m\""]}}' | nc localhost 9999
```
**Expected:** `stream_data` contains ANSI escape codes for red text.

#### Test 3: Long-running Command
```bash
# Test incremental streaming
echo '{"type":"execute_stream","payload":{"cmd":"bash","args":["-c","for i in 1 2 3; do echo $i; sleep 1; done"]}}' | nc localhost 9999
```
**Expected:** Three `stream_data` messages ~1 second apart, then `stream_exit`.

### Integration Tests

#### Test 4: Firecracker Orchestrator Streaming
```go
func TestExecuteCommandStream(t *testing.T) {
    // Create VM
    // Call ExecuteCommandStream with handler that collects chunks
    // Verify chunks received incrementally
    // Verify exit code in final chunk
}
```

#### Test 5: WebSocket Streaming
```javascript
// Connect to WebSocket
// Send prompt message
// Verify multiple response messages received
// Verify content grows incrementally
```

### End-to-End Test

#### Test 6: Dashboard TUI Streaming

**Prerequisites:**
- API Gateway running
- Worker running  
- VM created and ready
- Workspace connected to VM

**Steps:**
1. Open dashboard at `http://localhost:3000`
2. Navigate to Workspaces → Select workspace
3. Submit a prompt: `"List files in the current directory and then wait 5 seconds"`
4. Observe terminal output

**Expected:**
- Output appears incrementally as command runs
- Claude Code TUI elements visible (if using Claude)
- Colors and formatting preserved
- No waiting for complete output before display

#### Test 7: Fallback Behavior

**Steps:**
1. Use an old fc-agent image without streaming support
2. Submit a prompt

**Expected:**
- Falls back to synchronous execution
- Output still appears (just all at once)
- No errors displayed to user

#### Test 8: Long Claude Session

**Steps:**
1. Submit a complex prompt that triggers multi-step Claude Code execution
2. Monitor terminal for 2+ minutes

**Expected:**
- All intermediate output visible
- TUI redraws correctly
- No idle timeout during active session
- Final result displayed

### Performance Tests

#### Test 9: Bandwidth Measurement
- Measure bytes sent over WebSocket for typical Claude session
- Compare cumulative-buffer approach vs delta-only approach
- **Acceptance:** < 5MB for 10-minute session with typical TUI output

#### Test 10: Latency Measurement  
- Measure time from PTY read to browser display
- **Acceptance:** < 100ms end-to-end latency

---

## Rollback Plan

If issues arise after deployment:

1. **Quick Rollback:** Remove `execute_stream` handling from `handleRequest()` in fc-agent
   - System falls back to sync path automatically

2. **Feature Flag:** Add environment variable `STREAMING_ENABLED=false` to disable streaming
   - Check in `handlePrompt()` before calling `ExecuteCommandStream()`

3. **Version Detection:** fc-agent version check before using streaming
   - Old agents return error on unknown request type → automatic fallback

---

## Future Enhancements

Once basic streaming works, consider:

1. **Interactive Terminal:** 
   - Add `terminal_input` message type for keystrokes
   - Add `terminal_resize` for PTY resizing
   - Enables full interactive Claude Code sessions

2. **Delta-Only Protocol:**
   - Send only new bytes instead of cumulative buffer
   - Reduces bandwidth for long sessions

3. **Connection Multiplexing:**
   - Add `command_id` to support multiple concurrent commands per vsock connection

4. **Session Recording:**
   - Store raw PTY output for playback/debugging

---

## Files Changed Summary

| File | Change Type | Description |
|------|-------------|-------------|
| `go.mod` | Add dependency | `creack/pty` |
| `cmd/fc-agent/main.go` | Major | Add streaming executor, new protocol types |
| `pkg/vmm/interface.go` | Add interface | `ExecuteCommandStream()` method |
| `pkg/vmm/firecracker/exec.go` | Add method | Streaming vsock client |
| `pkg/vmm/docker/docker.go` | Add method | Stub/fallback implementation |
| `pkg/websocket/session.go` | Modify | Use streaming in `handlePrompt()` |
| `dashboard/src/components/terminal-view.tsx` | Verify only | No changes needed (already supports incremental) |

---

## References

- [GoTTY source](https://github.com/yudai/gotty) - PTY streaming pattern
- [ttyd source](https://github.com/tsl0922/ttyd) - C-level async PTY handling
- [tuistory source](https://github.com/remorses/tuistory) - Persistent terminal state pattern
- [xterm.js AttachAddon](https://github.com/xtermjs/xterm.js/tree/master/addons/addon-attach) - Frontend WebSocket attachment
- [creack/pty](https://github.com/creack/pty) - Go PTY library
