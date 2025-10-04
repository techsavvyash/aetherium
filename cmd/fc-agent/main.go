package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/mdlayher/vsock"
)

const (
	// Vsock port for agent communication
	AgentPort = 9999
)

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

func main() {
	log.SetOutput(os.Stderr)
	log.Println("Firecracker Agent starting...")

	// Try vsock first, fall back to TCP if vsock not available
	listener, transport, err := createListener(AgentPort)
	if err != nil {
		log.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	log.Printf("Agent listening on %s port %d", transport, AgentPort)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Accept error: %v", err)
			continue
		}

		go handleConnection(conn)
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

func handleConnection(conn net.Conn) {
	defer conn.Close()

	log.Printf("New connection from %s", conn.RemoteAddr())

	reader := bufio.NewReader(conn)

	for {
		// Read command request (newline delimited JSON)
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				log.Printf("Read error: %v", err)
			}
			return
		}

		// Parse request
		var req CommandRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			sendError(conn, fmt.Sprintf("Invalid JSON: %v", err))
			continue
		}

		// Execute command
		resp := executeCommand(&req)

		// Send response
		data, _ := json.Marshal(resp)
		conn.Write(append(data, '\n'))
	}
}

func executeCommand(req *CommandRequest) CommandResponse {
	log.Printf("Executing: %s %v", req.Cmd, req.Args)

	cmd := exec.Command(req.Cmd, req.Args...)

	// Set environment if provided
	if len(req.Env) > 0 {
		cmd.Env = append(os.Environ(), req.Env...)
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
