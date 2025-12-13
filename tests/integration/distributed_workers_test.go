package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/aetherium/aetherium/services/gateway/pkg/discovery"
	"github.com/aetherium/aetherium/services/core/pkg/service"
	"github.com/aetherium/aetherium/services/core/pkg/storage"
	"github.com/aetherium/aetherium/services/core/pkg/storage/postgres"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test database configuration
func getTestDBConfig() postgres.Config {
	return postgres.Config{
		Host:         getEnv("TEST_POSTGRES_HOST", "localhost"),
		Port:         5432,
		User:         getEnv("TEST_POSTGRES_USER", "aetherium"),
		Password:     getEnv("TEST_POSTGRES_PASSWORD", "aetherium"),
		Database:     getEnv("TEST_POSTGRES_DB", "aetherium_test"),
		SSLMode:      "disable",
		MaxOpenConns: 10,
		MaxIdleConns: 5,
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// setupTestDB creates a test database store and runs migrations
func setupTestDB(t *testing.T) storage.Store {
	config := getTestDBConfig()

	store, err := postgres.NewStore(config)
	require.NoError(t, err, "Failed to create test database connection")

	// Run migrations
	err = store.RunMigrations("../../migrations")
	if err != nil {
		t.Logf("Warning: Migration error (may be expected if already migrated): %v", err)
	}

	return store
}

// cleanupTestDB cleans up test data
func cleanupTestDB(t *testing.T, store storage.Store) {
	ctx := context.Background()

	// Get all workers
	workers, err := store.Workers().List(ctx, nil)
	if err != nil {
		t.Logf("Warning: Failed to list workers for cleanup: %v", err)
		return
	}

	// Delete all test workers
	for _, w := range workers {
		if err := store.Workers().Delete(ctx, w.ID); err != nil {
			t.Logf("Warning: Failed to delete worker %s: %v", w.ID, err)
		}
	}

	// Delete all test VMs
	vms, err := store.VMs().List(ctx, nil)
	if err != nil {
		t.Logf("Warning: Failed to list VMs for cleanup: %v", err)
		return
	}

	for _, vm := range vms {
		if err := store.VMs().Delete(ctx, vm.ID); err != nil {
			t.Logf("Warning: Failed to delete VM %s: %v", vm.ID, err)
		}
	}

	store.Close()
}

// TestWorkerDatabaseOperations tests worker CRUD operations
func TestWorkerDatabaseOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	store := setupTestDB(t)
	defer cleanupTestDB(t, store)

	ctx := context.Background()

	t.Run("CreateWorker", func(t *testing.T) {
		worker := &storage.Worker{
			ID:       "test-worker-01",
			Hostname: "test-node1.example.com",
			Address:  "192.168.1.100:8081",
			Status:   string(discovery.WorkerStatusActive),
			LastSeen: time.Now(),
			StartedAt: time.Now(),
			Zone:     "test-zone-1a",
			Labels: map[string]interface{}{
				"env": "test",
				"tier": "compute",
			},
			Capabilities: []interface{}{"firecracker", "docker"},
			CPUCores:     16,
			MemoryMB:     32768,
			DiskGB:       500,
			UsedCPUCores: 0,
			UsedMemoryMB: 0,
			UsedDiskGB:   0,
			VMCount:      0,
			MaxVMs:       100,
			Metadata:     make(map[string]interface{}),
		}

		err := store.Workers().Create(ctx, worker)
		require.NoError(t, err, "Failed to create worker")
	})

	t.Run("GetWorker", func(t *testing.T) {
		worker, err := store.Workers().Get(ctx, "test-worker-01")
		require.NoError(t, err, "Failed to get worker")
		assert.Equal(t, "test-worker-01", worker.ID)
		assert.Equal(t, "test-node1.example.com", worker.Hostname)
		assert.Equal(t, "test-zone-1a", worker.Zone)
		assert.Equal(t, int64(32768), worker.MemoryMB)
	})

	t.Run("ListWorkers", func(t *testing.T) {
		workers, err := store.Workers().List(ctx, nil)
		require.NoError(t, err, "Failed to list workers")
		assert.GreaterOrEqual(t, len(workers), 1, "Should have at least one worker")
	})

	t.Run("ListWorkersByZone", func(t *testing.T) {
		workers, err := store.Workers().ListByZone(ctx, "test-zone-1a")
		require.NoError(t, err, "Failed to list workers by zone")
		assert.GreaterOrEqual(t, len(workers), 1, "Should find workers in test-zone-1a")
	})

	t.Run("UpdateWorkerResources", func(t *testing.T) {
		resources := map[string]interface{}{
			"used_cpu_cores": 4,
			"used_memory_mb": int64(8192),
			"vm_count":       2,
		}

		err := store.Workers().UpdateResources(ctx, "test-worker-01", resources)
		require.NoError(t, err, "Failed to update worker resources")

		// Verify update
		worker, err := store.Workers().Get(ctx, "test-worker-01")
		require.NoError(t, err, "Failed to get worker after update")
		assert.Equal(t, 4, worker.UsedCPUCores)
		assert.Equal(t, int64(8192), worker.UsedMemoryMB)
		assert.Equal(t, 2, worker.VMCount)
	})

	t.Run("UpdateWorkerStatus", func(t *testing.T) {
		err := store.Workers().UpdateStatus(ctx, "test-worker-01", string(discovery.WorkerStatusDraining))
		require.NoError(t, err, "Failed to update worker status")

		// Verify update
		worker, err := store.Workers().Get(ctx, "test-worker-01")
		require.NoError(t, err, "Failed to get worker after status update")
		assert.Equal(t, string(discovery.WorkerStatusDraining), worker.Status)
	})

	t.Run("UpdateLastSeen", func(t *testing.T) {
		beforeUpdate := time.Now()
		time.Sleep(100 * time.Millisecond)

		err := store.Workers().UpdateLastSeen(ctx, "test-worker-01")
		require.NoError(t, err, "Failed to update last_seen")

		worker, err := store.Workers().Get(ctx, "test-worker-01")
		require.NoError(t, err, "Failed to get worker after last_seen update")
		assert.True(t, worker.LastSeen.After(beforeUpdate), "Last seen should be updated")
	})

	t.Run("DeleteWorker", func(t *testing.T) {
		err := store.Workers().Delete(ctx, "test-worker-01")
		require.NoError(t, err, "Failed to delete worker")

		// Verify deletion
		_, err = store.Workers().Get(ctx, "test-worker-01")
		assert.Error(t, err, "Should not find deleted worker")
	})
}

// TestWorkerMetrics tests worker metrics storage
func TestWorkerMetrics(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	store := setupTestDB(t)
	defer cleanupTestDB(t, store)

	ctx := context.Background()

	// Create a worker first
	worker := &storage.Worker{
		ID:       "metrics-worker-01",
		Hostname: "metrics-node1.example.com",
		Address:  "192.168.1.101:8081",
		Status:   string(discovery.WorkerStatusActive),
		LastSeen: time.Now(),
		StartedAt: time.Now(),
		Zone:     "test-zone-1a",
		Labels:   make(map[string]interface{}),
		Capabilities: []interface{}{"firecracker"},
		CPUCores:     8,
		MemoryMB:     16384,
		DiskGB:       250,
		MaxVMs:       50,
		Metadata:     make(map[string]interface{}),
	}

	err := store.Workers().Create(ctx, worker)
	require.NoError(t, err, "Failed to create worker for metrics test")

	t.Run("CreateMetric", func(t *testing.T) {
		metric := &storage.WorkerMetric{
			WorkerID:       "metrics-worker-01",
			Timestamp:      time.Now(),
			CPUUsage:       45.5,
			MemoryUsage:    60.0,
			DiskUsage:      30.0,
			VMCount:        5,
			TasksProcessed: 100,
			Metadata:       make(map[string]interface{}),
		}

		err := store.WorkerMetrics().Create(ctx, metric)
		require.NoError(t, err, "Failed to create worker metric")
		assert.NotEqual(t, uuid.Nil, metric.ID, "Metric ID should be generated")
	})

	t.Run("ListMetricsByWorker", func(t *testing.T) {
		// Create multiple metrics
		for i := 0; i < 5; i++ {
			metric := &storage.WorkerMetric{
				WorkerID:       "metrics-worker-01",
				Timestamp:      time.Now().Add(time.Duration(i) * time.Minute),
				CPUUsage:       float64(40 + i*5),
				MemoryUsage:    float64(50 + i*2),
				DiskUsage:      30.0,
				VMCount:        i + 1,
				TasksProcessed: (i + 1) * 20,
				Metadata:       make(map[string]interface{}),
			}
			err := store.WorkerMetrics().Create(ctx, metric)
			require.NoError(t, err, "Failed to create metric %d", i)
		}

		// List metrics
		metrics, err := store.WorkerMetrics().ListByWorker(ctx, "metrics-worker-01", 10)
		require.NoError(t, err, "Failed to list metrics")
		assert.GreaterOrEqual(t, len(metrics), 5, "Should have at least 5 metrics")
	})

	t.Run("ListMetricsInTimeRange", func(t *testing.T) {
		start := time.Now().Add(-1 * time.Hour)
		end := time.Now().Add(1 * time.Hour)

		metrics, err := store.WorkerMetrics().ListByWorkerInTimeRange(ctx, "metrics-worker-01", start, end)
		require.NoError(t, err, "Failed to list metrics in time range")
		assert.GreaterOrEqual(t, len(metrics), 1, "Should find metrics in time range")
	})
}

// TestWorkerService tests the worker service layer
func TestWorkerService(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	store := setupTestDB(t)
	defer cleanupTestDB(t, store)

	ctx := context.Background()

	// Create worker service
	workerService := service.NewWorkerService(store, nil)

	// Create test workers
	workers := []storage.Worker{
		{
			ID:       "service-worker-01",
			Hostname: "node1.example.com",
			Address:  "192.168.1.10:8081",
			Status:   string(discovery.WorkerStatusActive),
			LastSeen: time.Now(),
			StartedAt: time.Now(),
			Zone:     "us-west-1a",
			Labels:   map[string]interface{}{"env": "prod", "tier": "compute"},
			Capabilities: []interface{}{"firecracker", "docker"},
			CPUCores:     16,
			MemoryMB:     32768,
			DiskGB:       500,
			UsedCPUCores: 4,
			UsedMemoryMB: 8192,
			VMCount:      3,
			MaxVMs:       100,
			Metadata:     make(map[string]interface{}),
		},
		{
			ID:       "service-worker-02",
			Hostname: "node2.example.com",
			Address:  "192.168.1.11:8081",
			Status:   string(discovery.WorkerStatusActive),
			LastSeen: time.Now(),
			StartedAt: time.Now().Add(-2 * time.Hour),
			Zone:     "us-west-1b",
			Labels:   map[string]interface{}{"env": "prod", "tier": "gpu"},
			Capabilities: []interface{}{"firecracker"},
			CPUCores:     32,
			MemoryMB:     65536,
			DiskGB:       1000,
			UsedCPUCores: 8,
			UsedMemoryMB: 16384,
			VMCount:      5,
			MaxVMs:       200,
			Metadata:     make(map[string]interface{}),
		},
	}

	for _, w := range workers {
		err := store.Workers().Create(ctx, &w)
		require.NoError(t, err, "Failed to create worker %s", w.ID)
	}

	t.Run("ListWorkers", func(t *testing.T) {
		stats, err := workerService.ListWorkers(ctx)
		require.NoError(t, err, "Failed to list workers")
		assert.GreaterOrEqual(t, len(stats), 2, "Should have at least 2 workers")

		// Verify stats calculation
		for _, stat := range stats {
			assert.NotEmpty(t, stat.ID)
			assert.NotEmpty(t, stat.Hostname)
			assert.True(t, stat.CPUUsagePercent >= 0 && stat.CPUUsagePercent <= 100)
			assert.True(t, stat.MemoryUsagePercent >= 0 && stat.MemoryUsagePercent <= 100)
			assert.NotEmpty(t, stat.Uptime)
		}
	})

	t.Run("GetWorker", func(t *testing.T) {
		stat, err := workerService.GetWorker(ctx, "service-worker-01")
		require.NoError(t, err, "Failed to get worker")
		assert.Equal(t, "service-worker-01", stat.ID)
		assert.Equal(t, "node1.example.com", stat.Hostname)
		assert.Equal(t, 25.0, stat.CPUUsagePercent) // 4/16 * 100
		assert.Equal(t, 3, stat.VMCount)
	})

	t.Run("ListActiveWorkers", func(t *testing.T) {
		stats, err := workerService.ListActiveWorkers(ctx)
		require.NoError(t, err, "Failed to list active workers")
		assert.GreaterOrEqual(t, len(stats), 2, "Should have at least 2 active workers")

		for _, stat := range stats {
			assert.Equal(t, string(discovery.WorkerStatusActive), stat.Status)
		}
	})

	t.Run("ListWorkersByZone", func(t *testing.T) {
		stats, err := workerService.ListWorkersByZone(ctx, "us-west-1a")
		require.NoError(t, err, "Failed to list workers by zone")
		assert.GreaterOrEqual(t, len(stats), 1, "Should have at least 1 worker in us-west-1a")

		for _, stat := range stats {
			assert.Equal(t, "us-west-1a", stat.Zone)
		}
	})

	t.Run("GetClusterStats", func(t *testing.T) {
		stats, err := workerService.GetClusterStats(ctx)
		require.NoError(t, err, "Failed to get cluster stats")

		assert.GreaterOrEqual(t, stats.TotalWorkers, 2)
		assert.Equal(t, stats.TotalWorkers, stats.ActiveWorkers)
		assert.Equal(t, 48, stats.TotalCPUCores) // 16 + 32
		assert.Equal(t, int64(98304), stats.TotalMemoryMB) // 32768 + 65536
		assert.Equal(t, 12, stats.UsedCPUCores) // 4 + 8
		assert.Equal(t, int64(24576), stats.UsedMemoryMB) // 8192 + 16384
		assert.Equal(t, 8, stats.TotalVMs) // 3 + 5
		assert.Equal(t, 2, len(stats.Zones))
		assert.True(t, stats.ClusterCPUUsagePercent > 0)
	})

	t.Run("DrainWorker", func(t *testing.T) {
		err := workerService.DrainWorker(ctx, "service-worker-01")
		require.NoError(t, err, "Failed to drain worker")

		// Verify status
		worker, err := store.Workers().Get(ctx, "service-worker-01")
		require.NoError(t, err, "Failed to get worker")
		assert.Equal(t, string(discovery.WorkerStatusDraining), worker.Status)
	})

	t.Run("ActivateWorker", func(t *testing.T) {
		err := workerService.ActivateWorker(ctx, "service-worker-01")
		require.NoError(t, err, "Failed to activate worker")

		// Verify status
		worker, err := store.Workers().Get(ctx, "service-worker-01")
		require.NoError(t, err, "Failed to get worker")
		assert.Equal(t, string(discovery.WorkerStatusActive), worker.Status)
	})
}

// TestVMWorkerAssignment tests VM-to-worker assignment
func TestVMWorkerAssignment(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	store := setupTestDB(t)
	defer cleanupTestDB(t, store)

	ctx := context.Background()

	// Create a worker
	worker := &storage.Worker{
		ID:       "vm-worker-01",
		Hostname: "vm-node1.example.com",
		Address:  "192.168.1.20:8081",
		Status:   string(discovery.WorkerStatusActive),
		LastSeen: time.Now(),
		StartedAt: time.Now(),
		Zone:     "test-zone",
		Labels:   make(map[string]interface{}),
		Capabilities: []interface{}{"firecracker"},
		CPUCores:     8,
		MemoryMB:     16384,
		MaxVMs:       50,
		Metadata:     make(map[string]interface{}),
	}

	err := store.Workers().Create(ctx, worker)
	require.NoError(t, err, "Failed to create worker")

	t.Run("CreateVMWithWorkerID", func(t *testing.T) {
		vcpus := 2
		memory := 2048
		workerID := "vm-worker-01"
		kernel := "/var/firecracker/vmlinux"
		rootfs := "/var/firecracker/rootfs.ext4"

		vm := &storage.VM{
			ID:           uuid.New(),
			Name:         "test-vm-01",
			Orchestrator: "firecracker",
			Status:       "running",
			KernelPath:   &kernel,
			RootFSPath:   &rootfs,
			VCPUCount:    &vcpus,
			MemoryMB:     &memory,
			WorkerID:     &workerID,
			CreatedAt:    time.Now(),
			Metadata:     make(map[string]interface{}),
		}

		err := store.VMs().Create(ctx, vm)
		require.NoError(t, err, "Failed to create VM with worker_id")
	})

	t.Run("ListVMsByWorker", func(t *testing.T) {
		vms, err := store.VMs().List(ctx, map[string]interface{}{
			"worker_id": "vm-worker-01",
		})
		require.NoError(t, err, "Failed to list VMs by worker")
		assert.GreaterOrEqual(t, len(vms), 1, "Should have at least 1 VM assigned to worker")

		for _, vm := range vms {
			assert.NotNil(t, vm.WorkerID)
			assert.Equal(t, "vm-worker-01", *vm.WorkerID)
		}
	})

	t.Run("GetWorkerVMs", func(t *testing.T) {
		workerService := service.NewWorkerService(store, nil)

		vms, err := workerService.GetWorkerVMs(ctx, "vm-worker-01")
		require.NoError(t, err, "Failed to get worker VMs")
		assert.GreaterOrEqual(t, len(vms), 1, "Should have at least 1 VM")

		for _, vm := range vms {
			assert.NotEmpty(t, vm.ID)
			assert.NotEmpty(t, vm.Name)
		}
	})

	t.Run("GetVMDistribution", func(t *testing.T) {
		workerService := service.NewWorkerService(store, nil)

		distribution, err := workerService.GetVMDistribution(ctx)
		require.NoError(t, err, "Failed to get VM distribution")
		assert.GreaterOrEqual(t, len(distribution), 1, "Should have at least 1 worker in distribution")

		found := false
		for _, d := range distribution {
			if d.WorkerID == "vm-worker-01" {
				found = true
				assert.GreaterOrEqual(t, d.VMCount, 1)
				assert.GreaterOrEqual(t, len(d.VMs), 1)
			}
		}
		assert.True(t, found, "Should find vm-worker-01 in distribution")
	})
}

// TestMultiWorkerScenario tests a realistic multi-worker cluster scenario
func TestMultiWorkerScenario(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	store := setupTestDB(t)
	defer cleanupTestDB(t, store)

	ctx := context.Background()
	workerService := service.NewWorkerService(store, nil)

	t.Run("SetupThreeWorkerCluster", func(t *testing.T) {
		// Create 3 workers in different zones
		workers := []storage.Worker{
			{
				ID: "cluster-worker-01", Hostname: "node1.us-west-1a.example.com",
				Address: "192.168.1.10:8081", Status: string(discovery.WorkerStatusActive),
				LastSeen: time.Now(), StartedAt: time.Now(),
				Zone: "us-west-1a", Labels: map[string]interface{}{"env": "prod"},
				Capabilities: []interface{}{"firecracker"}, CPUCores: 16, MemoryMB: 32768,
				DiskGB: 500, MaxVMs: 100, Metadata: make(map[string]interface{}),
			},
			{
				ID: "cluster-worker-02", Hostname: "node2.us-west-1b.example.com",
				Address: "192.168.1.11:8081", Status: string(discovery.WorkerStatusActive),
				LastSeen: time.Now(), StartedAt: time.Now(),
				Zone: "us-west-1b", Labels: map[string]interface{}{"env": "prod"},
				Capabilities: []interface{}{"firecracker"}, CPUCores: 16, MemoryMB: 32768,
				DiskGB: 500, MaxVMs: 100, Metadata: make(map[string]interface{}),
			},
			{
				ID: "cluster-worker-03", Hostname: "node3.us-east-1a.example.com",
				Address: "192.168.2.10:8081", Status: string(discovery.WorkerStatusActive),
				LastSeen: time.Now(), StartedAt: time.Now(),
				Zone: "us-east-1a", Labels: map[string]interface{}{"env": "prod"},
				Capabilities: []interface{}{"firecracker"}, CPUCores: 32, MemoryMB: 65536,
				DiskGB: 1000, MaxVMs: 200, Metadata: make(map[string]interface{}),
			},
		}

		for _, w := range workers {
			err := store.Workers().Create(ctx, &w)
			require.NoError(t, err, "Failed to create worker %s", w.ID)
		}

		// Verify cluster stats
		stats, err := workerService.GetClusterStats(ctx)
		require.NoError(t, err, "Failed to get cluster stats")
		assert.Equal(t, 3, stats.TotalWorkers)
		assert.Equal(t, 64, stats.TotalCPUCores) // 16+16+32
		assert.Equal(t, int64(131072), stats.TotalMemoryMB) // 32768+32768+65536
		assert.Equal(t, 3, len(stats.Zones))
	})

	t.Run("SimulateVMCreationAcrossWorkers", func(t *testing.T) {
		// Create VMs on different workers
		vmConfigs := []struct {
			name     string
			workerID string
			vcpus    int
			memoryMB int
		}{
			{"vm-1", "cluster-worker-01", 2, 2048},
			{"vm-2", "cluster-worker-01", 1, 1024},
			{"vm-3", "cluster-worker-02", 4, 4096},
			{"vm-4", "cluster-worker-02", 2, 2048},
			{"vm-5", "cluster-worker-03", 8, 8192},
		}

		kernel := "/var/firecracker/vmlinux"
		rootfs := "/var/firecracker/rootfs.ext4"

		for _, cfg := range vmConfigs {
			vm := &storage.VM{
				ID:           uuid.New(),
				Name:         cfg.name,
				Orchestrator: "firecracker",
				Status:       "running",
				KernelPath:   &kernel,
				RootFSPath:   &rootfs,
				VCPUCount:    &cfg.vcpus,
				MemoryMB:     &cfg.memoryMB,
				WorkerID:     &cfg.workerID,
				CreatedAt:    time.Now(),
				Metadata:     make(map[string]interface{}),
			}

			err := store.VMs().Create(ctx, vm)
			require.NoError(t, err, "Failed to create VM %s", cfg.name)
		}

		// Update worker resources
		updateResources := func(workerID string, usedCPU int, usedMem int64, vmCount int) {
			err := store.Workers().UpdateResources(ctx, workerID, map[string]interface{}{
				"used_cpu_cores": usedCPU,
				"used_memory_mb": usedMem,
				"vm_count":       vmCount,
			})
			require.NoError(t, err, "Failed to update resources for %s", workerID)
		}

		updateResources("cluster-worker-01", 3, 3072, 2)
		updateResources("cluster-worker-02", 6, 6144, 2)
		updateResources("cluster-worker-03", 8, 8192, 1)
	})

	t.Run("VerifyClusterResourceUsage", func(t *testing.T) {
		stats, err := workerService.GetClusterStats(ctx)
		require.NoError(t, err, "Failed to get cluster stats")

		assert.Equal(t, 17, stats.UsedCPUCores) // 3+6+8
		assert.Equal(t, int64(17408), stats.UsedMemoryMB) // 3072+6144+8192
		assert.Equal(t, 5, stats.TotalVMs)
		assert.True(t, stats.ClusterCPUUsagePercent > 0)
		assert.True(t, stats.ClusterMemoryUsagePercent > 0)
	})

	t.Run("VerifyVMDistribution", func(t *testing.T) {
		distribution, err := workerService.GetVMDistribution(ctx)
		require.NoError(t, err, "Failed to get VM distribution")

		assert.Equal(t, 3, len(distribution))

		// Check each worker's VM count
		vmCounts := make(map[string]int)
		for _, d := range distribution {
			vmCounts[d.WorkerID] = d.VMCount
		}

		assert.Equal(t, 2, vmCounts["cluster-worker-01"])
		assert.Equal(t, 2, vmCounts["cluster-worker-02"])
		assert.Equal(t, 1, vmCounts["cluster-worker-03"])
	})

	t.Run("DrainWorkerInCluster", func(t *testing.T) {
		// Drain worker-01
		err := workerService.DrainWorker(ctx, "cluster-worker-01")
		require.NoError(t, err, "Failed to drain worker")

		// Verify cluster stats reflect draining worker
		stats, err := workerService.GetClusterStats(ctx)
		require.NoError(t, err, "Failed to get cluster stats")
		assert.Equal(t, 2, stats.ActiveWorkers)
		assert.Equal(t, 1, stats.DrainingWorkers)
	})
}
