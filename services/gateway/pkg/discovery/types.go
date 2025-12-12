package discovery

import (
	"time"

	"github.com/google/uuid"
)

// WorkerStatus represents the current status of a worker
type WorkerStatus string

const (
	WorkerStatusActive   WorkerStatus = "active"
	WorkerStatusDraining WorkerStatus = "draining"
	WorkerStatusOffline  WorkerStatus = "offline"
)

// WorkerInfo contains metadata about a worker node
type WorkerInfo struct {
	// Identity
	ID       string `json:"id"`        // Unique worker identifier
	Hostname string `json:"hostname"`  // Hostname of the worker machine
	Address  string `json:"address"`   // Network address (IP:port)

	// Status
	Status    WorkerStatus `json:"status"`
	LastSeen  time.Time    `json:"last_seen"`
	StartedAt time.Time    `json:"started_at"`

	// Location
	Zone   string            `json:"zone"`   // Geographic zone (e.g., us-west-1a)
	Labels map[string]string `json:"labels"` // Custom labels (env=prod, tier=gpu)

	// Capabilities
	Capabilities []string `json:"capabilities"` // Supported orchestrators (firecracker, docker)

	// Resources
	Resources WorkerResources `json:"resources"`

	// Metadata
	Metadata map[string]string `json:"metadata,omitempty"`
}

// WorkerResources tracks resource capacity and usage
type WorkerResources struct {
	// Capacity
	CPUCores  int   `json:"cpu_cores"`   // Total CPU cores
	MemoryMB  int64 `json:"memory_mb"`   // Total memory in MB
	DiskGB    int64 `json:"disk_gb"`     // Total disk space in GB

	// Current Usage
	UsedCPUCores int   `json:"used_cpu_cores"` // CPU cores in use
	UsedMemoryMB int64 `json:"used_memory_mb"` // Memory in use
	UsedDiskGB   int64 `json:"used_disk_gb"`   // Disk in use

	// VM Tracking
	VMCount    int `json:"vm_count"`     // Number of VMs running
	MaxVMs     int `json:"max_vms"`      // Maximum VMs allowed
}

// Available returns remaining available resources
func (r *WorkerResources) Available() WorkerResources {
	return WorkerResources{
		CPUCores:  r.CPUCores - r.UsedCPUCores,
		MemoryMB:  r.MemoryMB - r.UsedMemoryMB,
		DiskGB:    r.DiskGB - r.UsedDiskGB,
		VMCount:   r.MaxVMs - r.VMCount,
	}
}

// CanAllocate checks if worker can accommodate requested resources
func (r *WorkerResources) CanAllocate(cpuCores int, memoryMB int64) bool {
	available := r.Available()
	return available.CPUCores >= cpuCores &&
	       available.MemoryMB >= memoryMB &&
	       r.VMCount < r.MaxVMs
}

// WorkerFilter allows filtering workers by various criteria
type WorkerFilter struct {
	Zone         string            // Filter by zone
	Labels       map[string]string // Filter by labels (all must match)
	Capabilities []string          // Filter by capabilities (all must be present)
	Status       WorkerStatus      // Filter by status
	MinCPU       int               // Minimum available CPU cores
	MinMemoryMB  int64             // Minimum available memory
}

// Matches checks if a worker matches the filter
func (f *WorkerFilter) Matches(w *WorkerInfo) bool {
	// Check zone
	if f.Zone != "" && w.Zone != f.Zone {
		return false
	}

	// Check status
	if f.Status != "" && w.Status != f.Status {
		return false
	}

	// Check labels
	for key, value := range f.Labels {
		if w.Labels[key] != value {
			return false
		}
	}

	// Check capabilities
	capMap := make(map[string]bool)
	for _, cap := range w.Capabilities {
		capMap[cap] = true
	}
	for _, required := range f.Capabilities {
		if !capMap[required] {
			return false
		}
	}

	// Check resources
	available := w.Resources.Available()
	if f.MinCPU > 0 && available.CPUCores < f.MinCPU {
		return false
	}
	if f.MinMemoryMB > 0 && available.MemoryMB < f.MinMemoryMB {
		return false
	}

	return true
}

// HealthCheckConfig configures worker health checks
type HealthCheckConfig struct {
	Interval      time.Duration // How often to check
	Timeout       time.Duration // Health check timeout
	DeregisterAge time.Duration // Time before deregistering unhealthy worker
}

// DefaultHealthCheckConfig returns default health check configuration
func DefaultHealthCheckConfig() HealthCheckConfig {
	return HealthCheckConfig{
		Interval:      10 * time.Second,
		Timeout:       5 * time.Second,
		DeregisterAge: 1 * time.Minute,
	}
}

// WorkerMetrics contains metrics for a worker at a point in time
type WorkerMetrics struct {
	ID          uuid.UUID `json:"id"`
	WorkerID    string    `json:"worker_id"`
	Timestamp   time.Time `json:"timestamp"`

	// Resource metrics
	CPUUsage     float64 `json:"cpu_usage"`      // CPU usage percentage (0-100)
	MemoryUsage  float64 `json:"memory_usage"`   // Memory usage percentage (0-100)
	DiskUsage    float64 `json:"disk_usage"`     // Disk usage percentage (0-100)

	// VM metrics
	VMCount      int     `json:"vm_count"`       // Number of VMs
	TasksProcessed int   `json:"tasks_processed"` // Tasks processed in interval

	// Network metrics (optional)
	NetworkInMB  float64 `json:"network_in_mb,omitempty"`
	NetworkOutMB float64 `json:"network_out_mb,omitempty"`
}
