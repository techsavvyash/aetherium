package factories

import (
	"context"
	"fmt"

	"github.com/aetherium/aetherium/pkg/events"
	memoryevents "github.com/aetherium/aetherium/pkg/events/memory"
)

// DefaultEventBusFactory creates EventBus instances
type DefaultEventBusFactory struct{}

// NewEventBusFactory creates a new event bus factory
func NewEventBusFactory() *DefaultEventBusFactory {
	return &DefaultEventBusFactory{}
}

// Create creates an EventBus based on the provider name
func (f *DefaultEventBusFactory) Create(ctx context.Context, provider string, cfg map[string]interface{}) (events.EventBus, error) {
	switch provider {
	case "memory":
		return memoryevents.NewMemoryEventBus(), nil
	case "redis":
		// TODO: Implement Redis event bus (pub/sub)
		return nil, fmt.Errorf("redis event bus not yet implemented")
	default:
		return nil, fmt.Errorf("unsupported event bus provider: %s", provider)
	}
}

// SupportedProviders returns list of supported providers
func (f *DefaultEventBusFactory) SupportedProviders() []string {
	return []string{"memory", "redis"}
}
