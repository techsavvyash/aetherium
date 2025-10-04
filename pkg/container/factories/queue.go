package factories

import (
	"context"
	"fmt"

	"github.com/aetherium/aetherium/pkg/config"
	"github.com/aetherium/aetherium/pkg/queue"
	"github.com/aetherium/aetherium/pkg/queue/memory"
)

// DefaultQueueFactory creates TaskQueue instances
type DefaultQueueFactory struct{}

// NewQueueFactory creates a new queue factory
func NewQueueFactory() *DefaultQueueFactory {
	return &DefaultQueueFactory{}
}

// Create creates a TaskQueue based on the provider name
func (f *DefaultQueueFactory) Create(ctx context.Context, provider string, cfg map[string]interface{}) (queue.TaskQueue, error) {
	switch provider {
	case "memory":
		bufferSize := config.GetIntOrDefault(cfg, "buffer_size", 100)
		workers := config.GetIntOrDefault(cfg, "workers", 1)
		return memory.NewMemoryQueue(bufferSize, workers), nil
	case "redis":
		// TODO: Implement Redis queue with Asynq
		return nil, fmt.Errorf("redis queue not yet implemented")
	default:
		return nil, fmt.Errorf("unsupported queue provider: %s", provider)
	}
}

// SupportedProviders returns list of supported providers
func (f *DefaultQueueFactory) SupportedProviders() []string {
	return []string{"memory", "redis"}
}
