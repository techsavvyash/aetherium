package factories

import (
	"context"
	"fmt"

	"github.com/aetherium/aetherium/pkg/storage"
	memorystorage "github.com/aetherium/aetherium/pkg/storage/memory"
)

// DefaultStorageFactory creates StateStore instances
type DefaultStorageFactory struct{}

// NewStorageFactory creates a new storage factory
func NewStorageFactory() *DefaultStorageFactory {
	return &DefaultStorageFactory{}
}

// Create creates a StateStore based on the provider name
func (f *DefaultStorageFactory) Create(ctx context.Context, provider string, cfg map[string]interface{}) (storage.StateStore, error) {
	switch provider {
	case "memory":
		return memorystorage.NewMemoryStore(), nil
	case "postgres":
		// TODO: Implement PostgreSQL store
		return nil, fmt.Errorf("postgres store not yet implemented")
	default:
		return nil, fmt.Errorf("unsupported storage provider: %s", provider)
	}
}

// SupportedProviders returns list of supported providers
func (f *DefaultStorageFactory) SupportedProviders() []string {
	return []string{"memory", "postgres"}
}
