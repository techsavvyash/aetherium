package firecracker

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/aetherium/aetherium/pkg/vmm"
	fcvsock "github.com/firecracker-microvm/firecracker-go-sdk/vsock"
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
	// Try vsock first with longer timeout to allow agent to start
	conn, err := f.connectViaVsock(ctx, handle, 15*time.Second)
	if err != nil {
		// Vsock failed - check if it's a kernel support issue
		return &vmm.ExecResult{
			ExitCode: 1,
			Stdout:   "",
			Stderr: fmt.Sprintf(`Cannot connect to VM agent via vsock: %v

Diagnosis:
- Host vsock support: ✓ (vhost-vsock module loaded)
- Guest vsock support: ✗ (likely missing vsock_guest kernel module)

The guest kernel (vmlinux.bin) may not have vsock support compiled in.

Solutions:
1. Use a kernel with CONFIG_VIRTIO_VSOCK=y
   Download: Run: sudo ./scripts/download-vsock-kernel.sh
   (Downloads kernel v6.1.141 with vsock support)

2. Configure network (TAP device) and use TCP:
   - Create TAP device
   - Add network interface to VM config
   - Agent will fall back to TCP port 9999

3. Use Docker orchestrator (has built-in networking):
   ./bin/docker-demo

Current status: VM is running but agent cannot be reached via vsock.
`, err),
		}, nil
	}
	defer conn.Close()

	return f.sendCommandAndWait(ctx, conn, cmd)
}

func (f *FirecrackerOrchestrator) connectViaVsock(ctx context.Context, handle *vmHandle, timeout time.Duration) (net.Conn, error) {
	// Firecracker vsock uses a UNIX socket on the host with a special protocol
	// The socket path is: <vm-socket-path>.vsock
	vsockPath := handle.vm.Config.SocketPath + ".vsock"

	// Use Firecracker's vsock dialer which handles the CONNECT/OK handshake
	conn, err := fcvsock.DialContext(ctx, vsockPath, uint32(AgentPort),
		fcvsock.WithRetryTimeout(timeout),
		fcvsock.WithRetryInterval(500*time.Millisecond),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to vsock at %s port %d: %w", vsockPath, AgentPort, err)
	}

	return conn, nil
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
