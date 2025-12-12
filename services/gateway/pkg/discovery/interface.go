package discovery

import (
	"context"
)

// ServiceRegistry provides an interface for service discovery and worker registration
type ServiceRegistry interface {
	// Register registers a worker with the service discovery system
	// Returns an error if registration fails
	Register(ctx context.Context, worker *WorkerInfo) error

	// Deregister removes a worker from the service discovery system
	Deregister(ctx context.Context, workerID string) error

	// UpdateStatus updates a worker's status (e.g., active -> draining)
	UpdateStatus(ctx context.Context, workerID string, status WorkerStatus) error

	// UpdateResources updates a worker's resource usage
	UpdateResources(ctx context.Context, workerID string, resources *WorkerResources) error

	// Heartbeat sends a heartbeat to keep the worker registration alive
	// Returns an error if the heartbeat fails
	Heartbeat(ctx context.Context, workerID string) error

	// GetWorker retrieves information about a specific worker
	GetWorker(ctx context.Context, workerID string) (*WorkerInfo, error)

	// ListWorkers lists all registered workers
	ListWorkers(ctx context.Context) ([]*WorkerInfo, error)

	// ListWorkersWithFilter lists workers matching the filter criteria
	ListWorkersWithFilter(ctx context.Context, filter *WorkerFilter) ([]*WorkerInfo, error)

	// Watch watches for changes in worker registrations
	// Returns a channel that emits WorkerEvent when workers join/leave/change
	Watch(ctx context.Context) (<-chan *WorkerEvent, error)

	// Health checks if the service discovery system is healthy
	Health(ctx context.Context) error

	// Close closes the connection to the service discovery system
	Close() error
}

// WorkerEventType represents the type of worker event
type WorkerEventType string

const (
	WorkerEventJoined  WorkerEventType = "joined"  // Worker registered
	WorkerEventLeft    WorkerEventType = "left"    // Worker deregistered
	WorkerEventUpdated WorkerEventType = "updated" // Worker info changed
)

// WorkerEvent represents a change in worker registration
type WorkerEvent struct {
	Type   WorkerEventType `json:"type"`
	Worker *WorkerInfo     `json:"worker"`
}

// RegistryConfig contains configuration for service discovery
type RegistryConfig struct {
	// Provider specifies the service discovery provider (consul, etcd, etc.)
	Provider string `yaml:"provider" json:"provider"`

	// HealthCheck configuration
	HealthCheck HealthCheckConfig `yaml:"health_check" json:"health_check"`

	// Provider-specific configuration
	Consul *ConsulConfig `yaml:"consul,omitempty" json:"consul,omitempty"`
	Etcd   *EtcdConfig   `yaml:"etcd,omitempty" json:"etcd,omitempty"`
}

// ConsulConfig contains Consul-specific configuration
type ConsulConfig struct {
	Address    string `yaml:"address" json:"address"`         // Consul agent address
	Datacenter string `yaml:"datacenter" json:"datacenter"`   // Datacenter name
	Token      string `yaml:"token,omitempty" json:"token,omitempty"` // ACL token
	Scheme     string `yaml:"scheme" json:"scheme"`           // http or https

	// Service name for workers in Consul
	ServiceName string `yaml:"service_name" json:"service_name"`
}

// EtcdConfig contains etcd-specific configuration (for future implementation)
type EtcdConfig struct {
	Endpoints []string `yaml:"endpoints" json:"endpoints"` // etcd endpoints
	Username  string   `yaml:"username,omitempty" json:"username,omitempty"`
	Password  string   `yaml:"password,omitempty" json:"password,omitempty"`

	// Prefix for worker keys in etcd
	KeyPrefix string `yaml:"key_prefix" json:"key_prefix"`
}

// DefaultRegistryConfig returns default configuration for service discovery
func DefaultRegistryConfig() *RegistryConfig {
	return &RegistryConfig{
		Provider:    "consul",
		HealthCheck: DefaultHealthCheckConfig(),
		Consul: &ConsulConfig{
			Address:     "localhost:8500",
			Datacenter:  "dc1",
			Scheme:      "http",
			ServiceName: "aetherium-worker",
		},
	}
}
