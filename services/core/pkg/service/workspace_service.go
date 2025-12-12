package service

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"time"

	"github.com/aetherium/aetherium/pkg/api"
	"github.com/aetherium/aetherium/pkg/queue"
	"github.com/aetherium/aetherium/pkg/storage"
	"github.com/google/uuid"
)

// WorkspaceService handles workspace operations
type WorkspaceService struct {
	queue         queue.Queue
	store         storage.Store
	encryptionKey []byte // AES-256 key for secret encryption
}

// NewWorkspaceService creates a new workspace service
func NewWorkspaceService(q queue.Queue, s storage.Store, encryptionKeyHex string) (*WorkspaceService, error) {
	var encryptionKey []byte
	if encryptionKeyHex != "" {
		var err error
		encryptionKey, err = hex.DecodeString(encryptionKeyHex)
		if err != nil {
			return nil, fmt.Errorf("invalid encryption key: %w", err)
		}
		if len(encryptionKey) != 32 {
			return nil, fmt.Errorf("encryption key must be 32 bytes (64 hex characters)")
		}
	} else {
		// Generate a random key if not provided (useful for development)
		encryptionKey = make([]byte, 32)
		if _, err := rand.Read(encryptionKey); err != nil {
			return nil, fmt.Errorf("failed to generate encryption key: %w", err)
		}
	}

	return &WorkspaceService{
		queue:         q,
		store:         s,
		encryptionKey: encryptionKey,
	}, nil
}

// CreateWorkspace submits a workspace creation task
func (s *WorkspaceService) CreateWorkspace(ctx context.Context, req *api.CreateWorkspaceRequest) (taskID, workspaceID uuid.UUID, err error) {
	// Create workspace record in pending state
	workspaceID = uuid.New()
	workspace := &storage.Workspace{
		ID:                workspaceID,
		Name:              req.Name,
		Description:       stringPtr(req.Description),
		Status:            "creating",
		AIAssistant:       req.AIAssistant,
		AIAssistantConfig: req.AIAssistantConfig,
		WorkingDirectory:  req.WorkingDirectory,
	}

	// Handle environment_id if provided
	if req.EnvironmentID != "" {
		envID, err := uuid.Parse(req.EnvironmentID)
		if err != nil {
			return uuid.Nil, uuid.Nil, fmt.Errorf("invalid environment_id: %w", err)
		}
		workspace.EnvironmentID = &envID
	}

	if workspace.WorkingDirectory == "" {
		workspace.WorkingDirectory = "/workspace"
	}

	if err := s.store.Workspaces().Create(ctx, workspace); err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("failed to create workspace: %w", err)
	}

	// Store secrets (encrypted)
	for _, secretReq := range req.Secrets {
		if _, err := s.addSecret(ctx, workspaceID, &secretReq, "workspace"); err != nil {
			// Cleanup on failure
			s.store.Workspaces().Delete(ctx, workspaceID)
			return uuid.Nil, uuid.Nil, fmt.Errorf("failed to store secret %s: %w", secretReq.Name, err)
		}
	}

	// Store prep steps
	prepSteps := make([]*storage.PrepStep, len(req.PrepSteps))
	for i, stepReq := range req.PrepSteps {
		prepSteps[i] = &storage.PrepStep{
			ID:          uuid.New(),
			WorkspaceID: workspaceID,
			StepType:    stepReq.Type,
			StepOrder:   stepReq.Order,
			Config:      stepReq.Config,
			Status:      "pending",
		}
	}

	if len(prepSteps) > 0 {
		if err := s.store.PrepSteps().CreateBatch(ctx, prepSteps); err != nil {
			s.store.Workspaces().Delete(ctx, workspaceID)
			return uuid.Nil, uuid.Nil, fmt.Errorf("failed to store prep steps: %w", err)
		}
	}

	// Build task payload
	payload := map[string]interface{}{
		"workspace_id": workspaceID.String(),
		"name":         req.Name,
		"vcpus":        req.VCPUs,
		"memory_mb":    req.MemoryMB,
		"ai_assistant": req.AIAssistant,
		"working_dir":  workspace.WorkingDirectory,
	}

	if len(req.AdditionalTools) > 0 {
		payload["additional_tools"] = req.AdditionalTools
	}
	if len(req.ToolVersions) > 0 {
		payload["tool_versions"] = req.ToolVersions
	}
	if req.AIAssistantConfig != nil {
		payload["ai_assistant_config"] = req.AIAssistantConfig
	}

	task := &queue.Task{
		ID:      uuid.New(),
		Type:    queue.TaskTypeWorkspaceCreate,
		Payload: payload,
	}

	if err := s.queue.Enqueue(ctx, task, &queue.TaskOptions{
		MaxRetry: 2,
		Timeout:  30 * time.Minute, // Long timeout for VM + tools + prep steps
		Queue:    "default",
		Priority: 5,
	}); err != nil {
		s.store.Workspaces().Delete(ctx, workspaceID)
		return uuid.Nil, uuid.Nil, fmt.Errorf("failed to enqueue workspace creation task: %w", err)
	}

	return task.ID, workspaceID, nil
}

// DeleteWorkspace submits a workspace deletion task
func (s *WorkspaceService) DeleteWorkspace(ctx context.Context, workspaceID uuid.UUID) (uuid.UUID, error) {
	// Verify workspace exists
	workspace, err := s.store.Workspaces().Get(ctx, workspaceID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("workspace not found: %w", err)
	}

	task := &queue.Task{
		ID:   uuid.New(),
		Type: queue.TaskTypeWorkspaceDelete,
		Payload: map[string]interface{}{
			"workspace_id": workspaceID.String(),
			"vm_id":        workspace.VMID,
		},
	}

	if err := s.queue.Enqueue(ctx, task, &queue.TaskOptions{
		MaxRetry: 2,
		Timeout:  5 * time.Minute,
		Queue:    "default",
		Priority: 5,
	}); err != nil {
		return uuid.Nil, fmt.Errorf("failed to enqueue workspace deletion task: %w", err)
	}

	return task.ID, nil
}

// GetWorkspace retrieves workspace information
func (s *WorkspaceService) GetWorkspace(ctx context.Context, workspaceID uuid.UUID) (*storage.Workspace, error) {
	return s.store.Workspaces().Get(ctx, workspaceID)
}

// GetWorkspaceByName retrieves workspace by name
func (s *WorkspaceService) GetWorkspaceByName(ctx context.Context, name string) (*storage.Workspace, error) {
	return s.store.Workspaces().GetByName(ctx, name)
}

// ListWorkspaces lists all workspaces
func (s *WorkspaceService) ListWorkspaces(ctx context.Context) ([]*storage.Workspace, error) {
	return s.store.Workspaces().List(ctx, map[string]interface{}{})
}

// GetPrepSteps retrieves prep steps for a workspace
func (s *WorkspaceService) GetPrepSteps(ctx context.Context, workspaceID uuid.UUID) ([]*storage.PrepStep, error) {
	return s.store.PrepSteps().ListByWorkspace(ctx, workspaceID)
}

// SubmitPrompt creates a prompt task and optionally enqueues it for execution
func (s *WorkspaceService) SubmitPrompt(ctx context.Context, workspaceID uuid.UUID, req *api.SubmitPromptRequest) (uuid.UUID, error) {
	// Verify workspace exists and is ready
	workspace, err := s.store.Workspaces().Get(ctx, workspaceID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("workspace not found: %w", err)
	}

	if workspace.Status != "ready" {
		return uuid.Nil, fmt.Errorf("workspace is not ready (status: %s)", workspace.Status)
	}

	// Set defaults
	priority := req.Priority
	if priority == 0 {
		priority = 5
	}

	workingDir := req.WorkingDirectory
	if workingDir == "" {
		workingDir = workspace.WorkingDirectory
	}

	// Create prompt task record
	promptID := uuid.New()
	now := time.Now()
	promptTask := &storage.PromptTask{
		ID:               promptID,
		WorkspaceID:      workspaceID,
		Prompt:           req.Prompt,
		SystemPrompt:     stringPtr(req.SystemPrompt),
		WorkingDirectory: stringPtr(workingDir),
		Environment:      req.Environment,
		Priority:         priority,
		Status:           "pending",
		CreatedAt:        now,
		ScheduledAt:      now,
	}

	if err := s.store.PromptTasks().Create(ctx, promptTask); err != nil {
		return uuid.Nil, fmt.Errorf("failed to create prompt task: %w", err)
	}

	// Enqueue for execution
	task := &queue.Task{
		ID:   uuid.New(),
		Type: queue.TaskTypePromptExecute,
		Payload: map[string]interface{}{
			"prompt_id":    promptID.String(),
			"workspace_id": workspaceID.String(),
		},
		Priority: priority,
	}

	if err := s.queue.Enqueue(ctx, task, &queue.TaskOptions{
		MaxRetry: 1, // Prompts are idempotent, don't retry
		Timeout:  30 * time.Minute,
		Queue:    "default",
		Priority: priority,
	}); err != nil {
		// Mark prompt as failed if enqueue fails
		s.store.PromptTasks().UpdateStatus(ctx, promptID, "failed", &storage.PromptResult{
			Error: fmt.Sprintf("failed to enqueue: %v", err),
		})
		return uuid.Nil, fmt.Errorf("failed to enqueue prompt execution: %w", err)
	}

	return promptID, nil
}

// GetPrompt retrieves a prompt task
func (s *WorkspaceService) GetPrompt(ctx context.Context, promptID uuid.UUID) (*storage.PromptTask, error) {
	return s.store.PromptTasks().Get(ctx, promptID)
}

// ListPrompts lists prompts for a workspace
func (s *WorkspaceService) ListPrompts(ctx context.Context, workspaceID uuid.UUID) ([]*storage.PromptTask, error) {
	return s.store.PromptTasks().ListByWorkspace(ctx, workspaceID, 100) // Default limit of 100
}

// CancelPrompt cancels a pending prompt
func (s *WorkspaceService) CancelPrompt(ctx context.Context, promptID uuid.UUID) error {
	return s.store.PromptTasks().Cancel(ctx, promptID)
}

// AddSecret adds a secret to a workspace
func (s *WorkspaceService) AddSecret(ctx context.Context, workspaceID uuid.UUID, req *api.AddSecretRequest) (uuid.UUID, error) {
	scope := req.Scope
	if scope == "" {
		scope = "workspace"
	}

	secretReq := &api.SecretRequest{
		Name:        req.Name,
		Value:       req.Value,
		Type:        req.Type,
		Description: req.Description,
	}

	return s.addSecret(ctx, workspaceID, secretReq, scope)
}

// addSecret is an internal method to add an encrypted secret
func (s *WorkspaceService) addSecret(ctx context.Context, workspaceID uuid.UUID, req *api.SecretRequest, scope string) (uuid.UUID, error) {
	// Encrypt the secret value
	encryptedValue, nonce, err := s.encryptSecret([]byte(req.Value))
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to encrypt secret: %w", err)
	}

	secretType := req.Type
	if secretType == "" {
		secretType = "api_key"
	}

	secret := &storage.WorkspaceSecret{
		ID:              uuid.New(),
		WorkspaceID:     &workspaceID,
		Name:            req.Name,
		Description:     stringPtr(req.Description),
		SecretType:      secretType,
		EncryptedValue:  encryptedValue,
		EncryptionKeyID: "default",
		Nonce:           nonce,
		Scope:           scope,
	}

	if err := s.store.Secrets().Create(ctx, secret); err != nil {
		return uuid.Nil, fmt.Errorf("failed to store secret: %w", err)
	}

	return secret.ID, nil
}

// ListSecrets lists secrets for a workspace (names only, no values)
func (s *WorkspaceService) ListSecrets(ctx context.Context, workspaceID uuid.UUID) ([]*storage.WorkspaceSecret, error) {
	return s.store.Secrets().ListByWorkspace(ctx, workspaceID)
}

// DeleteSecret deletes a secret
func (s *WorkspaceService) DeleteSecret(ctx context.Context, secretID uuid.UUID) error {
	return s.store.Secrets().Delete(ctx, secretID)
}

// GetDecryptedSecret retrieves and decrypts a secret (for internal use by worker)
func (s *WorkspaceService) GetDecryptedSecret(ctx context.Context, secretID uuid.UUID) (string, error) {
	secret, err := s.store.Secrets().Get(ctx, secretID)
	if err != nil {
		return "", fmt.Errorf("secret not found: %w", err)
	}

	decrypted, err := s.decryptSecret(secret.EncryptedValue, secret.Nonce)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt secret: %w", err)
	}

	return string(decrypted), nil
}

// GetDecryptedSecretByName retrieves and decrypts a secret by name (for internal use)
func (s *WorkspaceService) GetDecryptedSecretByName(ctx context.Context, workspaceID uuid.UUID, name string) (string, error) {
	secret, err := s.store.Secrets().GetByName(ctx, &workspaceID, name)
	if err != nil {
		return "", fmt.Errorf("secret not found: %w", err)
	}

	decrypted, err := s.decryptSecret(secret.EncryptedValue, secret.Nonce)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt secret: %w", err)
	}

	return string(decrypted), nil
}

// encryptSecret encrypts a secret value using AES-256-GCM
func (s *WorkspaceService) encryptSecret(plaintext []byte) (ciphertext, nonce []byte, err error) {
	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return nil, nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}

	nonce = make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}

	ciphertext = gcm.Seal(nil, nonce, plaintext, nil)
	return ciphertext, nonce, nil
}

// decryptSecret decrypts a secret value using AES-256-GCM
func (s *WorkspaceService) decryptSecret(ciphertext, nonce []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// Session management

// CreateSession creates a new interactive session
func (s *WorkspaceService) CreateSession(ctx context.Context, workspaceID uuid.UUID, clientIP, userAgent string) (uuid.UUID, error) {
	// Verify workspace exists and is ready
	workspace, err := s.store.Workspaces().Get(ctx, workspaceID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("workspace not found: %w", err)
	}

	if workspace.Status != "ready" {
		return uuid.Nil, fmt.Errorf("workspace is not ready (status: %s)", workspace.Status)
	}

	session := &storage.WorkspaceSession{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		Status:      "active",
		ClientIP:    stringPtr(clientIP),
		UserAgent:   stringPtr(userAgent),
	}

	if err := s.store.Sessions().Create(ctx, session); err != nil {
		return uuid.Nil, fmt.Errorf("failed to create session: %w", err)
	}

	return session.ID, nil
}

// GetSession retrieves a session
func (s *WorkspaceService) GetSession(ctx context.Context, sessionID uuid.UUID) (*storage.WorkspaceSession, error) {
	return s.store.Sessions().Get(ctx, sessionID)
}

// EndSession terminates a session
func (s *WorkspaceService) EndSession(ctx context.Context, sessionID uuid.UUID) error {
	return s.store.Sessions().UpdateStatus(ctx, sessionID, "terminated")
}

// UpdateSessionActivity updates the last activity timestamp
func (s *WorkspaceService) UpdateSessionActivity(ctx context.Context, sessionID uuid.UUID) error {
	return s.store.Sessions().UpdateLastActivity(ctx, sessionID)
}

// AddSessionMessage adds a message to a session
func (s *WorkspaceService) AddSessionMessage(ctx context.Context, sessionID uuid.UUID, messageType, content string, exitCode *int) (uuid.UUID, error) {
	msg := &storage.SessionMessage{
		ID:          uuid.New(),
		SessionID:   sessionID,
		MessageType: messageType,
		Content:     content,
		ExitCode:    exitCode,
	}

	if err := s.store.SessionMessages().Create(ctx, msg); err != nil {
		return uuid.Nil, fmt.Errorf("failed to add session message: %w", err)
	}

	return msg.ID, nil
}

// GetSessionMessages retrieves messages for a session
func (s *WorkspaceService) GetSessionMessages(ctx context.Context, sessionID uuid.UUID) ([]*storage.SessionMessage, error) {
	return s.store.SessionMessages().ListBySession(ctx, sessionID, 1000) // Default limit of 1000
}

// Helper function for string pointers
func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
