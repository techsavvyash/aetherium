package consul

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aetherium/aetherium/services/gateway/pkg/discovery"
	consulapi "github.com/hashicorp/consul/api"
)

// ConsulRegistry implements ServiceRegistry using HashiCorp Consul
type ConsulRegistry struct {
	client      *consulapi.Client
	config      *discovery.ConsulConfig
	healthCheck discovery.HealthCheckConfig
}

// NewConsulRegistry creates a new Consul-based service registry
func NewConsulRegistry(config *discovery.ConsulConfig, healthCheck discovery.HealthCheckConfig) (*ConsulRegistry, error) {
	consulConfig := consulapi.DefaultConfig()
	consulConfig.Address = config.Address
	consulConfig.Datacenter = config.Datacenter
	consulConfig.Scheme = config.Scheme
	if config.Token != "" {
		consulConfig.Token = config.Token
	}

	client, err := consulapi.NewClient(consulConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Consul client: %w", err)
	}

	return &ConsulRegistry{
		client:      client,
		config:      config,
		healthCheck: healthCheck,
	}, nil
}

// Register registers a worker with Consul
func (c *ConsulRegistry) Register(ctx context.Context, worker *discovery.WorkerInfo) error {
	// Marshal worker info as metadata
	metaBytes, err := json.Marshal(worker)
	if err != nil {
		return fmt.Errorf("failed to marshal worker info: %w", err)
	}

	// Create service registration
	registration := &consulapi.AgentServiceRegistration{
		ID:   worker.ID,
		Name: c.config.ServiceName,
		Tags: c.buildTags(worker),
		Meta: map[string]string{
			"worker_info": string(metaBytes),
			"zone":        worker.Zone,
			"hostname":    worker.Hostname,
			"status":      string(worker.Status),
		},
		Address: worker.Address,
		Check: &consulapi.AgentServiceCheck{
			TTL:                            c.healthCheck.Interval.String(),
			DeregisterCriticalServiceAfter: c.healthCheck.DeregisterAge.String(),
			Status:                         consulapi.HealthPassing,
		},
	}

	// Register with Consul
	if err := c.client.Agent().ServiceRegister(registration); err != nil {
		return fmt.Errorf("failed to register worker with Consul: %w", err)
	}

	return nil
}

// Deregister removes a worker from Consul
func (c *ConsulRegistry) Deregister(ctx context.Context, workerID string) error {
	if err := c.client.Agent().ServiceDeregister(workerID); err != nil {
		return fmt.Errorf("failed to deregister worker: %w", err)
	}
	return nil
}

// UpdateStatus updates a worker's status
func (c *ConsulRegistry) UpdateStatus(ctx context.Context, workerID string, status discovery.WorkerStatus) error {
	worker, err := c.GetWorker(ctx, workerID)
	if err != nil {
		return err
	}

	worker.Status = status
	worker.LastSeen = time.Now()

	// Re-register with updated status
	return c.Register(ctx, worker)
}

// UpdateResources updates a worker's resource usage
func (c *ConsulRegistry) UpdateResources(ctx context.Context, workerID string, resources *discovery.WorkerResources) error {
	worker, err := c.GetWorker(ctx, workerID)
	if err != nil {
		return err
	}

	worker.Resources = *resources
	worker.LastSeen = time.Now()

	// Re-register with updated resources
	return c.Register(ctx, worker)
}

// Heartbeat sends a heartbeat to keep the worker alive
func (c *ConsulRegistry) Heartbeat(ctx context.Context, workerID string) error {
	checkID := fmt.Sprintf("service:%s", workerID)
	if err := c.client.Agent().UpdateTTL(checkID, "Worker is healthy", consulapi.HealthPassing); err != nil {
		return fmt.Errorf("failed to send heartbeat: %w", err)
	}
	return nil
}

// GetWorker retrieves information about a specific worker
func (c *ConsulRegistry) GetWorker(ctx context.Context, workerID string) (*discovery.WorkerInfo, error) {
	service, _, err := c.client.Agent().Service(workerID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get worker: %w", err)
	}

	if service == nil {
		return nil, fmt.Errorf("worker not found: %s", workerID)
	}

	return c.serviceToWorkerInfo(service)
}

// ListWorkers lists all registered workers
func (c *ConsulRegistry) ListWorkers(ctx context.Context) ([]*discovery.WorkerInfo, error) {
	services, err := c.client.Agent().ServicesWithFilter(
		fmt.Sprintf("Service == %q", c.config.ServiceName),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list workers: %w", err)
	}

	workers := make([]*discovery.WorkerInfo, 0, len(services))
	for _, service := range services {
		worker, err := c.serviceToWorkerInfo(service)
		if err != nil {
			// Log error but continue with other workers
			continue
		}
		workers = append(workers, worker)
	}

	return workers, nil
}

// ListWorkersWithFilter lists workers matching the filter criteria
func (c *ConsulRegistry) ListWorkersWithFilter(ctx context.Context, filter *discovery.WorkerFilter) ([]*discovery.WorkerInfo, error) {
	allWorkers, err := c.ListWorkers(ctx)
	if err != nil {
		return nil, err
	}

	filtered := make([]*discovery.WorkerInfo, 0)
	for _, worker := range allWorkers {
		if filter.Matches(worker) {
			filtered = append(filtered, worker)
		}
	}

	return filtered, nil
}

// Watch watches for changes in worker registrations
func (c *ConsulRegistry) Watch(ctx context.Context) (<-chan *discovery.WorkerEvent, error) {
	eventChan := make(chan *discovery.WorkerEvent, 100)

	go func() {
		defer close(eventChan)

		lastIndex := uint64(0)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			// Watch for service changes
			services, meta, err := c.client.Health().Service(
				c.config.ServiceName,
				"",
				false,
				&consulapi.QueryOptions{
					WaitIndex: lastIndex,
					WaitTime:  5 * time.Minute,
				},
			)
			if err != nil {
				time.Sleep(1 * time.Second)
				continue
			}

			// Update last index
			if meta.LastIndex <= lastIndex {
				continue
			}
			lastIndex = meta.LastIndex

			// Process service changes
			for _, entry := range services {
				worker, err := c.healthEntryToWorkerInfo(entry)
				if err != nil {
					continue
				}

				// Determine event type based on health status
				eventType := discovery.WorkerEventUpdated
				if len(entry.Checks) > 0 {
					for _, check := range entry.Checks {
						if check.Status == consulapi.HealthPassing {
							eventType = discovery.WorkerEventJoined
						} else if check.Status == consulapi.HealthCritical {
							eventType = discovery.WorkerEventLeft
						}
					}
				}

				eventChan <- &discovery.WorkerEvent{
					Type:   eventType,
					Worker: worker,
				}
			}
		}
	}()

	return eventChan, nil
}

// Health checks if Consul is healthy
func (c *ConsulRegistry) Health(ctx context.Context) error {
	_, err := c.client.Agent().Self()
	if err != nil {
		return fmt.Errorf("Consul health check failed: %w", err)
	}
	return nil
}

// Close closes the connection to Consul
func (c *ConsulRegistry) Close() error {
	// Consul client doesn't need explicit closing
	return nil
}

// Helper functions

func (c *ConsulRegistry) buildTags(worker *discovery.WorkerInfo) []string {
	tags := []string{
		fmt.Sprintf("zone:%s", worker.Zone),
		fmt.Sprintf("status:%s", worker.Status),
	}

	// Add capabilities as tags
	for _, cap := range worker.Capabilities {
		tags = append(tags, fmt.Sprintf("capability:%s", cap))
	}

	// Add labels as tags
	for key, value := range worker.Labels {
		tags = append(tags, fmt.Sprintf("label:%s=%s", key, value))
	}

	return tags
}

func (c *ConsulRegistry) serviceToWorkerInfo(service *consulapi.AgentService) (*discovery.WorkerInfo, error) {
	// Try to unmarshal from metadata
	if workerInfoJSON, ok := service.Meta["worker_info"]; ok {
		var worker discovery.WorkerInfo
		if err := json.Unmarshal([]byte(workerInfoJSON), &worker); err == nil {
			return &worker, nil
		}
	}

	// Fallback: construct from available metadata
	worker := &discovery.WorkerInfo{
		ID:       service.ID,
		Hostname: service.Meta["hostname"],
		Address:  service.Address,
		Zone:     service.Meta["zone"],
		Status:   discovery.WorkerStatus(service.Meta["status"]),
		LastSeen: time.Now(),
		Labels:   make(map[string]string),
		Metadata: service.Meta,
	}

	return worker, nil
}

func (c *ConsulRegistry) healthEntryToWorkerInfo(entry *consulapi.ServiceEntry) (*discovery.WorkerInfo, error) {
	return c.serviceToWorkerInfo(entry.Service)
}
