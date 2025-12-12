package queue

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// TaskType defines the type of task
type TaskType string

const (
	// VM task types
	TaskTypeVMCreate    TaskType = "vm:create"
	TaskTypeVMStart     TaskType = "vm:start"
	TaskTypeVMStop      TaskType = "vm:stop"
	TaskTypeVMDelete    TaskType = "vm:delete"
	TaskTypeVMExecute   TaskType = "vm:execute"
	TaskTypeJobExecute  TaskType = "job:execute"
	TaskTypeIntegration TaskType = "integration:run"

	// Workspace task types
	TaskTypeWorkspaceCreate TaskType = "workspace:create"
	TaskTypeWorkspaceDelete TaskType = "workspace:delete"
	TaskTypePromptExecute   TaskType = "prompt:execute"
)

// Task represents a distributed task
type Task struct {
	ID       uuid.UUID              `json:"id"`
	Type     TaskType               `json:"type"`
	Payload  map[string]interface{} `json:"payload"`
	Priority int                    `json:"priority"`
}

// TaskResult represents the result of a task execution
type TaskResult struct {
	TaskID    uuid.UUID              `json:"task_id"`
	Success   bool                   `json:"success"`
	Result    map[string]interface{} `json:"result,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Duration  time.Duration          `json:"duration"`
	StartedAt time.Time              `json:"started_at"`
}

// TaskHandler is a function that processes a task
type TaskHandler func(ctx context.Context, task *Task) (*TaskResult, error)

// TaskOptions configures task enqueueing
type TaskOptions struct {
	ProcessAt   time.Time     // Schedule task for future processing
	MaxRetry    int           // Max number of retries
	Timeout     time.Duration // Task execution timeout
	Queue       string        // Queue name (default: "default")
	Priority    int           // Priority (higher = more important)
}

// Queue interface for distributed task processing
type Queue interface {
	// Enqueue adds a task to the queue
	Enqueue(ctx context.Context, task *Task, opts *TaskOptions) error

	// RegisterHandler registers a handler for a task type
	RegisterHandler(taskType TaskType, handler TaskHandler) error

	// Start starts processing tasks
	Start(ctx context.Context) error

	// Stop gracefully stops the queue
	Stop(ctx context.Context) error

	// Stats returns queue statistics
	Stats(ctx context.Context) (*QueueStats, error)
}

// QueueStats represents queue statistics
type QueueStats struct {
	Pending    int            `json:"pending"`
	Active     int            `json:"active"`
	Completed  int            `json:"completed"`
	Failed     int            `json:"failed"`
	Retrying   int            `json:"retrying"`
	ByType     map[string]int `json:"by_type"`
	Throughput float64        `json:"throughput"` // tasks/second
}

// MarshalPayload marshals a payload to JSON
func MarshalPayload(v interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}

	return payload, nil
}

// UnmarshalPayload unmarshals a payload from JSON
func UnmarshalPayload(payload map[string]interface{}, v interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, v)
}
