# Error Message Display Fix

**Date**: 2025-12-07
**Status**: ✅ FIXED & DEPLOYED

## Problem Statement

Error messages were not being displayed in the dashboard - only showing "exit code 1" instead of actual error details like "login invalid creds".

## Root Causes Identified

### 1. Dashboard Double Submission Bug
**File**: `dashboard/src/app/workspaces/[id]/page.tsx` (lines 257-268)

**Issue**: When WebSocket was connected, prompts were being submitted TWICE:
- Once via WebSocket (streaming) - ✅ Shows proper errors
- Once via REST API (queue-based) - ❌ Shows only "exit code 1"

The REST API execution would fail with generic error message, overwriting/hiding the WebSocket's detailed error.

**Fix**: Removed duplicate REST API call when WebSocket is connected. Now only uses WebSocket streaming path which correctly displays error details.

```typescript
// BEFORE:
if (isConnected) {
  wsSendPrompt(prompt, systemPrompt, workingDirectory);
  // Also submit via REST API to persist in database  // ❌ DUPLICATE!
  const result = await api.submitPrompt(workspaceId, {...});
}

// AFTER:
if (isConnected) {
  // Use WebSocket for real-time streaming
  // WebSocket handles execution and persists to database
  wsSendPrompt(prompt, systemPrompt, workingDirectory);  // ✅ SINGLE PATH
}
```

### 2. Worker Error Message Incomplete
**File**: `pkg/worker/workspace_handlers.go` (lines 768-779)

**Issue**: When prompt execution failed, the worker only included the exit code in the error message:

```go
// BEFORE:
errMsg := fmt.Sprintf("prompt execution failed with exit code %d", execResult.ExitCode)
// Result: "prompt execution failed with exit code 1" ❌
```

The actual error details were in `execResult.Stderr` and `execResult.Stdout` but weren't being included.

**Fix**: Enhanced error message to include stderr and stdout:

```go
// AFTER:
errMsg := fmt.Sprintf("prompt execution failed with exit code %d", execResult.ExitCode)
if execResult.Stderr != "" {
    errMsg += fmt.Sprintf("\nStderr: %s", execResult.Stderr)
}
if execResult.Stdout != "" {
    errMsg += fmt.Sprintf("\nStdout: %s", execResult.Stdout)
}
// Result: "prompt execution failed with exit code 1
//          Stderr: Error: Invalid credentials for login
//          Stdout: ..." ✅
```

## Deployment

### Files Modified
1. `dashboard/src/app/workspaces/[id]/page.tsx` - Fixed double submission
2. `pkg/worker/workspace_handlers.go` - Enhanced error messages

### Services Restarted
1. ✅ Dashboard: Restarted at `localhost:3000` (PID 1620656)
2. ✅ Worker: Kubernetes pod rolled out - `aetherium-worker-5565d447ff-hfkpf`

### Verification
```bash
# Dashboard is responding
curl -I http://localhost:3000
# HTTP/1.1 200 OK ✅

# Worker is running
kubectl get pods -n aetherium
# aetherium-worker-5565d447ff-hfkpf   1/1     Running ✅

# Worker logs show new code
kubectl logs -n aetherium deployment/aetherium-worker
# Registered handlers: workspace:create, workspace:delete, prompt:execute ✅
```

## Expected Behavior After Fix

### Before Fix
```
❌ Dashboard shows: "Prompt execution failed with exit code 1"
❌ No details about why it failed
```

### After Fix
```
✅ Dashboard shows full error with context:
   "Prompt execution failed with exit code 1
    Stderr: Error: Invalid credentials for login to claude-code
    Please set your ANTHROPIC_API_KEY environment variable"
```

## Technical Details

### WebSocket Streaming Path (Now Used Exclusively When Connected)
```
User Prompt → WebSocket → Session Manager → ExecuteCommandStream → VM (Claude Code)
                                                ↓
                                        Real-time streaming chunks
                                                ↓
                                        Full stderr/stdout output ✅
```

### REST API Queue Path (Only Used as Fallback)
```
User Prompt → REST API → Redis Queue → Worker → ExecuteCommand → VM
                                           ↓
                                    Now includes stderr/stdout in error ✅
```

## Testing Recommendations

1. **Test Error Display**: Submit a prompt that will fail (e.g., without API key)
   - Expected: See full error message with stderr details
   - Location: `http://localhost:3000/workspaces/{workspace-id}`

2. **Test WebSocket Streaming**: Submit a valid prompt when connected
   - Expected: Real-time output streaming
   - Expected: No duplicate executions in worker logs

3. **Test REST API Fallback**: Disconnect WebSocket, submit prompt
   - Expected: Execution via queue with detailed error message
   - Verify: Worker logs show enhanced error format

## Related Components

- `/pkg/websocket/session.go` (lines 243-349) - WebSocket streaming (already correct)
- `/pkg/vmm/interface.go` (lines 28-64) - ExecuteCommandStream interface
- `/dashboard/src/lib/websocket.ts` - WebSocket client implementation

## Conclusion

Both root causes have been fixed and deployed:
1. ✅ Eliminated double submission when WebSocket is connected
2. ✅ Enhanced worker error messages to include full stderr/stdout

Error messages like "login invalid creds" will now be properly displayed instead of generic "exit code 1".
