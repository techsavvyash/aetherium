package asynq

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/aetherium/aetherium/services/core/pkg/queue"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

// Config holds Asynq configuration
type Config struct {
	RedisAddr     string
	RedisPassword string
	RedisDB       int
	Concurrency   int // Number of worker goroutines
	Queues        map[string]int // Queue name -> priority
}

// AsynqQueue implements queue.Queue using Asynq
type AsynqQueue struct {
	client   *asynq.Client
	server   *asynq.Server
	mux      *asynq.ServeMux
	handlers map[queue.TaskType]queue.TaskHandler
	mu       sync.RWMutex
	config   Config
}

// NewQueue creates a new Asynq queue
func NewQueue(config Config) (*AsynqQueue, error) {
	redisOpt := asynq.RedisClientOpt{
		Addr:     config.RedisAddr,
		Password: config.RedisPassword,
		DB:       config.RedisDB,
	}

	client := asynq.NewClient(redisOpt)

	// Default queues if not specified
	if config.Queues == nil {
		config.Queues = map[string]int{
			"critical": 6,
			"high":     5,
			"default":  3,
			"low":      1,
		}
	}

	if config.Concurrency == 0 {
		config.Concurrency = 10
	}

	server := asynq.NewServer(
		redisOpt,
		asynq.Config{
			Concurrency: config.Concurrency,
			Queues:      config.Queues,
			ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
				// Log error (in production, send to logging system)
				fmt.Printf("Error processing task %s: %v\n", task.Type(), err)
			}),
		},
	)

	return &AsynqQueue{
		client:   client,
		server:   server,
		mux:      asynq.NewServeMux(),
		handlers: make(map[queue.TaskType]queue.TaskHandler),
		config:   config,
	}, nil
}

// Enqueue adds a task to the queue
func (q *AsynqQueue) Enqueue(ctx context.Context, task *queue.Task, opts *queue.TaskOptions) error {
	payload, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	asynqTask := asynq.NewTask(string(task.Type), payload)

	// Build options
	var asynqOpts []asynq.Option

	if opts != nil {
		if !opts.ProcessAt.IsZero() {
			asynqOpts = append(asynqOpts, asynq.ProcessAt(opts.ProcessAt))
		}
		if opts.MaxRetry > 0 {
			asynqOpts = append(asynqOpts, asynq.MaxRetry(opts.MaxRetry))
		} else {
			asynqOpts = append(asynqOpts, asynq.MaxRetry(3)) // Default
		}
		if opts.Timeout > 0 {
			asynqOpts = append(asynqOpts, asynq.Timeout(opts.Timeout))
		} else {
			asynqOpts = append(asynqOpts, asynq.Timeout(10*time.Minute)) // Default
		}
		if opts.Queue != "" {
			asynqOpts = append(asynqOpts, asynq.Queue(opts.Queue))
		}
		if opts.Priority > 0 {
			// Map priority to queue
			queueName := q.getQueueForPriority(opts.Priority)
			asynqOpts = append(asynqOpts, asynq.Queue(queueName))
		}
	} else {
		asynqOpts = append(asynqOpts, asynq.MaxRetry(3))
		asynqOpts = append(asynqOpts, asynq.Timeout(10*time.Minute))
	}

	_, err = q.client.EnqueueContext(ctx, asynqTask, asynqOpts...)
	if err != nil {
		return fmt.Errorf("failed to enqueue task: %w", err)
	}

	return nil
}

// RegisterHandler registers a handler for a task type
func (q *AsynqQueue) RegisterHandler(taskType queue.TaskType, handler queue.TaskHandler) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if _, exists := q.handlers[taskType]; exists {
		return fmt.Errorf("handler already registered for task type: %s", taskType)
	}

	q.handlers[taskType] = handler

	// Register with Asynq mux
	q.mux.HandleFunc(string(taskType), func(ctx context.Context, asynqTask *asynq.Task) error {
		var task queue.Task
		if err := json.Unmarshal(asynqTask.Payload(), &task); err != nil {
			return fmt.Errorf("failed to unmarshal task: %w", err)
		}

		startTime := time.Now()

		result, err := handler(ctx, &task)
		if err != nil {
			return err
		}

		if result != nil && !result.Success {
			return fmt.Errorf("task failed: %s", result.Error)
		}

		duration := time.Since(startTime)
		fmt.Printf("Task %s completed in %v\n", task.ID, duration)

		return nil
	})

	return nil
}

// Start starts processing tasks
func (q *AsynqQueue) Start(ctx context.Context) error {
	go func() {
		if err := q.server.Run(q.mux); err != nil {
			fmt.Printf("Asynq server error: %v\n", err)
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()

	return q.Stop(context.Background())
}

// Stop gracefully stops the queue
func (q *AsynqQueue) Stop(ctx context.Context) error {
	q.server.Shutdown()
	if err := q.client.Close(); err != nil {
		return fmt.Errorf("failed to close client: %w", err)
	}
	return nil
}

// Stats returns queue statistics
func (q *AsynqQueue) Stats(ctx context.Context) (*queue.QueueStats, error) {
	inspector := asynq.NewInspector(asynq.RedisClientOpt{
		Addr:     q.config.RedisAddr,
		Password: q.config.RedisPassword,
		DB:       q.config.RedisDB,
	})
	defer inspector.Close()

	stats := &queue.QueueStats{
		ByType: make(map[string]int),
	}

	// Get stats for all queues
	for queueName := range q.config.Queues {
		info, err := inspector.GetQueueInfo(queueName)
		if err != nil {
			return nil, fmt.Errorf("failed to get queue info: %w", err)
		}

		stats.Pending += info.Pending
		stats.Active += info.Active
		stats.Completed += info.Completed
		stats.Failed += info.Failed
	}

	return stats, nil
}

// getQueueForPriority maps priority to queue name
func (q *AsynqQueue) getQueueForPriority(priority int) string {
	switch {
	case priority >= 8:
		return "critical"
	case priority >= 5:
		return "high"
	case priority >= 2:
		return "default"
	default:
		return "low"
	}
}

// Helper function to create a task with UUID
func CreateTask(taskType queue.TaskType, payload map[string]interface{}) *queue.Task {
	return &queue.Task{
		ID:      uuid.New(),
		Type:    taskType,
		Payload: payload,
	}
}
