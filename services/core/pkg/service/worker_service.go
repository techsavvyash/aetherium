package service

import (
	"context"
	"fmt"
	"time"

	"github.com/aetherium/aetherium/services/gateway/pkg/discovery"
	"github.com/aetherium/aetherium/services/core/pkg/storage"
)

// WorkerService provides high-level worker management operations
type WorkerService struct {
	store    storage.Store
	registry discovery.ServiceRegistry
}

// NewWorkerService creates a new worker service
func NewWorkerService(store storage.Store, registry discovery.ServiceRegistry) *WorkerService {
	return &WorkerService{
		store:    store,
		registry: registry,
	}
}

// WorkerStats represents worker statistics
type WorkerStats struct {
	ID           string                 `json:"id"`
	Hostname     string                 `json:"hostname"`
	Address      string                 `json:"address"`
	Zone         string                 `json:"zone"`
	Status       string                 `json:"status"`
	Labels       map[string]string      `json:"labels"`
	Capabilities []string               `json:"capabilities"`

	// Resources
	CPUCores         int     `json:"cpu_cores"`
	MemoryMB         int64   `json:"memory_mb"`
	DiskGB           int64   `json:"disk_gb"`
	UsedCPUCores     int     `json:"used_cpu_cores"`
	UsedMemoryMB     int64   `json:"used_memory_mb"`
	UsedDiskGB       int64   `json:"used_disk_gb"`
	CPUUsagePercent  float64 `json:"cpu_usage_percent"`
	MemoryUsagePercent float64 `json:"memory_usage_percent"`

	// VMs
	VMCount int `json:"vm_count"`
	MaxVMs  int `json:"max_vms"`

	// Timestamps
	StartedAt time.Time `json:"started_at"`
	LastSeen  time.Time `json:"last_seen"`
	Uptime    string    `json:"uptime"`

	// Health
	IsHealthy bool `json:"is_healthy"`
}

// ClusterStats represents overall cluster statistics
type ClusterStats struct {
	TotalWorkers   int     `json:"total_workers"`
	ActiveWorkers  int     `json:"active_workers"`
	DrainingWorkers int    `json:"draining_workers"`
	OfflineWorkers int     `json:"offline_workers"`

	// Aggregate resources
	TotalCPUCores    int     `json:"total_cpu_cores"`
	TotalMemoryMB    int64   `json:"total_memory_mb"`
	UsedCPUCores     int     `json:"used_cpu_cores"`
	UsedMemoryMB     int64   `json:"used_memory_mb"`
	AvailableCPUCores int    `json:"available_cpu_cores"`
	AvailableMemoryMB int64  `json:"available_memory_mb"`
	ClusterCPUUsagePercent float64 `json:"cluster_cpu_usage_percent"`
	ClusterMemoryUsagePercent float64 `json:"cluster_memory_usage_percent"`

	// VMs
	TotalVMs    int `json:"total_vms"`
	MaxVMs      int `json:"max_vms"`
	AvailableVMSlots int `json:"available_vm_slots"`

	// Zones
	Zones map[string]int `json:"zones"` // zone -> worker count
}

// VMDistribution represents VM distribution across workers
type VMDistribution struct {
	WorkerID string `json:"worker_id"`
	Hostname string `json:"hostname"`
	Zone     string `json:"zone"`
	VMCount  int    `json:"vm_count"`
	VMs      []VMInfo `json:"vms"`
}

// VMInfo contains basic VM information
type VMInfo struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
	VCPUs  int    `json:"vcpus"`
	MemoryMB int  `json:"memory_mb"`
}

// ListWorkers returns all workers
func (s *WorkerService) ListWorkers(ctx context.Context) ([]*WorkerStats, error) {
	// Get workers from database
	dbWorkers, err := s.store.Workers().List(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list workers: %w", err)
	}

	stats := make([]*WorkerStats, len(dbWorkers))
	for i, w := range dbWorkers {
		stats[i] = s.workerToStats(w)
	}

	return stats, nil
}

// GetWorker returns a specific worker
func (s *WorkerService) GetWorker(ctx context.Context, workerID string) (*WorkerStats, error) {
	dbWorker, err := s.store.Workers().Get(ctx, workerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get worker: %w", err)
	}

	return s.workerToStats(dbWorker), nil
}

// ListActiveWorkers returns only active workers
func (s *WorkerService) ListActiveWorkers(ctx context.Context) ([]*WorkerStats, error) {
	dbWorkers, err := s.store.Workers().ListActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list active workers: %w", err)
	}

	stats := make([]*WorkerStats, len(dbWorkers))
	for i, w := range dbWorkers {
		stats[i] = s.workerToStats(w)
	}

	return stats, nil
}

// ListWorkersByZone returns workers in a specific zone
func (s *WorkerService) ListWorkersByZone(ctx context.Context, zone string) ([]*WorkerStats, error) {
	dbWorkers, err := s.store.Workers().ListByZone(ctx, zone)
	if err != nil {
		return nil, fmt.Errorf("failed to list workers by zone: %w", err)
	}

	stats := make([]*WorkerStats, len(dbWorkers))
	for i, w := range dbWorkers {
		stats[i] = s.workerToStats(w)
	}

	return stats, nil
}

// GetClusterStats returns overall cluster statistics
func (s *WorkerService) GetClusterStats(ctx context.Context) (*ClusterStats, error) {
	workers, err := s.store.Workers().List(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get workers: %w", err)
	}

	stats := &ClusterStats{
		Zones: make(map[string]int),
	}

	for _, w := range workers {
		stats.TotalWorkers++

		switch w.Status {
		case string(discovery.WorkerStatusActive):
			stats.ActiveWorkers++
		case string(discovery.WorkerStatusDraining):
			stats.DrainingWorkers++
		case string(discovery.WorkerStatusOffline):
			stats.OfflineWorkers++
		}

		// Aggregate resources
		stats.TotalCPUCores += w.CPUCores
		stats.TotalMemoryMB += w.MemoryMB
		stats.UsedCPUCores += w.UsedCPUCores
		stats.UsedMemoryMB += w.UsedMemoryMB

		// Aggregate VMs
		stats.TotalVMs += w.VMCount
		stats.MaxVMs += w.MaxVMs

		// Count zones
		if w.Zone != "" {
			stats.Zones[w.Zone]++
		}
	}

	// Calculate available resources
	stats.AvailableCPUCores = stats.TotalCPUCores - stats.UsedCPUCores
	stats.AvailableMemoryMB = stats.TotalMemoryMB - stats.UsedMemoryMB
	stats.AvailableVMSlots = stats.MaxVMs - stats.TotalVMs

	// Calculate usage percentages
	if stats.TotalCPUCores > 0 {
		stats.ClusterCPUUsagePercent = float64(stats.UsedCPUCores) / float64(stats.TotalCPUCores) * 100
	}
	if stats.TotalMemoryMB > 0 {
		stats.ClusterMemoryUsagePercent = float64(stats.UsedMemoryMB) / float64(stats.TotalMemoryMB) * 100
	}

	return stats, nil
}

// GetVMDistribution returns VM distribution across all workers
func (s *WorkerService) GetVMDistribution(ctx context.Context) ([]*VMDistribution, error) {
	workers, err := s.store.Workers().List(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get workers: %w", err)
	}

	distribution := make([]*VMDistribution, 0, len(workers))

	for _, worker := range workers {
		// Get VMs for this worker
		vms, err := s.store.VMs().List(ctx, map[string]interface{}{
			"worker_id": worker.ID,
		})
		if err != nil {
			// Log but continue
			continue
		}

		vmInfos := make([]VMInfo, len(vms))
		for i, vm := range vms {
			vmInfos[i] = VMInfo{
				ID:       vm.ID.String(),
				Name:     vm.Name,
				Status:   vm.Status,
				VCPUs:    *vm.VCPUCount,
				MemoryMB: *vm.MemoryMB,
			}
		}

		distribution = append(distribution, &VMDistribution{
			WorkerID: worker.ID,
			Hostname: worker.Hostname,
			Zone:     worker.Zone,
			VMCount:  len(vms),
			VMs:      vmInfos,
		})
	}

	return distribution, nil
}

// GetWorkerVMs returns all VMs running on a specific worker
func (s *WorkerService) GetWorkerVMs(ctx context.Context, workerID string) ([]VMInfo, error) {
	// Verify worker exists
	if _, err := s.store.Workers().Get(ctx, workerID); err != nil {
		return nil, fmt.Errorf("worker not found: %w", err)
	}

	// Get VMs for this worker
	vms, err := s.store.VMs().List(ctx, map[string]interface{}{
		"worker_id": workerID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get VMs: %w", err)
	}

	vmInfos := make([]VMInfo, len(vms))
	for i, vm := range vms {
		vmInfos[i] = VMInfo{
			ID:       vm.ID.String(),
			Name:     vm.Name,
			Status:   vm.Status,
			VCPUs:    *vm.VCPUCount,
			MemoryMB: *vm.MemoryMB,
		}
	}

	return vmInfos, nil
}

// DrainWorker marks a worker as draining (stops accepting new tasks)
func (s *WorkerService) DrainWorker(ctx context.Context, workerID string) error {
	// Update status in database
	if err := s.store.Workers().UpdateStatus(ctx, workerID, string(discovery.WorkerStatusDraining)); err != nil {
		return fmt.Errorf("failed to update worker status: %w", err)
	}

	// Update status in service discovery if configured
	if s.registry != nil {
		if err := s.registry.UpdateStatus(ctx, workerID, discovery.WorkerStatusDraining); err != nil {
			return fmt.Errorf("failed to update worker status in service discovery: %w", err)
		}
	}

	return nil
}

// ActivateWorker marks a worker as active (resumes accepting tasks)
func (s *WorkerService) ActivateWorker(ctx context.Context, workerID string) error {
	// Update status in database
	if err := s.store.Workers().UpdateStatus(ctx, workerID, string(discovery.WorkerStatusActive)); err != nil {
		return fmt.Errorf("failed to update worker status: %w", err)
	}

	// Update status in service discovery if configured
	if s.registry != nil {
		if err := s.registry.UpdateStatus(ctx, workerID, discovery.WorkerStatusActive); err != nil {
			return fmt.Errorf("failed to update worker status in service discovery: %w", err)
		}
	}

	return nil
}

// Helper: convert storage.Worker to WorkerStats
func (s *WorkerService) workerToStats(w *storage.Worker) *WorkerStats {
	// Convert capabilities
	capabilities := make([]string, len(w.Capabilities))
	for i, cap := range w.Capabilities {
		if capStr, ok := cap.(string); ok {
			capabilities[i] = capStr
		}
	}

	// Convert labels
	labels := make(map[string]string)
	for k, v := range w.Labels {
		if vStr, ok := v.(string); ok {
			labels[k] = vStr
		}
	}

	// Calculate usage percentages
	cpuUsage := 0.0
	if w.CPUCores > 0 {
		cpuUsage = float64(w.UsedCPUCores) / float64(w.CPUCores) * 100
	}
	memoryUsage := 0.0
	if w.MemoryMB > 0 {
		memoryUsage = float64(w.UsedMemoryMB) / float64(w.MemoryMB) * 100
	}

	// Calculate uptime
	uptime := time.Since(w.StartedAt).Round(time.Second).String()

	// Check if healthy (last seen within 1 minute)
	isHealthy := time.Since(w.LastSeen) < 1*time.Minute

	return &WorkerStats{
		ID:                 w.ID,
		Hostname:           w.Hostname,
		Address:            w.Address,
		Zone:               w.Zone,
		Status:             w.Status,
		Labels:             labels,
		Capabilities:       capabilities,
		CPUCores:           w.CPUCores,
		MemoryMB:           w.MemoryMB,
		DiskGB:             w.DiskGB,
		UsedCPUCores:       w.UsedCPUCores,
		UsedMemoryMB:       w.UsedMemoryMB,
		UsedDiskGB:         w.UsedDiskGB,
		CPUUsagePercent:    cpuUsage,
		MemoryUsagePercent: memoryUsage,
		VMCount:            w.VMCount,
		MaxVMs:             w.MaxVMs,
		StartedAt:          w.StartedAt,
		LastSeen:           w.LastSeen,
		Uptime:             uptime,
		IsHealthy:          isHealthy,
	}
}
