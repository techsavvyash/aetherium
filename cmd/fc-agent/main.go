package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/mdlayher/vsock"
)

const (
	// Vsock port for agent communication
	AgentPort = 9999
	// Vsock port for secret fetching (reverse connection to host)
	SecretPort = 9998
	// Host CID (always 2 for Firecracker)
	HostCID = 2
	// Idle timeout duration
	IdleTimeout = 30 * time.Minute
)

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

// Generic request/response structures
type Request struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type Response struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
	Error   string          `json:"error,omitempty"`
}

// Legacy structures for backward compatibility
type CommandRequest struct {
	Cmd  string   `json:"cmd"`
	Args []string `json:"args"`
	Env  []string `json:"env,omitempty"`
}

type CommandResponse struct {
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	Error    string `json:"error,omitempty"`
}

// SecretStore stores secrets in memory only (never persisted to filesystem)
type SecretStore struct {
	mu      sync.RWMutex
	secrets map[string]string
}

func NewSecretStore() *SecretStore {
	return &SecretStore{
		secrets: make(map[string]string),
	}
}

func (s *SecretStore) Set(secrets map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.secrets = secrets
	log.Printf("Stored %d secrets in memory", len(secrets))
}

func (s *SecretStore) GetAll() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return copy to prevent external modification
	copy := make(map[string]string, len(s.secrets))
	for k, v := range s.secrets {
		copy[k] = v
	}
	return copy
}

func (s *SecretStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.secrets = make(map[string]string)
	log.Println("Cleared all secrets from memory")
}

// IdleTracker tracks last activity time and triggers shutdown on idle timeout
type IdleTracker struct {
	mu           sync.RWMutex
	lastActivity time.Time
	idleTimeout  time.Duration
}

func NewIdleTracker(timeout time.Duration) *IdleTracker {
	return &IdleTracker{
		lastActivity: time.Now(),
		idleTimeout:  timeout,
	}
}

func (t *IdleTracker) UpdateActivity() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.lastActivity = time.Now()
}

func (t *IdleTracker) GetIdleDuration() time.Duration {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return time.Since(t.lastActivity)
}

func (t *IdleTracker) StartMonitoring(ctx context.Context, shutdownFn func()) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	log.Printf("Starting idle monitoring (timeout: %v, check interval: 5m)", t.idleTimeout)

	for {
		select {
		case <-ctx.Done():
			log.Println("Idle monitoring stopped")
			return
		case <-ticker.C:
			idleDuration := t.GetIdleDuration()
			log.Printf("Idle check: last activity %v ago", idleDuration)

			if idleDuration > t.idleTimeout {
				log.Printf("Idle timeout reached (%v > %v), shutting down VM", idleDuration, t.idleTimeout)
				shutdownFn()
				return
			}
		}
	}
}

func main() {
	log.SetOutput(os.Stderr)
	log.Println("Firecracker Agent starting...")

	// Initialize secret store
	secretStore := NewSecretStore()

	// Initialize idle tracker
	idleTracker := NewIdleTracker(IdleTimeout)

	// Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Step 1: Fetch secrets from host via vsock (if available)
	if err := fetchSecretsFromHost(secretStore); err != nil {
		log.Printf("Warning: Failed to fetch secrets from host: %v", err)
		log.Println("Continuing without secrets (may be normal for non-workspace VMs)")
	}

	// Step 2: Start idle timeout monitoring
	go idleTracker.StartMonitoring(ctx, shutdownVM)

	// Step 3: Start listening for commands
	listener, transport, err := createListener(AgentPort)
	if err != nil {
		log.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	log.Printf("Agent listening on %s port %d", transport, AgentPort)
	log.Println("✓ Agent ready to process commands")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Accept error: %v", err)
			continue
		}

		go handleConnection(conn, secretStore, idleTracker)
	}
}

// fetchSecretsFromHost attempts to connect to host and retrieve secrets
func fetchSecretsFromHost(store *SecretStore) error {
	log.Println("Fetching secrets from host via vsock...")

	// Try to dial host via vsock
	conn, err := vsock.Dial(HostCID, SecretPort, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to host: %w", err)
	}
	defer conn.Close()

	// Send GET_SECRETS request
	request := Request{
		Type: RequestTypeGetSecrets,
	}

	data, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	if _, err := conn.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	// Read response
	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	var response Response
	if err := json.Unmarshal([]byte(line), &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if response.Type == ResponseTypeError {
		return fmt.Errorf("host returned error: %s", response.Error)
	}

	// Parse secrets from payload
	var secrets map[string]string
	if err := json.Unmarshal(response.Payload, &secrets); err != nil {
		return fmt.Errorf("failed to parse secrets: %w", err)
	}

	// Store secrets in memory
	store.Set(secrets)
	log.Printf("✓ Successfully fetched and stored %d secrets in memory", len(secrets))

	return nil
}

// shutdownVM triggers VM shutdown
func shutdownVM() {
	log.Println("Initiating VM shutdown...")

	cmd := exec.Command("systemctl", "poweroff")
	if err := cmd.Run(); err != nil {
		log.Printf("Failed to shutdown VM: %v", err)
		// Fallback to immediate shutdown
		syscall.Sync()
		syscall.Reboot(syscall.LINUX_REBOOT_CMD_POWER_OFF)
	}
}

func createListener(port uint32) (net.Listener, string, error) {
	// First, try to create a vsock listener
	vsockListener, err := vsock.Listen(port, nil)
	if err == nil {
		return vsockListener, "vsock", nil
	}

	log.Printf("Vsock not available (%v), falling back to TCP", err)

	// Fall back to TCP on all interfaces
	addr := fmt.Sprintf(":%d", port)
	tcpListener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, "", fmt.Errorf("both vsock and TCP failed: %w", err)
	}

	return tcpListener, "TCP", nil
}

func handleConnection(conn net.Conn, secretStore *SecretStore, idleTracker *IdleTracker) {
	defer conn.Close()

	log.Printf("New connection from %s", conn.RemoteAddr())

	reader := bufio.NewReader(conn)

	for {
		// Read request (newline delimited JSON)
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				log.Printf("Read error: %v", err)
			}
			return
		}

		// Update activity timestamp
		idleTracker.UpdateActivity()

		// Try to parse as new Request format first
		var req Request
		if err := json.Unmarshal([]byte(line), &req); err == nil && req.Type != "" {
			// New format - handle based on type
			handleRequest(conn, &req, secretStore, idleTracker)
			continue
		}

		// Fall back to legacy CommandRequest format for backward compatibility
		var legacyReq CommandRequest
		if err := json.Unmarshal([]byte(line), &legacyReq); err != nil {
			sendError(conn, fmt.Sprintf("Invalid JSON: %v", err))
			continue
		}

		// Execute legacy command with secrets injected
		resp := executeCommandWithSecrets(&legacyReq, secretStore)

		// Send legacy response
		data, _ := json.Marshal(resp)
		conn.Write(append(data, '\n'))
	}
}

// handleRequest processes new-format requests
func handleRequest(conn net.Conn, req *Request, secretStore *SecretStore, idleTracker *IdleTracker) {
	switch req.Type {
	case RequestTypeCommand:
		// Parse command from payload
		var cmdReq CommandRequest
		if err := json.Unmarshal(req.Payload, &cmdReq); err != nil {
			sendResponse(conn, ResponseTypeError, nil, fmt.Sprintf("Invalid command payload: %v", err))
			return
		}

		// Execute command with secrets
		cmdResp := executeCommandWithSecrets(&cmdReq, secretStore)

		// Wrap in Response
		payload, _ := json.Marshal(cmdResp)
		sendResponse(conn, ResponseTypeSuccess, payload, "")

	case RequestTypeShutdown:
		log.Println("Received shutdown request")
		sendResponse(conn, ResponseTypeSuccess, nil, "")

		// Shutdown VM in background
		go func() {
			time.Sleep(1 * time.Second)
			shutdownVM()
		}()

	default:
		sendResponse(conn, ResponseTypeError, nil, fmt.Sprintf("Unknown request type: %s", req.Type))
	}
}

// sendResponse sends a Response to the connection
func sendResponse(conn net.Conn, respType string, payload json.RawMessage, errMsg string) {
	resp := Response{
		Type:    respType,
		Payload: payload,
		Error:   errMsg,
	}
	data, _ := json.Marshal(resp)
	conn.Write(append(data, '\n'))
}

// executeCommandWithSecrets executes a command with secrets injected from memory
func executeCommandWithSecrets(req *CommandRequest, secretStore *SecretStore) CommandResponse {
	log.Printf("Executing: %s %v", req.Cmd, req.Args)

	cmd := exec.Command(req.Cmd, req.Args...)

	// Build environment: base + request-specific + secrets from memory
	env := os.Environ()

	// Add request-specific environment variables if provided
	if len(req.Env) > 0 {
		env = append(env, req.Env...)
	}

	// ✅ SECURITY: Inject secrets from in-memory store (never from filesystem)
	secrets := secretStore.GetAll()
	for key, value := range secrets {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	cmd.Env = env

	if len(secrets) > 0 {
		log.Printf("Injected %d secrets into command environment from memory", len(secrets))
	}

	// Capture output
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute
	err := cmd.Run()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				exitCode = status.ExitStatus()
			} else {
				exitCode = 1
			}
		} else {
			return CommandResponse{
				ExitCode: 1,
				Stdout:   stdout.String(),
				Stderr:   stderr.String(),
				Error:    fmt.Sprintf("Failed to execute: %v", err),
			}
		}
	}

	return CommandResponse{
		ExitCode: exitCode,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
	}
}

func sendError(conn net.Conn, errMsg string) {
	resp := CommandResponse{
		ExitCode: 1,
		Error:    errMsg,
	}
	data, _ := json.Marshal(resp)
	conn.Write(append(data, '\n'))
}
