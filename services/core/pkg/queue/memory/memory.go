package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aetherium/aetherium/services/core/pkg/queue"
	"github.com/aetherium/aetherium/libs/types/pkg/domain"
)

// MemoryQueue is an in-memory implementation of TaskQueue for testing
type MemoryQueue struct {
	tasks    chan *types.Task
	handlers map[string]queue.HandlerFunc
	mu       sync.RWMutex
	workers  int
	done     chan struct{}
	wg       sync.WaitGroup
}

// NewMemoryQueue creates a new in-memory task queue
func NewMemoryQueue(bufferSize int, workers int) *MemoryQueue {
	if bufferSize <= 0 {
		bufferSize = 100
	}
	if workers <= 0 {
		workers = 1
	}

	return &MemoryQueue{
		tasks:    make(chan *types.Task, bufferSize),
		handlers: make(map[string]queue.HandlerFunc),
		workers:  workers,
		done:     make(chan struct{}),
	}
}

// Enqueue adds a task to the queue
func (m *MemoryQueue) Enqueue(ctx context.Context, task *types.Task) error {
	if task == nil {
		return fmt.Errorf("task cannot be nil")
	}

	select {
	case m.tasks <- task:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-m.done:
		return fmt.Errorf("queue is closed")
	}
}

// Dequeue retrieves a task from the queue (blocking)
func (m *MemoryQueue) Dequeue(ctx context.Context) (*types.Task, error) {
	select {
	case task := <-m.tasks:
		return task, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-m.done:
		return nil, fmt.Errorf("queue is closed")
	}
}

// RegisterHandler registers a handler for a specific task type
func (m *MemoryQueue) RegisterHandler(taskType string, handler queue.HandlerFunc) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.handlers[taskType]; exists {
		return fmt.Errorf("handler for task type '%s' already registered", taskType)
	}

	m.handlers[taskType] = handler
	return nil
}

// Start starts the worker pool to process tasks
func (m *MemoryQueue) Start(ctx context.Context) error {
	for i := 0; i < m.workers; i++ {
		m.wg.Add(1)
		go m.worker(ctx, i)
	}
	return nil
}

func (m *MemoryQueue) worker(ctx context.Context, id int) {
	defer m.wg.Done()

	for {
		select {
		case task := <-m.tasks:
			if err := m.processTask(ctx, task); err != nil {
				// In a real implementation, we'd log this
				// For now, we'll just continue
				task.Status = types.TaskStatusFailed
				errMsg := err.Error()
				task.ErrorMessage = &errMsg
			}
		case <-ctx.Done():
			return
		case <-m.done:
			return
		}
	}
}

func (m *MemoryQueue) processTask(ctx context.Context, task *types.Task) error {
	m.mu.RLock()
	handler, exists := m.handlers[task.Type]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("no handler registered for task type: %s", task.Type)
	}

	// Update task status
	task.Status = types.TaskStatusRunning
	task.StartedAt = timePtr(time.Now())

	// Execute handler
	err := handler(ctx, task)

	// Update task status based on result
	task.CompletedAt = timePtr(time.Now())
	if err != nil {
		task.Status = types.TaskStatusFailed
		errMsg := err.Error()
		task.ErrorMessage = &errMsg
		return err
	}

	task.Status = types.TaskStatusCompleted
	return nil
}

// Stop gracefully stops the queue
func (m *MemoryQueue) Stop(ctx context.Context) error {
	close(m.done)

	// Wait for workers to finish with timeout
	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// UpdateStatus updates the status of a task (no-op for in-memory queue)
func (m *MemoryQueue) UpdateStatus(ctx context.Context, taskID string, status types.TaskStatus) error {
	// In a real implementation, this would update task in storage
	// For in-memory queue, tasks are updated directly by handlers
	return nil
}

// DeleteTask removes a task from the queue (no-op for in-memory queue)
func (m *MemoryQueue) DeleteTask(ctx context.Context, taskID string) error {
	// In a real implementation, this would remove task from storage
	// For in-memory queue, tasks are removed after processing
	return nil
}

// GetQueueSize returns the current number of tasks in the queue
func (m *MemoryQueue) GetQueueSize() int {
	return len(m.tasks)
}

// Health checks if the queue is operational
func (m *MemoryQueue) Health(ctx context.Context) error {
	select {
	case <-m.done:
		return fmt.Errorf("queue is closed")
	default:
		return nil
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}

// Ensure MemoryQueue implements TaskQueue interface
var _ queue.TaskQueue = (*MemoryQueue)(nil)
