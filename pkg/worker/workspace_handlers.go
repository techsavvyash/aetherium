package worker

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aetherium/aetherium/pkg/mcp"
	"github.com/aetherium/aetherium/pkg/queue"
	"github.com/aetherium/aetherium/pkg/service"
	"github.com/aetherium/aetherium/pkg/storage"
	"github.com/aetherium/aetherium/pkg/tools"
	"github.com/aetherium/aetherium/pkg/types"
	"github.com/aetherium/aetherium/pkg/vmm"
	"github.com/google/uuid"
)

// WorkspaceCreatePayload represents workspace creation task payload
type WorkspaceCreatePayload struct {
	WorkspaceID       string                 `json:"workspace_id"`
	Name              string                 `json:"name"`
	VCPUs             int                    `json:"vcpus"`
	MemoryMB          int                    `json:"memory_mb"`
	AIAssistant       string                 `json:"ai_assistant"`
	AIAssistantConfig map[string]interface{} `json:"ai_assistant_config,omitempty"`
	WorkingDir        string                 `json:"working_dir"`
	AdditionalTools   []string               `json:"additional_tools,omitempty"`
	ToolVersions      map[string]string      `json:"tool_versions,omitempty"`
}

// WorkspaceDeletePayload represents workspace deletion task payload
type WorkspaceDeletePayload struct {
	WorkspaceID string     `json:"workspace_id"`
	VMID        *uuid.UUID `json:"vm_id,omitempty"`
}

// PromptExecutePayload represents prompt execution task payload
type PromptExecutePayload struct {
	PromptID    string `json:"prompt_id"`
	WorkspaceID string `json:"workspace_id"`
}

// SetWorkspaceService sets the workspace service for workspace handlers
func (w *Worker) SetWorkspaceService(ws *service.WorkspaceService) {
	w.workspaceService = ws
}

// RegisterWorkspaceHandlers registers workspace-related task handlers
func (w *Worker) RegisterWorkspaceHandlers(q queue.Queue) error {
	if err := q.RegisterHandler(queue.TaskTypeWorkspaceCreate, w.HandleWorkspaceCreate); err != nil {
		return fmt.Errorf("failed to register workspace create handler: %w", err)
	}

	if err := q.RegisterHandler(queue.TaskTypeWorkspaceDelete, w.HandleWorkspaceDelete); err != nil {
		return fmt.Errorf("failed to register workspace delete handler: %w", err)
	}

	if err := q.RegisterHandler(queue.TaskTypePromptExecute, w.HandlePromptExecute); err != nil {
		return fmt.Errorf("failed to register prompt execute handler: %w", err)
	}

	return nil
}

// HandleWorkspaceCreate handles workspace creation tasks
func (w *Worker) HandleWorkspaceCreate(ctx context.Context, task *queue.Task) (*queue.TaskResult, error) {
	startTime := time.Now()

	var payload WorkspaceCreatePayload
	if err := queue.UnmarshalPayload(task.Payload, &payload); err != nil {
		return nil, fmt.Errorf("invalid payload: %w", err)
	}

	workspaceID, err := uuid.Parse(payload.WorkspaceID)
	if err != nil {
		return nil, fmt.Errorf("invalid workspace_id: %w", err)
	}

	log.Printf("Creating workspace: %s (ai=%s, vcpu=%d, mem=%dMB)", payload.Name, payload.AIAssistant, payload.VCPUs, payload.MemoryMB)

	// Update workspace status to preparing
	if err := w.store.Workspaces().UpdateStatus(ctx, workspaceID, "preparing"); err != nil {
		log.Printf("Warning: Failed to update workspace status: %v", err)
	}

	// Create VM config
	vmID := uuid.New().String()
	vmConfig := &types.VMConfig{
		ID:         vmID,
		KernelPath: "/var/firecracker/vmlinux",
		RootFSPath: "", // Will be set by orchestrator.CreateVM() from template
		SocketPath: fmt.Sprintf("/tmp/aetherium-vm-%s.sock", vmID),
		VCPUCount:  payload.VCPUs,
		MemoryMB:   payload.MemoryMB,
	}

	// Create VM using orchestrator
	vm, err := w.orchestrator.CreateVM(ctx, vmConfig)
	if err != nil {
		w.store.Workspaces().UpdateStatus(ctx, workspaceID, "failed")
		return &queue.TaskResult{
			TaskID:    task.ID,
			Success:   false,
			Error:     fmt.Sprintf("failed to create VM: %v", err),
			Duration:  time.Since(startTime),
			StartedAt: startTime,
		}, nil
	}

	// Start VM
	if err := w.orchestrator.StartVM(ctx, vm.ID); err != nil {
		w.store.Workspaces().UpdateStatus(ctx, workspaceID, "failed")
		return &queue.TaskResult{
			TaskID:    task.ID,
			Success:   false,
			Error:     fmt.Sprintf("failed to start VM: %v", err),
			Duration:  time.Since(startTime),
			StartedAt: startTime,
		}, nil
	}

	// Wait for agent to be ready
	time.Sleep(5 * time.Second)

	// Install tools (default + AI assistant + additional)
	log.Printf("Installing tools in workspace VM %s...", vm.ID)

	defaultTools := tools.GetDefaultTools()
	allTools := append(defaultTools, payload.AdditionalTools...)

	// Add the AI assistant tool if not already included
	aiTool := payload.AIAssistant
	if aiTool == "claude-code" || aiTool == "ampcode" {
		hasAITool := false
		for _, t := range allTools {
			if t == aiTool {
				hasAITool = true
				break
			}
		}
		if !hasAITool {
			allTools = append(allTools, aiTool)
		}
	}

	// Remove duplicates
	toolSet := make(map[string]bool)
	uniqueTools := []string{}
	for _, tool := range allTools {
		if !toolSet[tool] {
			toolSet[tool] = true
			uniqueTools = append(uniqueTools, tool)
		}
	}

	// Install tools with timeout
	toolVersions := payload.ToolVersions
	if toolVersions == nil {
		toolVersions = make(map[string]string)
	}

	if err := w.toolInstaller.InstallToolsWithTimeout(ctx, vm.ID, uniqueTools, toolVersions, 20*time.Minute); err != nil {
		log.Printf("Warning: Tool installation failed (workspace may be partially usable): %v", err)
	} else {
		log.Printf("✓ All tools installed successfully in workspace VM %s", vm.ID)
	}

	// Link VM to workspace
	vmUUID, _ := uuid.Parse(vm.ID)
	if err := w.store.Workspaces().SetVMID(ctx, workspaceID, vmUUID); err != nil {
		log.Printf("Warning: Failed to link VM to workspace: %v", err)
	}

	// Execute prep steps
	log.Printf("Executing preparation steps for workspace %s...", workspaceID)
	if err := w.executePrepSteps(ctx, workspaceID, vm.ID); err != nil {
		log.Printf("Warning: Prep steps failed: %v", err)
		// Don't fail the entire workspace creation, just log it
	}

	// Track VM resources
	w.mu.Lock()
	w.runningVMs[vm.ID] = &vmResourceUsage{
		VCPUs:    payload.VCPUs,
		MemoryMB: int64(payload.MemoryMB),
	}
	w.tasksProcessed++
	w.mu.Unlock()

	// Store VM in database
	kernelPath := vmConfig.KernelPath
	rootfsPath := vmConfig.RootFSPath
	socketPath := vmConfig.SocketPath

	var workerID *string
	if w.workerInfo != nil {
		workerID = &w.workerInfo.ID
	}

	dbVM := &storage.VM{
		ID:           vmUUID,
		Name:         fmt.Sprintf("workspace-%s", payload.Name),
		Orchestrator: "firecracker",
		Status:       string(vm.Status),
		KernelPath:   &kernelPath,
		RootFSPath:   &rootfsPath,
		SocketPath:   &socketPath,
		VCPUCount:    &payload.VCPUs,
		MemoryMB:     &payload.MemoryMB,
		WorkerID:     workerID,
		CreatedAt:    time.Now(),
		Metadata: map[string]interface{}{
			"workspace_id": workspaceID.String(),
		},
	}

	if err := w.store.VMs().Create(ctx, dbVM); err != nil {
		log.Printf("Warning: Failed to store VM in database: %v", err)
	}

	// Mark workspace as ready
	if err := w.store.Workspaces().SetReady(ctx, workspaceID); err != nil {
		log.Printf("Warning: Failed to mark workspace as ready: %v", err)
	}

	// Update worker resources in database
	if w.workerInfo != nil {
		w.updateWorkerResources(ctx)
	}

	log.Printf("✓ Workspace created successfully: %s (vm=%s)", payload.Name, vm.ID)

	result := map[string]interface{}{
		"workspace_id": workspaceID.String(),
		"vm_id":        vm.ID,
		"name":         payload.Name,
		"status":       "ready",
	}

	return &queue.TaskResult{
		TaskID:    task.ID,
		Success:   true,
		Result:    result,
		Duration:  time.Since(startTime),
		StartedAt: startTime,
	}, nil
}

// executePrepSteps executes preparation steps for a workspace
func (w *Worker) executePrepSteps(ctx context.Context, workspaceID uuid.UUID, vmID string) error {
	prepSteps, err := w.store.PrepSteps().ListByWorkspace(ctx, workspaceID)
	if err != nil {
		return fmt.Errorf("failed to get prep steps: %w", err)
	}

	for _, step := range prepSteps {
		log.Printf("Executing prep step %d (%s) for workspace %s", step.StepOrder, step.StepType, workspaceID)

		// Update step status to running
		w.store.PrepSteps().UpdateStatus(ctx, step.ID, "running", nil)
		stepStartTime := time.Now()

		var result *storage.PrepStepResult
		var execErr error

		switch step.StepType {
		case "git_clone":
			result, execErr = w.executeGitClone(ctx, vmID, step.Config)
		case "script":
			result, execErr = w.executeScript(ctx, vmID, step.Config)
		case "env_var":
			result, execErr = w.executeEnvVar(ctx, vmID, workspaceID, step.Config)
		default:
			execErr = fmt.Errorf("unknown step type: %s", step.StepType)
		}

		if result == nil {
			result = &storage.PrepStepResult{}
		}

		durationMS := int(time.Since(stepStartTime).Milliseconds())
		result.DurationMS = durationMS

		if execErr != nil {
			result.Error = execErr.Error()
			w.store.PrepSteps().UpdateStatus(ctx, step.ID, "failed", result)
			log.Printf("✗ Prep step %d failed: %v", step.StepOrder, execErr)
			// Continue to next step instead of failing entirely
			continue
		}

		w.store.PrepSteps().UpdateStatus(ctx, step.ID, "completed", result)
		log.Printf("✓ Prep step %d completed successfully", step.StepOrder)
	}

	return nil
}

// executeGitClone executes a git clone prep step
func (w *Worker) executeGitClone(ctx context.Context, vmID string, config map[string]interface{}) (*storage.PrepStepResult, error) {
	url, _ := config["url"].(string)
	branch, _ := config["branch"].(string)
	destPath, _ := config["dest_path"].(string)

	if url == "" {
		return nil, fmt.Errorf("git clone requires 'url' in config")
	}

	if destPath == "" {
		destPath = "/workspace"
	}

	// Build git clone command
	gitArgs := []string{"clone"}
	if branch != "" {
		gitArgs = append(gitArgs, "-b", branch)
	}
	gitArgs = append(gitArgs, url, destPath)

	cmd := &vmm.Command{
		Cmd:  "git",
		Args: gitArgs,
	}

	execResult, err := w.orchestrator.ExecuteCommand(ctx, vmID, cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to execute git clone: %w", err)
	}

	result := &storage.PrepStepResult{
		ExitCode: execResult.ExitCode,
		Stdout:   execResult.Stdout,
		Stderr:   execResult.Stderr,
	}

	if execResult.ExitCode != 0 {
		return result, fmt.Errorf("git clone failed with exit code %d: %s", execResult.ExitCode, execResult.Stderr)
	}

	return result, nil
}

// executeScript executes a script prep step
func (w *Worker) executeScript(ctx context.Context, vmID string, config map[string]interface{}) (*storage.PrepStepResult, error) {
	content, _ := config["content"].(string)
	interpreter, _ := config["interpreter"].(string)

	if content == "" {
		return nil, fmt.Errorf("script requires 'content' in config")
	}

	if interpreter == "" {
		interpreter = "bash"
	}

	cmd := &vmm.Command{
		Cmd:  interpreter,
		Args: []string{"-c", content},
	}

	execResult, err := w.orchestrator.ExecuteCommand(ctx, vmID, cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to execute script: %w", err)
	}

	result := &storage.PrepStepResult{
		ExitCode: execResult.ExitCode,
		Stdout:   execResult.Stdout,
		Stderr:   execResult.Stderr,
	}

	if execResult.ExitCode != 0 {
		return result, fmt.Errorf("script failed with exit code %d: %s", execResult.ExitCode, execResult.Stderr)
	}

	return result, nil
}

// executeEnvVar sets an environment variable
func (w *Worker) executeEnvVar(ctx context.Context, vmID string, workspaceID uuid.UUID, config map[string]interface{}) (*storage.PrepStepResult, error) {
	key, _ := config["key"].(string)
	value, _ := config["value"].(string)
	secretName, _ := config["secret_name"].(string)
	isSecret, _ := config["is_secret"].(bool)

	if key == "" {
		return nil, fmt.Errorf("env_var requires 'key' in config")
	}

	// If using a secret, retrieve its value
	if secretName != "" && w.workspaceService != nil {
		secretValue, err := w.workspaceService.GetDecryptedSecretByName(ctx, workspaceID, secretName)
		if err != nil {
			return nil, fmt.Errorf("failed to get secret %s: %w", secretName, err)
		}
		value = secretValue
	}

	// Export the environment variable by adding to .bashrc
	exportCmd := fmt.Sprintf("echo 'export %s=\"%s\"' >> ~/.bashrc", key, value)
	if isSecret {
		// For secrets, don't log the value
		log.Printf("Setting secret environment variable: %s", key)
	}

	cmd := &vmm.Command{
		Cmd:  "bash",
		Args: []string{"-c", exportCmd},
	}

	execResult, err := w.orchestrator.ExecuteCommand(ctx, vmID, cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to set env var: %w", err)
	}

	result := &storage.PrepStepResult{
		ExitCode: execResult.ExitCode,
		Stdout:   execResult.Stdout,
		Stderr:   execResult.Stderr,
	}

	if execResult.ExitCode != 0 {
		return result, fmt.Errorf("failed to set env var with exit code %d", execResult.ExitCode)
	}

	return result, nil
}

// HandleWorkspaceDelete handles workspace deletion tasks
func (w *Worker) HandleWorkspaceDelete(ctx context.Context, task *queue.Task) (*queue.TaskResult, error) {
	startTime := time.Now()

	var payload WorkspaceDeletePayload
	if err := queue.UnmarshalPayload(task.Payload, &payload); err != nil {
		return nil, fmt.Errorf("invalid payload: %w", err)
	}

	workspaceID, err := uuid.Parse(payload.WorkspaceID)
	if err != nil {
		return nil, fmt.Errorf("invalid workspace_id: %w", err)
	}

	log.Printf("Deleting workspace: %s", workspaceID)

	// Get workspace to find VM
	workspace, err := w.store.Workspaces().Get(ctx, workspaceID)
	if err != nil {
		// Workspace may already be deleted
		log.Printf("Workspace not found, may be already deleted: %v", err)
	} else if workspace.VMID != nil {
		// Delete the VM
		vmID := workspace.VMID.String()
		if err := w.orchestrator.DeleteVM(ctx, vmID); err != nil {
			log.Printf("Warning: Failed to delete VM: %v", err)
		}

		// Untrack VM resources
		w.mu.Lock()
		delete(w.runningVMs, vmID)
		w.mu.Unlock()

		// Delete VM from database
		if err := w.store.VMs().Delete(ctx, *workspace.VMID); err != nil {
			log.Printf("Warning: Failed to delete VM from database: %v", err)
		}
	}

	// Delete workspace (cascade will delete prep steps, secrets, etc.)
	if err := w.store.Workspaces().Delete(ctx, workspaceID); err != nil {
		return &queue.TaskResult{
			TaskID:    task.ID,
			Success:   false,
			Error:     fmt.Sprintf("failed to delete workspace: %v", err),
			Duration:  time.Since(startTime),
			StartedAt: startTime,
		}, nil
	}

	// Update worker resources
	if w.workerInfo != nil {
		w.updateWorkerResources(ctx)
	}

	log.Printf("✓ Workspace deleted: %s", workspaceID)

	return &queue.TaskResult{
		TaskID:    task.ID,
		Success:   true,
		Result:    map[string]interface{}{"workspace_id": workspaceID.String()},
		Duration:  time.Since(startTime),
		StartedAt: startTime,
	}, nil
}

// HandlePromptExecute handles prompt execution tasks with on-demand VM spawning
func (w *Worker) HandlePromptExecute(ctx context.Context, task *queue.Task) (*queue.TaskResult, error) {
	startTime := time.Now()

	var payload PromptExecutePayload
	if err := queue.UnmarshalPayload(task.Payload, &payload); err != nil {
		return nil, fmt.Errorf("invalid payload: %w", err)
	}

	promptID, err := uuid.Parse(payload.PromptID)
	if err != nil {
		return nil, fmt.Errorf("invalid prompt_id: %w", err)
	}

	workspaceID, err := uuid.Parse(payload.WorkspaceID)
	if err != nil {
		return nil, fmt.Errorf("invalid workspace_id: %w", err)
	}

	// Get prompt task
	promptTask, err := w.store.PromptTasks().Get(ctx, promptID)
	if err != nil {
		return nil, fmt.Errorf("prompt not found: %w", err)
	}

	// Get workspace
	workspace, err := w.store.Workspaces().Get(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("workspace not found: %w", err)
	}

	var vmID string

	// On-demand VM spawning: If workspace has no VM, spawn one from environment template
	if workspace.VMID == nil {
		log.Printf("Workspace %s has no VM, checking for environment template...", workspaceID)

		// Check if workspace has an environment
		if workspace.EnvironmentID == nil {
			errResult := &storage.PromptResult{Error: "workspace has no VM and no environment template configured"}
			w.store.PromptTasks().UpdateStatus(ctx, promptID, "failed", errResult)
			return &queue.TaskResult{
				TaskID:    task.ID,
				Success:   false,
				Error:     "workspace has no VM and no environment template configured",
				Duration:  time.Since(startTime),
				StartedAt: startTime,
			}, nil
		}

		// Get environment template
		env, err := w.store.Environments().Get(ctx, *workspace.EnvironmentID)
		if err != nil {
			errResult := &storage.PromptResult{Error: fmt.Sprintf("failed to get environment: %v", err)}
			w.store.PromptTasks().UpdateStatus(ctx, promptID, "failed", errResult)
			return &queue.TaskResult{
				TaskID:    task.ID,
				Success:   false,
				Error:     fmt.Sprintf("failed to get environment: %v", err),
				Duration:  time.Since(startTime),
				StartedAt: startTime,
			}, nil
		}

		log.Printf("Spawning on-demand VM for workspace %s using environment %s", workspaceID, env.Name)

		// Update workspace status to indicate VM is being created
		w.store.Workspaces().UpdateStatus(ctx, workspaceID, "spawning")

		// Spawn VM using environment template
		vm, err := w.spawnVMFromEnvironment(ctx, workspace, env)
		if err != nil {
			w.store.Workspaces().UpdateStatus(ctx, workspaceID, "failed")
			errResult := &storage.PromptResult{Error: fmt.Sprintf("failed to spawn VM: %v", err)}
			w.store.PromptTasks().UpdateStatus(ctx, promptID, "failed", errResult)
			return &queue.TaskResult{
				TaskID:    task.ID,
				Success:   false,
				Error:     fmt.Sprintf("failed to spawn VM: %v", err),
				Duration:  time.Since(startTime),
				StartedAt: startTime,
			}, nil
		}

		vmID = vm.ID

		// Setup MCP servers from environment config
		if len(env.MCPServers) > 0 {
			log.Printf("Setting up %d MCP server(s) for workspace %s", len(env.MCPServers), workspaceID)
			if err := w.setupClaudeCodeMCP(ctx, vmID, env); err != nil {
				log.Printf("Warning: Failed to setup MCP servers: %v", err)
				// Don't fail the entire operation, just log the warning
			}
		}

		// Clone git repository if configured
		if env.GitRepoURL != "" {
			log.Printf("Cloning repository %s for workspace %s", env.GitRepoURL, workspaceID)
			if err := w.cloneEnvironmentRepo(ctx, vmID, env); err != nil {
				log.Printf("Warning: Failed to clone repository: %v", err)
				// Don't fail the entire operation, just log the warning
			}
		}

		// Set environment variables from environment config
		if len(env.EnvVars) > 0 {
			log.Printf("Setting %d environment variable(s) for workspace %s", len(env.EnvVars), workspaceID)
			if err := w.setEnvironmentVars(ctx, vmID, env.EnvVars); err != nil {
				log.Printf("Warning: Failed to set environment variables: %v", err)
			}
		}

		// Mark workspace as ready
		w.store.Workspaces().SetReady(ctx, workspaceID)
		log.Printf("✓ On-demand VM %s spawned successfully for workspace %s", vmID, workspaceID)
	} else {
		vmID = workspace.VMID.String()
	}

	// Reset idle timer since workspace is now active
	w.store.Workspaces().UpdateIdleSince(ctx, workspaceID, nil)

	log.Printf("Executing prompt on workspace %s (vm=%s)", workspaceID, vmID)

	// Update prompt status to running
	w.store.PromptTasks().UpdateStatus(ctx, promptID, "running", nil)

	// Build the command based on AI assistant
	var cmd *vmm.Command
	workingDir := workspace.WorkingDirectory
	if promptTask.WorkingDirectory != nil && *promptTask.WorkingDirectory != "" {
		workingDir = *promptTask.WorkingDirectory
	}

	// Create a script that changes to the working directory and runs the AI assistant
	var aiCmd string
	switch workspace.AIAssistant {
	case "claude-code":
		// Claude Code takes prompt as input
		aiCmd = fmt.Sprintf("cd %s && claude-code --dangerously-skip-permissions '%s'", workingDir, escapeShellArg(promptTask.Prompt))
	case "ampcode", "amp":
		// Ampcode CLI
		aiCmd = fmt.Sprintf("cd %s && amp '%s'", workingDir, escapeShellArg(promptTask.Prompt))
	default:
		// Default to claude-code
		aiCmd = fmt.Sprintf("cd %s && claude-code --dangerously-skip-permissions '%s'", workingDir, escapeShellArg(promptTask.Prompt))
	}

	cmd = &vmm.Command{
		Cmd:  "bash",
		Args: []string{"-c", aiCmd},
	}

	execResult, err := w.orchestrator.ExecuteCommand(ctx, vmID, cmd)
	if err != nil {
		errResult := &storage.PromptResult{Error: err.Error()}
		w.store.PromptTasks().UpdateStatus(ctx, promptID, "failed", errResult)
		return &queue.TaskResult{
			TaskID:    task.ID,
			Success:   false,
			Error:     err.Error(),
			Duration:  time.Since(startTime),
			StartedAt: startTime,
		}, nil
	}

	durationMS := int(time.Since(startTime).Milliseconds())
	result := &storage.PromptResult{
		ExitCode:   execResult.ExitCode,
		Stdout:     execResult.Stdout,
		Stderr:     execResult.Stderr,
		DurationMS: durationMS,
	}

	status := "completed"
	if execResult.ExitCode != 0 {
		status = "failed"
		errMsg := fmt.Sprintf("prompt execution failed with exit code %d", execResult.ExitCode)
		result.Error = errMsg
		log.Printf("✗ Prompt execution failed: %s", errMsg)
	} else {
		log.Printf("✓ Prompt executed successfully on workspace %s", workspaceID)
	}

	w.store.PromptTasks().UpdateStatus(ctx, promptID, status, result)

	// Set idle timer since workspace is now idle again
	idleNow := time.Now()
	w.store.Workspaces().UpdateIdleSince(ctx, workspaceID, &idleNow)

	taskResult := map[string]interface{}{
		"prompt_id":    promptID.String(),
		"workspace_id": workspaceID.String(),
		"exit_code":    execResult.ExitCode,
		"stdout":       execResult.Stdout,
		"stderr":       execResult.Stderr,
	}

	return &queue.TaskResult{
		TaskID:    task.ID,
		Success:   execResult.ExitCode == 0,
		Result:    taskResult,
		Duration:  time.Since(startTime),
		StartedAt: startTime,
	}, nil
}

// spawnVMFromEnvironment creates and starts a VM using environment template configuration
func (w *Worker) spawnVMFromEnvironment(ctx context.Context, workspace *storage.Workspace, env *storage.Environment) (*types.VM, error) {
	// Create VM config from environment template
	vmID := uuid.New().String()
	vmConfig := &types.VMConfig{
		ID:         vmID,
		KernelPath: "/var/firecracker/vmlinux",
		RootFSPath: "", // Will be set by orchestrator.CreateVM() from template
		SocketPath: fmt.Sprintf("/tmp/aetherium-vm-%s.sock", vmID),
		VCPUCount:  env.VCPUs,
		MemoryMB:   env.MemoryMB,
	}

	// Create VM
	vm, err := w.orchestrator.CreateVM(ctx, vmConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create VM: %w", err)
	}

	// Start VM
	if err := w.orchestrator.StartVM(ctx, vm.ID); err != nil {
		return nil, fmt.Errorf("failed to start VM: %w", err)
	}

	// Wait for agent to be ready
	time.Sleep(5 * time.Second)

	// Install tools from environment template
	log.Printf("Installing tools from environment template for VM %s...", vm.ID)

	// Start with default tools
	defaultTools := tools.GetDefaultTools()
	allTools := append(defaultTools, env.Tools...)

	// Always include claude-code for AI assistant
	hasClaudeCode := false
	for _, t := range allTools {
		if t == "claude-code" {
			hasClaudeCode = true
			break
		}
	}
	if !hasClaudeCode {
		allTools = append(allTools, "claude-code")
	}

	// Remove duplicates
	toolSet := make(map[string]bool)
	uniqueTools := []string{}
	for _, tool := range allTools {
		if !toolSet[tool] {
			toolSet[tool] = true
			uniqueTools = append(uniqueTools, tool)
		}
	}

	// Install tools with timeout
	if err := w.toolInstaller.InstallToolsWithTimeout(ctx, vm.ID, uniqueTools, nil, 20*time.Minute); err != nil {
		log.Printf("Warning: Tool installation failed (workspace may be partially usable): %v", err)
	} else {
		log.Printf("✓ All tools installed successfully in VM %s", vm.ID)
	}

	// Link VM to workspace
	vmUUID, _ := uuid.Parse(vm.ID)
	if err := w.store.Workspaces().SetVMID(ctx, workspace.ID, vmUUID); err != nil {
		log.Printf("Warning: Failed to link VM to workspace: %v", err)
	}

	// Track VM resources
	w.mu.Lock()
	w.runningVMs[vm.ID] = &vmResourceUsage{
		VCPUs:    env.VCPUs,
		MemoryMB: int64(env.MemoryMB),
	}
	w.tasksProcessed++
	w.mu.Unlock()

	// Store VM in database
	kernelPath := vmConfig.KernelPath
	rootfsPath := vmConfig.RootFSPath
	socketPath := vmConfig.SocketPath

	var workerID *string
	if w.workerInfo != nil {
		workerID = &w.workerInfo.ID
	}

	dbVM := &storage.VM{
		ID:           vmUUID,
		Name:         fmt.Sprintf("env-%s-ws-%s", env.Name, workspace.Name),
		Orchestrator: "firecracker",
		Status:       string(vm.Status),
		KernelPath:   &kernelPath,
		RootFSPath:   &rootfsPath,
		SocketPath:   &socketPath,
		VCPUCount:    &env.VCPUs,
		MemoryMB:     &env.MemoryMB,
		WorkerID:     workerID,
		CreatedAt:    time.Now(),
		Metadata: map[string]interface{}{
			"workspace_id":   workspace.ID.String(),
			"environment_id": env.ID.String(),
			"on_demand":      true,
		},
	}

	if err := w.store.VMs().Create(ctx, dbVM); err != nil {
		log.Printf("Warning: Failed to store VM in database: %v", err)
	}

	// Update worker resources in database
	if w.workerInfo != nil {
		w.updateWorkerResources(ctx)
	}

	return vm, nil
}

// setupClaudeCodeMCP configures MCP servers in the VM by writing to ~/.claude/settings.json
func (w *Worker) setupClaudeCodeMCP(ctx context.Context, vmID string, env *storage.Environment) error {
	// Generate Claude Code settings from environment MCP config
	settings := mcp.GenerateClaudeSettings(env.MCPServers, env.EnvVars)
	settingsJSON, err := settings.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to generate MCP settings JSON: %w", err)
	}

	// Create ~/.claude directory and write settings.json
	script := fmt.Sprintf(`mkdir -p ~/.claude && cat > ~/.claude/settings.json << 'MCPEOF'
%s
MCPEOF`, string(settingsJSON))

	cmd := &vmm.Command{
		Cmd:  "bash",
		Args: []string{"-c", script},
	}

	execResult, err := w.orchestrator.ExecuteCommand(ctx, vmID, cmd)
	if err != nil {
		return fmt.Errorf("failed to execute MCP setup command: %w", err)
	}

	if execResult.ExitCode != 0 {
		return fmt.Errorf("MCP setup failed with exit code %d: %s", execResult.ExitCode, execResult.Stderr)
	}

	log.Printf("✓ MCP servers configured in VM %s", vmID)
	return nil
}

// cloneEnvironmentRepo clones the git repository specified in the environment
func (w *Worker) cloneEnvironmentRepo(ctx context.Context, vmID string, env *storage.Environment) error {
	destPath := env.WorkingDirectory
	if destPath == "" {
		destPath = "/workspace"
	}

	// Build git clone command
	gitArgs := []string{"clone"}
	if env.GitBranch != "" {
		gitArgs = append(gitArgs, "-b", env.GitBranch)
	}
	gitArgs = append(gitArgs, env.GitRepoURL, destPath)

	cmd := &vmm.Command{
		Cmd:  "git",
		Args: gitArgs,
	}

	execResult, err := w.orchestrator.ExecuteCommand(ctx, vmID, cmd)
	if err != nil {
		return fmt.Errorf("failed to execute git clone: %w", err)
	}

	if execResult.ExitCode != 0 {
		return fmt.Errorf("git clone failed with exit code %d: %s", execResult.ExitCode, execResult.Stderr)
	}

	log.Printf("✓ Repository cloned to %s in VM %s", destPath, vmID)
	return nil
}

// setEnvironmentVars sets environment variables in the VM from the environment config
func (w *Worker) setEnvironmentVars(ctx context.Context, vmID string, envVars map[string]string) error {
	// Build a script to append all env vars to .bashrc
	var exportCmds string
	for key, value := range envVars {
		// Escape special characters in the value
		escapedValue := escapeShellArg(value)
		exportCmds += fmt.Sprintf("echo 'export %s=\"%s\"' >> ~/.bashrc\n", key, escapedValue)
	}

	cmd := &vmm.Command{
		Cmd:  "bash",
		Args: []string{"-c", exportCmds},
	}

	execResult, err := w.orchestrator.ExecuteCommand(ctx, vmID, cmd)
	if err != nil {
		return fmt.Errorf("failed to set environment variables: %w", err)
	}

	if execResult.ExitCode != 0 {
		return fmt.Errorf("setting env vars failed with exit code %d: %s", execResult.ExitCode, execResult.Stderr)
	}

	log.Printf("✓ Environment variables set in VM %s", vmID)
	return nil
}

// escapeShellArg escapes a string for use in shell commands
func escapeShellArg(s string) string {
	// Replace single quotes with '\'' (end quote, escaped quote, start quote)
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

// StartIdleCleanup starts a background goroutine that periodically cleans up idle VMs
// VMs are destroyed when they've been idle longer than their environment's idle_timeout_seconds
func (w *Worker) StartIdleCleanup(ctx context.Context, checkInterval time.Duration) {
	log.Printf("Starting idle VM cleanup worker (check interval: %v)", checkInterval)

	go func() {
		ticker := time.NewTicker(checkInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Printf("Idle cleanup worker stopped")
				return
			case <-ticker.C:
				w.cleanupIdleWorkspaces(ctx)
			}
		}
	}()
}

// cleanupIdleWorkspaces finds and destroys VMs that have been idle too long
func (w *Worker) cleanupIdleWorkspaces(ctx context.Context) {
	// Get all workspaces with VMs that have an idle_since timestamp
	idleWorkspaces, err := w.store.Workspaces().ListIdleWithVMs(ctx)
	if err != nil {
		log.Printf("Error listing idle workspaces: %v", err)
		return
	}

	if len(idleWorkspaces) == 0 {
		return
	}

	log.Printf("Checking %d idle workspace(s) for cleanup...", len(idleWorkspaces))

	for _, workspace := range idleWorkspaces {
		// Skip if no idle timestamp or no environment
		if workspace.IdleSince == nil || workspace.EnvironmentID == nil {
			continue
		}

		// Get environment to check idle timeout
		env, err := w.store.Environments().Get(ctx, *workspace.EnvironmentID)
		if err != nil {
			log.Printf("Warning: Failed to get environment for workspace %s: %v", workspace.ID, err)
			continue
		}

		// Check if idle time exceeds timeout
		idleDuration := time.Since(*workspace.IdleSince)
		timeoutDuration := time.Duration(env.IdleTimeoutSeconds) * time.Second

		if idleDuration > timeoutDuration {
			log.Printf("Workspace %s has been idle for %v (timeout: %v), destroying VM...",
				workspace.ID, idleDuration.Round(time.Second), timeoutDuration)

			// Destroy the VM
			if err := w.destroyWorkspaceVM(ctx, workspace); err != nil {
				log.Printf("Error destroying VM for workspace %s: %v", workspace.ID, err)
				continue
			}

			log.Printf("✓ VM destroyed for idle workspace %s", workspace.ID)
		}
	}
}

// destroyWorkspaceVM destroys the VM associated with a workspace and updates workspace status
func (w *Worker) destroyWorkspaceVM(ctx context.Context, workspace *storage.Workspace) error {
	if workspace.VMID == nil {
		return nil
	}

	vmID := workspace.VMID.String()

	// Delete VM via orchestrator
	if err := w.orchestrator.DeleteVM(ctx, vmID); err != nil {
		return fmt.Errorf("failed to delete VM: %w", err)
	}

	// Untrack VM resources
	w.mu.Lock()
	delete(w.runningVMs, vmID)
	w.mu.Unlock()

	// Delete VM from database
	if err := w.store.VMs().Delete(ctx, *workspace.VMID); err != nil {
		log.Printf("Warning: Failed to delete VM from database: %v", err)
	}

	// Clear VM ID from workspace and update status to idle
	if err := w.store.Workspaces().ClearVMID(ctx, workspace.ID); err != nil {
		return fmt.Errorf("failed to clear workspace VM ID: %w", err)
	}

	// Update workspace status to indicate it's idle (no VM)
	if err := w.store.Workspaces().UpdateStatus(ctx, workspace.ID, "idle"); err != nil {
		log.Printf("Warning: Failed to update workspace status: %v", err)
	}

	// Clear idle timestamp
	if err := w.store.Workspaces().UpdateIdleSince(ctx, workspace.ID, nil); err != nil {
		log.Printf("Warning: Failed to clear idle timestamp: %v", err)
	}

	// Update worker resources
	if w.workerInfo != nil {
		w.updateWorkerResources(ctx)
	}

	return nil
}
