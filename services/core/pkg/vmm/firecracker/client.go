package firecracker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"
)

// FirecrackerClient is an HTTP client for the Firecracker API over Unix socket
type FirecrackerClient struct {
	socketPath string
	httpClient *http.Client
}

// NewFirecrackerClient creates a new Firecracker API client
func NewFirecrackerClient(socketPath string) *FirecrackerClient {
	return &FirecrackerClient{
		socketPath: socketPath,
		httpClient: &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					return net.Dial("unix", socketPath)
				},
			},
			Timeout: 10 * time.Second,
		},
	}
}

// WaitForSocket waits for the Firecracker socket to be created
func (c *FirecrackerClient) WaitForSocket(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(c.socketPath); err == nil {
			// Socket exists, try to connect
			time.Sleep(100 * time.Millisecond) // Give it a moment to be ready
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for socket %s", c.socketPath)
}

// PutBootSource configures the boot source (kernel)
func (c *FirecrackerClient) PutBootSource(kernelPath string, bootArgs *string) error {
	body := BootSource{
		KernelImagePath: kernelPath,
		BootArgs:        bootArgs,
	}

	return c.makeRequest("PUT", "/boot-source", body)
}

// PutDrive configures a block device
func (c *FirecrackerClient) PutDrive(driveID, pathOnHost string, isRoot, isReadOnly bool) error {
	body := Drive{
		DriveID:      driveID,
		PathOnHost:   pathOnHost,
		IsRootDevice: isRoot,
		IsReadOnly:   isReadOnly,
	}

	return c.makeRequest("PUT", fmt.Sprintf("/drives/%s", driveID), body)
}

// PutMachineConfig configures CPU and memory
func (c *FirecrackerClient) PutMachineConfig(vcpuCount, memSizeMB int) error {
	body := MachineConfiguration{
		VcpuCount:  vcpuCount,
		MemSizeMib: memSizeMB,
	}

	return c.makeRequest("PUT", "/machine-config", body)
}

// StartInstance starts the VM
func (c *FirecrackerClient) StartInstance() error {
	body := InstanceActionInfo{
		ActionType: ActionInstanceStart,
	}

	return c.makeRequest("PUT", "/actions", body)
}

// SendCtrlAltDel sends a shutdown signal to the guest
func (c *FirecrackerClient) SendCtrlAltDel() error {
	body := InstanceActionInfo{
		ActionType: ActionSendCtrlAltDel,
	}

	return c.makeRequest("PUT", "/actions", body)
}

// GetInstanceInfo retrieves VM state information
func (c *FirecrackerClient) GetInstanceInfo() (*InstanceInfo, error) {
	req, err := http.NewRequest("GET", "http://localhost/", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var info InstanceInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &info, nil
}

// makeRequest is a helper for making JSON requests to the Firecracker API
func (c *FirecrackerClient) makeRequest(method, path string, body interface{}) error {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequest(method, "http://localhost"+path, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request to %s: %w", path, err)
	}
	defer resp.Body.Close()

	// Firecracker returns 204 No Content on success for most operations
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d for %s %s: %s", resp.StatusCode, method, path, string(bodyBytes))
	}

	return nil
}
