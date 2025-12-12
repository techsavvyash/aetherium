package integrations

import (
	"context"
	"fmt"
	"sync"
)

// Registry manages registered integrations
type Registry struct {
	mu           sync.RWMutex
	integrations map[string]Integration
}

// NewRegistry creates a new integration registry
func NewRegistry() *Registry {
	return &Registry{
		integrations: make(map[string]Integration),
	}
}

// Register adds an integration to the registry
func (r *Registry) Register(integration Integration) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := integration.Name()
	if _, exists := r.integrations[name]; exists {
		return fmt.Errorf("integration %s already registered", name)
	}

	r.integrations[name] = integration
	return nil
}

// Get retrieves an integration by name
func (r *Registry) Get(name string) (Integration, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	integration, exists := r.integrations[name]
	if !exists {
		return nil, fmt.Errorf("integration %s not found", name)
	}

	return integration, nil
}

// List returns all registered integration names
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.integrations))
	for name := range r.integrations {
		names = append(names, name)
	}
	return names
}

// Unregister removes an integration from the registry
func (r *Registry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.integrations[name]; !exists {
		return fmt.Errorf("integration %s not found", name)
	}

	delete(r.integrations, name)
	return nil
}

// Close closes all integrations
func (r *Registry) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var errs []error
	for name, integration := range r.integrations {
		if err := integration.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close %s: %w", name, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing integrations: %v", errs)
	}

	return nil
}

// HealthCheck checks the health of all integrations
func (r *Registry) HealthCheck(ctx context.Context) map[string]error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	results := make(map[string]error)
	for name, integration := range r.integrations {
		results[name] = integration.Health(ctx)
	}

	return results
}
