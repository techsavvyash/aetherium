package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/aetherium/aetherium/pkg/events"
	"github.com/aetherium/aetherium/pkg/types"
	"github.com/redis/go-redis/v9"
)

// RedisEventBus implements the EventBus interface using Redis Pub/Sub
type RedisEventBus struct {
	client        *redis.Client
	subscriptions map[string]map[string]events.EventHandler // topic -> subscriptionID -> handler
	subsMu        sync.RWMutex
	cancelFuncs   map[string]context.CancelFunc // topic -> cancel function
	cancelMu      sync.Mutex
}

// Config holds Redis event bus configuration
type Config struct {
	Addr     string
	Password string
	DB       int
}

// NewRedisEventBus creates a new Redis event bus
func NewRedisEventBus(config *Config) (*RedisEventBus, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     config.Addr,
		Password: config.Password,
		DB:       config.DB,
	})

	// Test connection
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &RedisEventBus{
		client:        client,
		subscriptions: make(map[string]map[string]events.EventHandler),
		cancelFuncs:   make(map[string]context.CancelFunc),
	}, nil
}

// Publish publishes an event to a topic
func (r *RedisEventBus) Publish(ctx context.Context, topic string, event *types.Event) error {
	// Ensure topic is set in event for subscribers
	eventCopy := *event
	eventCopy.Type = topic

	// Serialize event
	data, err := json.Marshal(eventCopy)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Publish to Redis channel
	if err := r.client.Publish(ctx, topic, data).Err(); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}

// Subscribe subscribes to a topic with a handler
func (r *RedisEventBus) Subscribe(ctx context.Context, topic string, handler events.EventHandler) (string, error) {
	r.subsMu.Lock()
	defer r.subsMu.Unlock()

	// Generate subscription ID
	subscriptionID := generateSubscriptionID()

	// Add handler to subscriptions
	if _, exists := r.subscriptions[topic]; !exists {
		r.subscriptions[topic] = make(map[string]events.EventHandler)
	}
	r.subscriptions[topic][subscriptionID] = handler

	// Start Redis subscriber for this topic if not already running
	if len(r.subscriptions[topic]) == 1 {
		if err := r.startTopicSubscriber(topic); err != nil {
			delete(r.subscriptions[topic], subscriptionID)
			return "", err
		}
	}

	return subscriptionID, nil
}

// startTopicSubscriber starts a Redis subscriber for a topic
func (r *RedisEventBus) startTopicSubscriber(topic string) error {
	pubsub := r.client.Subscribe(context.Background(), topic)

	// Test subscription
	if _, err := pubsub.Receive(context.Background()); err != nil {
		return fmt.Errorf("failed to subscribe to topic: %w", err)
	}

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	r.cancelMu.Lock()
	r.cancelFuncs[topic] = cancel
	r.cancelMu.Unlock()

	// Start message handler in background
	go func() {
		defer pubsub.Close()

		ch := pubsub.Channel()
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}

				// Deserialize event
				var event types.Event
				if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
					fmt.Printf("Error unmarshaling event: %v\n", err)
					continue
				}

				// Call all handlers for this topic
				r.subsMu.RLock()
				handlers := make([]events.EventHandler, 0, len(r.subscriptions[topic]))
				for _, h := range r.subscriptions[topic] {
					handlers = append(handlers, h)
				}
				r.subsMu.RUnlock()

				// Execute handlers
				for _, handler := range handlers {
					go func(h events.EventHandler) {
						if err := h(context.Background(), &event); err != nil {
							fmt.Printf("Error handling event: %v\n", err)
						}
					}(handler)
				}
			}
		}
	}()

	return nil
}

// Unsubscribe removes a subscription
func (r *RedisEventBus) Unsubscribe(ctx context.Context, topic string, subscriptionID string) error {
	r.subsMu.Lock()
	defer r.subsMu.Unlock()

	topicSubs, exists := r.subscriptions[topic]
	if !exists {
		return fmt.Errorf("topic %s not found", topic)
	}

	if _, exists := topicSubs[subscriptionID]; !exists {
		return fmt.Errorf("subscription %s not found", subscriptionID)
	}

	delete(topicSubs, subscriptionID)

	// If no more subscriptions for this topic, stop the subscriber
	if len(topicSubs) == 0 {
		delete(r.subscriptions, topic)

		r.cancelMu.Lock()
		if cancel, exists := r.cancelFuncs[topic]; exists {
			cancel()
			delete(r.cancelFuncs, topic)
		}
		r.cancelMu.Unlock()
	}

	return nil
}

// Health returns the health status of the event bus
func (r *RedisEventBus) Health(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// Close closes the event bus connection
func (r *RedisEventBus) Close() error {
	r.cancelMu.Lock()
	for _, cancel := range r.cancelFuncs {
		cancel()
	}
	r.cancelFuncs = make(map[string]context.CancelFunc)
	r.cancelMu.Unlock()

	return r.client.Close()
}

// generateSubscriptionID generates a unique subscription ID
func generateSubscriptionID() string {
	return fmt.Sprintf("sub_%d", time.Now().UnixNano())
}
