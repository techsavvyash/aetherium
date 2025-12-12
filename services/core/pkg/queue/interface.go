package queue

import (
	"context"

	"github.com/aetherium/aetherium/libs/types/pkg/domain"
)

// TaskQueue defines the interface for task queue implementations
type TaskQueue interface {
	// Enqueue adds a new task to the queue
	Enqueue(ctx context.Context, task *types.Task) error

	// Dequeue retrieves the next task from the queue
	// Returns nil if no tasks are available
	Dequeue(ctx context.Context) (*types.Task, error)

	// UpdateStatus updates the status of a task
	UpdateStatus(ctx context.Context, taskID string, status types.TaskStatus) error

	// DeleteTask removes a task from the queue
	DeleteTask(ctx context.Context, taskID string) error

	// RegisterHandler registers a handler function for a specific task type
	RegisterHandler(taskType string, handler HandlerFunc) error

	// Start begins processing tasks from the queue
	Start(ctx context.Context) error

	// Stop gracefully stops the queue processor
	Stop(ctx context.Context) error

	// Health returns the health status of the queue
	Health(ctx context.Context) error
}

// HandlerFunc is a function that processes a task
type HandlerFunc func(ctx context.Context, task *types.Task) error

// Config represents queue configuration
type Config struct {
	// Provider-specific configuration
	Options map[string]interface{}
}
