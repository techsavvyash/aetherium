package firecracker

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/aetherium/aetherium/pkg/vmm"
	"github.com/mdlayher/vsock"
)

type commandRequest struct {
	Cmd  string   `json:"cmd"`
	Args []string `json:"args"`
	Env  []string `json:"env,omitempty"`
}

type commandResponse struct {
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	Error    string `json:"error,omitempty"`
}

const (
	// Guest CID is what we configured in the VM
	GuestCID = 3
	// Agent port inside the VM
	AgentPort = 9999
)

// ExecuteCommand executes a command in a Firecracker VM
func (f *FirecrackerOrchestrator) ExecuteCommand(ctx context.Context, vmID string, cmd *vmm.Command) (*vmm.ExecResult, error) {
	handle, exists := f.vms[vmID]
	if !exists {
		return nil, fmt.Errorf("VM %s not found", vmID)
	}

	if handle.vm.Status != "RUNNING" {
		return nil, fmt.Errorf("VM %s is not running (status: %s)", vmID, handle.vm.Status)
	}

	// Try vsock first, then fall back to network
	return f.executeCommand(ctx, handle, cmd)
}

// executeCommand tries vsock first, falls back to network if needed
func (f *FirecrackerOrchestrator) executeCommand(ctx context.Context, handle *vmHandle, cmd *vmm.Command) (*vmm.ExecResult, error) {
	// Try vsock first
	conn, err := f.connectViaVsock(ctx, 5*time.Second)
	if err != nil {
		// Vsock failed, try network (the agent will be on TCP if vsock unavailable in guest)
		// For now, return error as we haven't configured network yet
		return &vmm.ExecResult{
			ExitCode: 1,
			Stdout:   "",
			Stderr: fmt.Sprintf(`Cannot connect to VM agent: %v

The agent is running on TCP inside the VM, but no network is configured.

Options to fix this:
1. Use a kernel with vsock support in the guest
2. Configure network interface for the VM (requires TAP device setup)
3. Use Docker orchestrator instead (has built-in networking)

For now, you can verify the VM is working by seeing the boot logs.
The agent should be running on TCP port 9999 inside the VM.
`, err),
		}, nil
	}
	defer conn.Close()

	return f.sendCommandAndWait(ctx, conn, cmd)
}

func (f *FirecrackerOrchestrator) connectViaVsock(ctx context.Context, timeout time.Duration) (net.Conn, error) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		conn, err := vsock.Dial(GuestCID, AgentPort, nil)
		if err == nil {
			return conn, nil
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(500 * time.Millisecond):
			// Retry
		}
	}

	return nil, fmt.Errorf("vsock connection timeout (guest CID: %d, port: %d)", GuestCID, AgentPort)
}

func (f *FirecrackerOrchestrator) sendCommandAndWait(ctx context.Context, conn net.Conn, cmd *vmm.Command) (*vmm.ExecResult, error) {
	// Prepare command request
	req := commandRequest{
		Cmd:  cmd.Cmd,
		Args: cmd.Args,
	}

	// Convert env map to []string if needed
	if cmd.Env != nil {
		for k, v := range cmd.Env {
			req.Env = append(req.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	// Send request
	reqData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	_, err = conn.Write(append(reqData, '\n'))
	if err != nil {
		return nil, fmt.Errorf("failed to send command: %w", err)
	}

	// Read response with timeout
	respCh := make(chan *commandResponse, 1)
	errCh := make(chan error, 1)

	go func() {
		reader := bufio.NewReader(conn)
		line, err := reader.ReadString('\n')
		if err != nil {
			errCh <- err
			return
		}

		var resp commandResponse
		if err := json.Unmarshal([]byte(line), &resp); err != nil {
			errCh <- err
			return
		}

		respCh <- &resp
	}()

	select {
	case resp := <-respCh:
		if resp.Error != "" {
			return &vmm.ExecResult{
				ExitCode: resp.ExitCode,
				Stdout:   resp.Stdout,
				Stderr:   resp.Stderr + "\nAgent error: " + resp.Error,
			}, nil
		}
		return &vmm.ExecResult{
			ExitCode: resp.ExitCode,
			Stdout:   resp.Stdout,
			Stderr:   resp.Stderr,
		}, nil
	case err := <-errCh:
		return nil, fmt.Errorf("failed to read response: %w", err)
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("command execution timeout")
	}
}
