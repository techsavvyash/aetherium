package events

import (
	"context"

	"github.com/aetherium/aetherium/pkg/types"
)

// EventBus defines the interface for event bus implementations
type EventBus interface {
	// Publish publishes an event to a topic
	Publish(ctx context.Context, topic string, event *types.Event) error

	// Subscribe subscribes to a topic with a handler
	// Returns a subscription ID that can be used to unsubscribe
	Subscribe(ctx context.Context, topic string, handler EventHandler) (string, error)

	// Unsubscribe removes a subscription
	Unsubscribe(ctx context.Context, topic string, subscriptionID string) error

	// Health returns the health status of the event bus
	Health(ctx context.Context) error

	// Close closes the event bus connection
	Close() error
}

// EventHandler is a function that processes an event
type EventHandler func(ctx context.Context, event *types.Event) error

// Config represents event bus configuration
type Config struct {
	// Provider-specific configuration
	Options map[string]interface{}
}

// Topic constants for common events
const (
	TopicTaskCreated   = "task.created"
	TopicTaskStarted   = "task.started"
	TopicTaskCompleted = "task.completed"
	TopicTaskFailed    = "task.failed"

	TopicVMCreated = "vm.created"
	TopicVMStarted = "vm.started"
	TopicVMStopped = "vm.stopped"
	TopicVMFailed  = "vm.failed"

	TopicIntegrationWebhook = "integration.webhook_received"
)
