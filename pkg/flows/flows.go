package flows

import (
	"context"
	"fmt"
	"time"

	"github.com/aetherium/aetherium/pkg/queue"
	"github.com/aetherium/aetherium/pkg/service"
	"github.com/aetherium/aetherium/pkg/storage"
	"github.com/aetherium/aetherium/pkg/types"
	"github.com/google/uuid"
)

// FlowExecutor executes predefined flows triggered from integrations
type FlowExecutor struct {
	taskService *service.TaskService
	store       storage.Store
	flows       map[string]*FlowDefinition
}

// FlowDefinition defines a multi-step workflow
type FlowDefinition struct {
	ID          string
	Name        string
	Description string
	Steps       []FlowStep
	Parameters  []FlowParameter
}

// FlowStep represents a single step in a flow
type FlowStep struct {
	ID          string
	Name        string
	Type        StepType
	Config      map[string]interface{}
	DependsOn   []string // IDs of steps that must complete first
	ContinueOnError bool
}

// StepType defines the type of flow step
type StepType string

const (
	StepTypeCreateVM   StepType = "create_vm"
	StepTypeExecuteCmd StepType = "execute_cmd"
	StepTypeDeleteVM   StepType = "delete_vm"
	StepTypeWait       StepType = "wait"
	StepTypeNotify     StepType = "notify"
)

// FlowParameter defines a parameter that can be passed to a flow
type FlowParameter struct {
	Name        string
	Description string
	Required    bool
	Default     interface{}
}

// FlowExecution tracks the execution of a flow
type FlowExecution struct {
	ID           uuid.UUID
	FlowID       string
	TriggerBy    string // User ID or integration
	TriggerFrom  string // Channel ID or source
	Status       FlowStatus
	StartedAt    time.Time
	CompletedAt  *time.Time
	Parameters   map[string]interface{}
	StepResults  map[string]*StepResult
	Error        string
	Resources    *FlowResources // VMs and other resources created
}

// FlowStatus represents the status of a flow execution
type FlowStatus string

const (
	FlowStatusPending   FlowStatus = "pending"
	FlowStatusRunning   FlowStatus = "running"
	FlowStatusCompleted FlowStatus = "completed"
	FlowStatusFailed    FlowStatus = "failed"
	FlowStatusCancelled FlowStatus = "cancelled"
)

// StepResult tracks the result of a single step
type StepResult struct {
	StepID      string
	Status      FlowStatus
	StartedAt   time.Time
	CompletedAt *time.Time
	Output      map[string]interface{}
	Error       string
}

// FlowResources tracks resources created during flow execution
type FlowResources struct {
	VMIDs      []string
	TaskIDs    []uuid.UUID
	Artifacts  []string
}

// NewFlowExecutor creates a new flow executor
func NewFlowExecutor(taskService *service.TaskService, store storage.Store) *FlowExecutor {
	executor := &FlowExecutor{
		taskService: taskService,
		store:       store,
		flows:       make(map[string]*FlowDefinition),
	}

	// Register built-in flows
	executor.registerBuiltInFlows()

	return executor
}

// RegisterFlow registers a custom flow definition
func (e *FlowExecutor) RegisterFlow(flow *FlowDefinition) {
	e.flows[flow.ID] = flow
}

// ListFlows returns all available flows
func (e *FlowExecutor) ListFlows() []*FlowDefinition {
	flows := make([]*FlowDefinition, 0, len(e.flows))
	for _, flow := range e.flows {
		flows = append(flows, flow)
	}
	return flows
}

// GetFlow retrieves a flow definition by ID
func (e *FlowExecutor) GetFlow(flowID string) (*FlowDefinition, error) {
	flow, ok := e.flows[flowID]
	if !ok {
		return nil, fmt.Errorf("flow not found: %s", flowID)
	}
	return flow, nil
}

// ExecuteFlow executes a flow with the given parameters
func (e *FlowExecutor) ExecuteFlow(ctx context.Context, flowID string, params map[string]interface{}, triggerBy, triggerFrom string) (*FlowExecution, error) {
	flow, err := e.GetFlow(flowID)
	if err != nil {
		return nil, err
	}

	// Validate parameters
	if err := e.validateParameters(flow, params); err != nil {
		return nil, fmt.Errorf("parameter validation failed: %w", err)
	}

	// Create flow execution
	execution := &FlowExecution{
		ID:          uuid.New(),
		FlowID:      flowID,
		TriggerBy:   triggerBy,
		TriggerFrom: triggerFrom,
		Status:      FlowStatusPending,
		StartedAt:   time.Now(),
		Parameters:  params,
		StepResults: make(map[string]*StepResult),
		Resources: &FlowResources{
			VMIDs:   []string{},
			TaskIDs: []uuid.UUID{},
			Artifacts: []string{},
		},
	}

	// Execute flow asynchronously
	go e.executeFlowAsync(context.Background(), execution, flow)

	return execution, nil
}

// executeFlowAsync executes a flow asynchronously
func (e *FlowExecutor) executeFlowAsync(ctx context.Context, execution *FlowExecution, flow *FlowDefinition) {
	execution.Status = FlowStatusRunning

	// Execute steps in order
	for _, step := range flow.Steps {
		// Check if dependencies are satisfied
		if err := e.checkDependencies(execution, step); err != nil {
			execution.Status = FlowStatusFailed
			execution.Error = fmt.Sprintf("dependency check failed for step %s: %v", step.ID, err)
			return
		}

		// Execute step
		result := e.executeStep(ctx, execution, step)
		execution.StepResults[step.ID] = result

		// Check if step failed
		if result.Status == FlowStatusFailed {
			if !step.ContinueOnError {
				execution.Status = FlowStatusFailed
				execution.Error = fmt.Sprintf("step %s failed: %s", step.ID, result.Error)
				now := time.Now()
				execution.CompletedAt = &now
				return
			}
		}
	}

	// Flow completed successfully
	execution.Status = FlowStatusCompleted
	now := time.Now()
	execution.CompletedAt = &now
}

// executeStep executes a single flow step
func (e *FlowExecutor) executeStep(ctx context.Context, execution *FlowExecution, step FlowStep) *StepResult {
	result := &StepResult{
		StepID:    step.ID,
		Status:    FlowStatusRunning,
		StartedAt: time.Now(),
		Output:    make(map[string]interface{}),
	}

	var err error

	switch step.Type {
	case StepTypeCreateVM:
		err = e.executeCreateVM(ctx, execution, step, result)
	case StepTypeExecuteCmd:
		err = e.executeCommand(ctx, execution, step, result)
	case StepTypeDeleteVM:
		err = e.executeDeleteVM(ctx, execution, step, result)
	case StepTypeWait:
		err = e.executeWait(ctx, execution, step, result)
	case StepTypeNotify:
		err = e.executeNotify(ctx, execution, step, result)
	default:
		err = fmt.Errorf("unknown step type: %s", step.Type)
	}

	now := time.Now()
	result.CompletedAt = &now

	if err != nil {
		result.Status = FlowStatusFailed
		result.Error = err.Error()
	} else {
		result.Status = FlowStatusCompleted
	}

	return result
}

// executeCreateVM creates a VM
func (e *FlowExecutor) executeCreateVM(ctx context.Context, execution *FlowExecution, step FlowStep, result *StepResult) error {
	name := e.getConfigString(step.Config, "name", fmt.Sprintf("flow-%s-vm", execution.ID.String()[:8]))
	vcpus := e.getConfigInt(step.Config, "vcpus", 1)
	memoryMB := e.getConfigInt(step.Config, "memory_mb", 512)
	tools := e.getConfigStringSlice(step.Config, "tools", []string{})

	taskID, err := e.taskService.CreateVMTaskWithTools(ctx, name, vcpus, memoryMB, tools, nil)
	if err != nil {
		return fmt.Errorf("failed to create VM task: %w", err)
	}

	execution.Resources.TaskIDs = append(execution.Resources.TaskIDs, taskID)
	result.Output["task_id"] = taskID.String()
	result.Output["vm_name"] = name

	// Wait for VM creation (with timeout)
	timeout := time.After(5 * time.Minute)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("VM creation timed out")
		case <-ticker.C:
			// Check if VM exists
			vm, err := e.taskService.GetVMByName(ctx, name)
			if err == nil && vm != nil {
				execution.Resources.VMIDs = append(execution.Resources.VMIDs, vm.ID)
				result.Output["vm_id"] = vm.ID
				return nil
			}
		}
	}
}

// executeCommand executes a command in a VM
func (e *FlowExecutor) executeCommand(ctx context.Context, execution *FlowExecution, step FlowStep, result *StepResult) error {
	vmID := e.getConfigString(step.Config, "vm_id", "")

	// If no VM ID specified, use the first VM created in this flow
	if vmID == "" && len(execution.Resources.VMIDs) > 0 {
		vmID = execution.Resources.VMIDs[0]
	}

	if vmID == "" {
		return fmt.Errorf("no VM ID specified and no VMs available")
	}

	command := e.getConfigString(step.Config, "command", "")
	if command == "" {
		return fmt.Errorf("command is required")
	}

	args := e.getConfigStringSlice(step.Config, "args", []string{})

	taskID, err := e.taskService.ExecuteCommandTask(ctx, vmID, command, args)
	if err != nil {
		return fmt.Errorf("failed to create execute task: %w", err)
	}

	execution.Resources.TaskIDs = append(execution.Resources.TaskIDs, taskID)
	result.Output["task_id"] = taskID.String()
	result.Output["vm_id"] = vmID
	result.Output["command"] = command

	// Wait for execution (with timeout)
	timeout := time.After(10 * time.Minute)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("command execution timed out")
		case <-ticker.C:
			// Check if execution completed
			executions, err := e.taskService.GetExecutions(ctx, vmID)
			if err == nil && len(executions) > 0 {
				// Find the latest execution
				for _, exec := range executions {
					if exec.CompletedAt != nil {
						result.Output["exit_code"] = exec.ExitCode
						result.Output["stdout"] = exec.Stdout
						result.Output["stderr"] = exec.Stderr

						if exec.ExitCode != 0 {
							return fmt.Errorf("command failed with exit code %d", exec.ExitCode)
						}
						return nil
					}
				}
			}
		}
	}
}

// executeDeleteVM deletes a VM
func (e *FlowExecutor) executeDeleteVM(ctx context.Context, execution *FlowExecution, step FlowStep, result *StepResult) error {
	vmID := e.getConfigString(step.Config, "vm_id", "")

	// If no VM ID specified, delete all VMs created in this flow
	if vmID == "" && len(execution.Resources.VMIDs) > 0 {
		for _, id := range execution.Resources.VMIDs {
			taskID, err := e.taskService.DeleteVMTask(ctx, id)
			if err != nil {
				return fmt.Errorf("failed to delete VM %s: %w", id, err)
			}
			execution.Resources.TaskIDs = append(execution.Resources.TaskIDs, taskID)
		}
		result.Output["deleted_vms"] = len(execution.Resources.VMIDs)
		return nil
	}

	if vmID == "" {
		return fmt.Errorf("no VM ID specified and no VMs available")
	}

	taskID, err := e.taskService.DeleteVMTask(ctx, vmID)
	if err != nil {
		return fmt.Errorf("failed to delete VM: %w", err)
	}

	execution.Resources.TaskIDs = append(execution.Resources.TaskIDs, taskID)
	result.Output["task_id"] = taskID.String()
	result.Output["vm_id"] = vmID

	return nil
}

// executeWait waits for a specified duration
func (e *FlowExecutor) executeWait(ctx context.Context, execution *FlowExecution, step FlowStep, result *StepResult) error {
	durationSec := e.getConfigInt(step.Config, "duration_seconds", 0)
	if durationSec <= 0 {
		return fmt.Errorf("duration_seconds must be positive")
	}

	time.Sleep(time.Duration(durationSec) * time.Second)
	result.Output["waited_seconds"] = durationSec

	return nil
}

// executeNotify sends a notification (placeholder)
func (e *FlowExecutor) executeNotify(ctx context.Context, execution *FlowExecution, step FlowStep, result *StepResult) error {
	message := e.getConfigString(step.Config, "message", "Notification from flow")
	result.Output["message"] = message

	// TODO: Integrate with notification service

	return nil
}

// checkDependencies checks if all dependencies for a step are satisfied
func (e *FlowExecutor) checkDependencies(execution *FlowExecution, step FlowStep) error {
	for _, depID := range step.DependsOn {
		result, ok := execution.StepResults[depID]
		if !ok {
			return fmt.Errorf("dependency %s not executed", depID)
		}
		if result.Status == FlowStatusFailed {
			return fmt.Errorf("dependency %s failed", depID)
		}
		if result.Status != FlowStatusCompleted {
			return fmt.Errorf("dependency %s not completed", depID)
		}
	}
	return nil
}

// validateParameters validates flow parameters
func (e *FlowExecutor) validateParameters(flow *FlowDefinition, params map[string]interface{}) error {
	for _, param := range flow.Parameters {
		value, ok := params[param.Name]
		if !ok || value == nil {
			if param.Required {
				return fmt.Errorf("required parameter missing: %s", param.Name)
			}
			// Set default value
			if param.Default != nil {
				params[param.Name] = param.Default
			}
		}
	}
	return nil
}

// Helper methods to extract config values
func (e *FlowExecutor) getConfigString(config map[string]interface{}, key string, defaultValue string) string {
	if val, ok := config[key].(string); ok {
		return val
	}
	return defaultValue
}

func (e *FlowExecutor) getConfigInt(config map[string]interface{}, key string, defaultValue int) int {
	if val, ok := config[key].(int); ok {
		return val
	}
	if val, ok := config[key].(float64); ok {
		return int(val)
	}
	return defaultValue
}

func (e *FlowExecutor) getConfigStringSlice(config map[string]interface{}, key string, defaultValue []string) []string {
	if val, ok := config[key].([]string); ok {
		return val
	}
	if val, ok := config[key].([]interface{}); ok {
		result := make([]string, 0, len(val))
		for _, v := range val {
			if s, ok := v.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	return defaultValue
}

// registerBuiltInFlows registers the built-in flow definitions
func (e *FlowExecutor) registerBuiltInFlows() {
	// Quick Test Flow - Create VM, run test, delete VM
	e.RegisterFlow(&FlowDefinition{
		ID:          "quick-test",
		Name:        "Quick Test",
		Description: "Create a VM, run a command, and clean up",
		Parameters: []FlowParameter{
			{Name: "command", Description: "Command to execute", Required: true},
			{Name: "args", Description: "Command arguments", Required: false, Default: []string{}},
		},
		Steps: []FlowStep{
			{
				ID:   "create-vm",
				Name: "Create Test VM",
				Type: StepTypeCreateVM,
				Config: map[string]interface{}{
					"vcpus":     1,
					"memory_mb": 512,
					"tools":     []string{"git", "nodejs"},
				},
			},
			{
				ID:        "run-command",
				Name:      "Execute Command",
				Type:      StepTypeExecuteCmd,
				DependsOn: []string{"create-vm"},
				Config: map[string]interface{}{
					"command": "{{.command}}",
					"args":    "{{.args}}",
				},
			},
			{
				ID:        "cleanup",
				Name:      "Delete VM",
				Type:      StepTypeDeleteVM,
				DependsOn: []string{"run-command"},
				ContinueOnError: true,
			},
		},
	})

	// Git Clone and Build Flow
	e.RegisterFlow(&FlowDefinition{
		ID:          "git-build",
		Name:        "Git Clone and Build",
		Description: "Clone a git repository and run build commands",
		Parameters: []FlowParameter{
			{Name: "repo_url", Description: "Git repository URL", Required: true},
			{Name: "build_command", Description: "Build command", Required: false, Default: "npm install && npm run build"},
		},
		Steps: []FlowStep{
			{
				ID:   "create-vm",
				Name: "Create Build VM",
				Type: StepTypeCreateVM,
				Config: map[string]interface{}{
					"vcpus":     2,
					"memory_mb": 2048,
					"tools":     []string{"git", "nodejs", "bun"},
				},
			},
			{
				ID:        "clone",
				Name:      "Clone Repository",
				Type:      StepTypeExecuteCmd,
				DependsOn: []string{"create-vm"},
				Config: map[string]interface{}{
					"command": "git",
					"args":    []string{"clone", "{{.repo_url}}"},
				},
			},
			{
				ID:        "build",
				Name:      "Run Build",
				Type:      StepTypeExecuteCmd,
				DependsOn: []string{"clone"},
				Config: map[string]interface{}{
					"command": "bash",
					"args":    []string{"-c", "{{.build_command}}"},
				},
			},
			{
				ID:        "cleanup",
				Name:      "Delete VM",
				Type:      StepTypeDeleteVM,
				DependsOn: []string{"build"},
				ContinueOnError: true,
			},
		},
	})

	// Benchmark Flow
	e.RegisterFlow(&FlowDefinition{
		ID:          "benchmark",
		Name:        "Run Benchmark",
		Description: "Create VM, run benchmark, report results",
		Parameters: []FlowParameter{
			{Name: "benchmark_command", Description: "Benchmark command to run", Required: true},
		},
		Steps: []FlowStep{
			{
				ID:   "create-vm",
				Name: "Create Benchmark VM",
				Type: StepTypeCreateVM,
				Config: map[string]interface{}{
					"vcpus":     2,
					"memory_mb": 1024,
					"tools":     []string{"git", "nodejs", "python"},
				},
			},
			{
				ID:        "benchmark",
				Name:      "Run Benchmark",
				Type:      StepTypeExecuteCmd,
				DependsOn: []string{"create-vm"},
				Config: map[string]interface{}{
					"command": "bash",
					"args":    []string{"-c", "{{.benchmark_command}}"},
				},
			},
			{
				ID:        "cleanup",
				Name:      "Delete VM",
				Type:      StepTypeDeleteVM,
				DependsOn: []string{"benchmark"},
				ContinueOnError: true,
			},
		},
	})
}
