package memory

import (
	"context"
	"fmt"
	"sync"

	"github.com/aetherium/aetherium/pkg/events"
	"github.com/aetherium/aetherium/pkg/types"
)

// MemoryEventBus is an in-memory implementation of EventBus for testing
type MemoryEventBus struct {
	subscribers map[string][]subscriber
	mu          sync.RWMutex
	idCounter   int
}

type subscriber struct {
	id      string
	handler events.EventHandler
}

// NewMemoryEventBus creates a new in-memory event bus
func NewMemoryEventBus() *MemoryEventBus {
	return &MemoryEventBus{
		subscribers: make(map[string][]subscriber),
	}
}

// Publish publishes an event to all subscribers of the topic
func (m *MemoryEventBus) Publish(ctx context.Context, topic string, event *types.Event) error {
	m.mu.RLock()
	subs := m.subscribers[topic]
	m.mu.RUnlock()

	// Call all handlers for this topic
	for _, sub := range subs {
		// Call handler in goroutine to avoid blocking
		go func(handler events.EventHandler) {
			_ = handler(ctx, event)
		}(sub.handler)
	}

	return nil
}

// Subscribe subscribes to a topic and returns a subscription ID
func (m *MemoryEventBus) Subscribe(ctx context.Context, topic string, handler events.EventHandler) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.idCounter++
	subID := fmt.Sprintf("sub-%d", m.idCounter)

	sub := subscriber{
		id:      subID,
		handler: handler,
	}

	m.subscribers[topic] = append(m.subscribers[topic], sub)
	return subID, nil
}

// Unsubscribe removes a subscription
func (m *MemoryEventBus) Unsubscribe(ctx context.Context, topic string, subscriptionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	subs, exists := m.subscribers[topic]
	if !exists {
		return fmt.Errorf("topic '%s' not found", topic)
	}

	// Find and remove the subscriber
	for i, sub := range subs {
		if sub.id == subscriptionID {
			// Remove from slice
			m.subscribers[topic] = append(subs[:i], subs[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("subscription '%s' not found in topic '%s'", subscriptionID, topic)
}

// Health checks if the event bus is operational
func (m *MemoryEventBus) Health(ctx context.Context) error {
	return nil
}

// Close closes the event bus
func (m *MemoryEventBus) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Clear all subscribers
	m.subscribers = make(map[string][]subscriber)
	return nil
}

// GetSubscriberCount returns the number of subscribers for a topic
func (m *MemoryEventBus) GetSubscriberCount(topic string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.subscribers[topic])
}

// Ensure MemoryEventBus implements EventBus interface
var _ events.EventBus = (*MemoryEventBus)(nil)
