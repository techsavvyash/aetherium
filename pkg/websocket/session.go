package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/aetherium/aetherium/pkg/queue"
	"github.com/aetherium/aetherium/pkg/storage"
	"github.com/aetherium/aetherium/pkg/vmm"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
}

// SessionManager handles WebSocket sessions for AI workspaces
type SessionManager struct {
	store        storage.Store
	orchestrator vmm.VMOrchestrator
	sessions     map[uuid.UUID]*Session
	mu           sync.RWMutex
}

// NewSessionManager creates a new session manager
func NewSessionManager(store storage.Store, orchestrator vmm.VMOrchestrator) *SessionManager {
	return &SessionManager{
		store:        store,
		orchestrator: orchestrator,
		sessions:     make(map[uuid.UUID]*Session),
	}
}

// Session represents an active WebSocket session
type Session struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	Conn        *websocket.Conn
	Manager     *SessionManager
	send        chan []byte
	done        chan struct{}
	mu          sync.Mutex
}

// Message types for WebSocket communication
type MessageType string

const (
	MessageTypePrompt   MessageType = "prompt"
	MessageTypeResponse MessageType = "response"
	MessageTypeError    MessageType = "error"
	MessageTypeStatus   MessageType = "status"
	MessageTypePing     MessageType = "ping"
	MessageTypePong     MessageType = "pong"
)

// IncomingMessage represents a message from the client
type IncomingMessage struct {
	Type             MessageType            `json:"type"`
	Prompt           string                 `json:"prompt,omitempty"`
	SystemPrompt     string                 `json:"system_prompt,omitempty"`
	WorkingDirectory string                 `json:"working_directory,omitempty"`
	Environment      map[string]interface{} `json:"environment,omitempty"`
}

// OutgoingMessage represents a message to the client
type OutgoingMessage struct {
	Type      MessageType `json:"type"`
	SessionID uuid.UUID   `json:"session_id,omitempty"`
	MessageID uuid.UUID   `json:"message_id,omitempty"`
	Content   string      `json:"content,omitempty"`
	ExitCode  *int        `json:"exit_code,omitempty"`
	Error     string      `json:"error,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// HandleSession handles a WebSocket connection for a workspace session
func (m *SessionManager) HandleSession(w http.ResponseWriter, r *http.Request, workspaceID uuid.UUID) {
	// Verify workspace exists and is ready
	workspace, err := m.store.Workspaces().Get(r.Context(), workspaceID)
	if err != nil {
		http.Error(w, "Workspace not found", http.StatusNotFound)
		return
	}

	if workspace.Status != "ready" {
		http.Error(w, fmt.Sprintf("Workspace is not ready (status: %s)", workspace.Status), http.StatusBadRequest)
		return
	}

	// Upgrade to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade WebSocket: %v", err)
		return
	}

	// Create session record
	sessionID := uuid.New()
	now := time.Now()
	clientIP := r.RemoteAddr

	dbSession := &storage.WorkspaceSession{
		ID:           sessionID,
		WorkspaceID:  workspaceID,
		Status:       "connected",
		ClientIP:     &clientIP,
		ConnectedAt:  now,
		LastActivity: now,
	}

	if err := m.store.Sessions().Create(r.Context(), dbSession); err != nil {
		log.Printf("Failed to create session record: %v", err)
		conn.Close()
		return
	}

	// Create session object
	session := &Session{
		ID:          sessionID,
		WorkspaceID: workspaceID,
		Conn:        conn,
		Manager:     m,
		send:        make(chan []byte, 256),
		done:        make(chan struct{}),
	}

	// Register session
	m.mu.Lock()
	m.sessions[sessionID] = session
	m.mu.Unlock()

	// Send welcome message
	session.sendMessage(&OutgoingMessage{
		Type:      MessageTypeStatus,
		SessionID: sessionID,
		Content:   fmt.Sprintf("Connected to workspace: %s", workspace.Name),
		Timestamp: time.Now(),
	})

	// Start goroutines for reading and writing
	go session.writePump()
	go session.readPump(workspace)
}

// readPump reads messages from the WebSocket connection
func (s *Session) readPump(workspace *storage.Workspace) {
	defer func() {
		s.cleanup()
	}()

	s.Conn.SetReadLimit(64 * 1024) // 64KB max message size
	s.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	s.Conn.SetPongHandler(func(string) error {
		s.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := s.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Parse incoming message
		var incoming IncomingMessage
		if err := json.Unmarshal(message, &incoming); err != nil {
			s.sendError("Invalid message format")
			continue
		}

		// Handle message based on type
		switch incoming.Type {
		case MessageTypePing:
			s.sendMessage(&OutgoingMessage{
				Type:      MessageTypePong,
				Timestamp: time.Now(),
			})
		case MessageTypePrompt:
			s.handlePrompt(workspace, &incoming)
		default:
			s.sendError(fmt.Sprintf("Unknown message type: %s", incoming.Type))
		}

		// Update last activity
		s.updateActivity()
	}
}

// writePump writes messages to the WebSocket connection
func (s *Session) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		s.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-s.send:
			s.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				s.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := s.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			s.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := s.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-s.done:
			return
		}
	}
}

// handlePrompt processes a prompt from the client
func (s *Session) handlePrompt(workspace *storage.Workspace, incoming *IncomingMessage) {
	ctx := context.Background()
	messageID := uuid.New()

	// Store the prompt message
	promptMsg := &storage.SessionMessage{
		ID:          uuid.New(),
		SessionID:   s.ID,
		MessageType: "user_prompt",
		Content:     incoming.Prompt,
		CreatedAt:   time.Now(),
	}
	if err := s.Manager.store.SessionMessages().Create(ctx, promptMsg); err != nil {
		log.Printf("Failed to store prompt message: %v", err)
	}

	// Send acknowledgment
	s.sendMessage(&OutgoingMessage{
		Type:      MessageTypeStatus,
		MessageID: messageID,
		Content:   "Processing prompt...",
		Timestamp: time.Now(),
	})

	// Determine working directory
	workingDir := workspace.WorkingDirectory
	if incoming.WorkingDirectory != "" {
		workingDir = incoming.WorkingDirectory
	}

	// Build AI command based on workspace configuration
	var aiCmd string
	escapedPrompt := escapeShellArg(incoming.Prompt)

	switch workspace.AIAssistant {
	case "claude-code":
		aiCmd = fmt.Sprintf("cd %s && claude-code --dangerously-skip-permissions '%s'", workingDir, escapedPrompt)
	case "ampcode", "amp":
		aiCmd = fmt.Sprintf("cd %s && amp '%s'", workingDir, escapedPrompt)
	default:
		s.sendError(fmt.Sprintf("Unknown AI assistant: %s", workspace.AIAssistant))
		return
	}

	// Execute command in VM
	vmID := ""
	if workspace.VMID != nil {
		vmID = workspace.VMID.String()
	} else {
		s.sendError("Workspace has no VM assigned")
		return
	}

	cmd := &vmm.Command{
		Cmd:  "bash",
		Args: []string{"-c", aiCmd},
	}

	result, err := s.Manager.orchestrator.ExecuteCommand(ctx, vmID, cmd)

	var exitCode *int
	var stdout, stderr string

	if err != nil {
		s.sendMessage(&OutgoingMessage{
			Type:      MessageTypeError,
			MessageID: messageID,
			Error:     fmt.Sprintf("Failed to execute command: %v", err),
			Timestamp: time.Now(),
		})
	} else {
		exitCode = &result.ExitCode
		stdout = result.Stdout
		stderr = result.Stderr

		// Send response
		content := stdout
		if stderr != "" && result.ExitCode != 0 {
			content += "\n\nStderr:\n" + stderr
		}

		s.sendMessage(&OutgoingMessage{
			Type:      MessageTypeResponse,
			MessageID: messageID,
			Content:   content,
			ExitCode:  exitCode,
			Timestamp: time.Now(),
		})
	}

	// Store the response message
	responseContent := stdout
	if stderr != "" {
		responseContent += "\n" + stderr
	}
	if err != nil {
		responseContent = err.Error()
	}

	responseMsg := &storage.SessionMessage{
		ID:          uuid.New(),
		SessionID:   s.ID,
		MessageType: "ai_response",
		Content:     responseContent,
		ExitCode:    exitCode,
		CreatedAt:   time.Now(),
	}
	if err := s.Manager.store.SessionMessages().Create(ctx, responseMsg); err != nil {
		log.Printf("Failed to store response message: %v", err)
	}
}

// sendMessage sends a message to the client
func (s *Session) sendMessage(msg *OutgoingMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal message: %v", err)
		return
	}

	select {
	case s.send <- data:
	default:
		log.Printf("Session %s send buffer full", s.ID)
	}
}

// sendError sends an error message to the client
func (s *Session) sendError(errMsg string) {
	s.sendMessage(&OutgoingMessage{
		Type:      MessageTypeError,
		Error:     errMsg,
		Timestamp: time.Now(),
	})
}

// updateActivity updates the session's last activity timestamp
func (s *Session) updateActivity() {
	ctx := context.Background()
	s.Manager.store.Sessions().UpdateLastActivity(ctx, s.ID)
}

// cleanup cleans up the session
func (s *Session) cleanup() {
	close(s.done)
	s.Conn.Close()

	// Update session status in database
	ctx := context.Background()
	s.Manager.store.Sessions().UpdateStatus(ctx, s.ID, "disconnected")

	// Remove from active sessions
	s.Manager.mu.Lock()
	delete(s.Manager.sessions, s.ID)
	s.Manager.mu.Unlock()

	log.Printf("Session %s disconnected", s.ID)
}

// GetSession returns a session by ID
func (m *SessionManager) GetSession(sessionID uuid.UUID) *Session {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sessions[sessionID]
}

// CloseSession closes a session
func (m *SessionManager) CloseSession(sessionID uuid.UUID) {
	m.mu.RLock()
	session, ok := m.sessions[sessionID]
	m.mu.RUnlock()

	if ok {
		session.cleanup()
	}
}

// escapeShellArg escapes a string for safe use in shell commands
func escapeShellArg(s string) string {
	result := ""
	for _, c := range s {
		if c == '\'' {
			result += "'\\''"
		} else {
			result += string(c)
		}
	}
	return result
}

// Unused import placeholder for queue (will be used for task-based session management)
var _ = queue.TaskTypePromptExecute
