package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/aetherium/aetherium/pkg/discovery"
	"github.com/aetherium/aetherium/pkg/discovery/consul"
	"github.com/aetherium/aetherium/pkg/queue/asynq"
	"github.com/aetherium/aetherium/pkg/service"
	"github.com/aetherium/aetherium/pkg/storage/postgres"
	"github.com/aetherium/aetherium/pkg/vmm/firecracker"
	"github.com/aetherium/aetherium/pkg/worker"
	"github.com/google/uuid"
)

func main() {
	log.Println("Aetherium Worker starting...")

	// Initialize PostgreSQL store
	store, err := postgres.NewStore(postgres.Config{
		Host:         getEnv("POSTGRES_HOST", "localhost"),
		Port:         getEnvInt("POSTGRES_PORT", 5432),
		User:         getEnv("POSTGRES_USER", "aetherium"),
		Password:     getEnv("POSTGRES_PASSWORD", "aetherium"),
		Database:     getEnv("POSTGRES_DB", "aetherium"),
		SSLMode:      "disable",
		MaxOpenConns: 10,
		MaxIdleConns: 5,
	})
	if err != nil {
		log.Fatalf("Failed to initialize store: %v", err)
	}
	defer store.Close()

	// Initialize Redis queue
	queue, err := asynq.NewQueue(asynq.Config{
		RedisAddr: getEnv("REDIS_ADDR", "localhost:6379"),
		Queues: map[string]int{
			"critical": 6,
			"high":     5,
			"default":  3,
			"low":      1,
		},
	})
	if err != nil {
		log.Fatalf("Failed to initialize queue: %v", err)
	}

	// Initialize Firecracker orchestrator
	orchestrator, err := firecracker.NewFirecrackerOrchestrator(map[string]interface{}{
		"kernel_path":       getEnv("KERNEL_PATH", "/var/firecracker/vmlinux"),
		"rootfs_template":   getEnv("ROOTFS_TEMPLATE", "/var/firecracker/rootfs.ext4"),
		"socket_dir":        getEnv("SOCKET_DIR", "/tmp"),
		"default_vcpu":      getEnvInt("DEFAULT_VCPU", 1),
		"default_memory_mb": getEnvInt("DEFAULT_MEMORY_MB", 256),
	})
	if err != nil {
		log.Fatalf("Failed to initialize orchestrator: %v", err)
	}

	// Check if Consul is configured
	consulAddr := getEnv("CONSUL_ADDR", "")
	var w *worker.Worker

	if consulAddr != "" {
		// Distributed mode with Consul service discovery
		log.Println("Initializing worker in distributed mode...")

		// Initialize Consul registry
		consulConfig := &discovery.ConsulConfig{
			Address:     consulAddr,
			Datacenter:  getEnv("CONSUL_DATACENTER", "dc1"),
			Scheme:      getEnv("CONSUL_SCHEME", "http"),
			ServiceName: getEnv("CONSUL_SERVICE_NAME", "aetherium-worker"),
			Token:       getEnv("CONSUL_TOKEN", ""),
		}

		healthCheckConfig := discovery.DefaultHealthCheckConfig()

		consulRegistry, err := consul.NewConsulRegistry(consulConfig, healthCheckConfig)
		if err != nil {
			log.Fatalf("Failed to initialize Consul registry: %v", err)
		}

		// Get worker configuration
		workerID := getEnv("WORKER_ID", "")
		if workerID == "" {
			workerID = "worker-" + uuid.New().String()[:8]
		}

		hostname, _ := os.Hostname()
		workerConfig := &worker.Config{
			ID:       workerID,
			Hostname: hostname,
			Address:  getEnv("WORKER_ADDRESS", hostname+":8081"),
			Zone:     getEnv("WORKER_ZONE", "default"),
			Labels:   parseLabels(getEnv("WORKER_LABELS", "")),
			Capabilities: []string{
				getEnv("WORKER_CAPABILITY", "firecracker"),
			},
			CPUCores: getEnvInt("WORKER_CPU_CORES", runtime.NumCPU()),
			MemoryMB: int64(getEnvInt("WORKER_MEMORY_MB", 32768)),
			DiskGB:   int64(getEnvInt("WORKER_DISK_GB", 500)),
			MaxVMs:   getEnvInt("WORKER_MAX_VMS", 100),
			Registry: consulRegistry,
		}

		// Create worker with configuration
		w, err = worker.NewWithConfig(store, orchestrator, workerConfig)
		if err != nil {
			log.Fatalf("Failed to create worker: %v", err)
		}

		// Register worker with Consul and database
		ctx := context.Background()
		if err := w.Register(ctx); err != nil {
			log.Fatalf("Failed to register worker: %v", err)
		}

		log.Printf("✓ Worker registered: ID=%s, Zone=%s, Address=%s",
			workerConfig.ID, workerConfig.Zone, workerConfig.Address)

		// Start heartbeat
		heartbeatInterval := time.Duration(getEnvInt("HEARTBEAT_INTERVAL_SECONDS", 10)) * time.Second
		if err := w.StartHeartbeat(heartbeatInterval); err != nil {
			log.Fatalf("Failed to start heartbeat: %v", err)
		}

		log.Printf("✓ Heartbeat started (interval: %v)", heartbeatInterval)

	} else {
		// Legacy mode without service discovery
		log.Println("Initializing worker in legacy mode (no service discovery)")
		log.Println("  Set CONSUL_ADDR environment variable to enable distributed mode")
		w = worker.New(store, orchestrator)
	}

	// Register VM handlers
	if err := w.RegisterHandlers(queue); err != nil {
		log.Fatalf("Failed to register handlers: %v", err)
	}

	// Initialize WorkspaceService for secret decryption
	encryptionKey := getEnv("WORKSPACE_ENCRYPTION_KEY", "")
	workspaceService, err := service.NewWorkspaceService(queue, store, encryptionKey)
	if err != nil {
		log.Printf("Warning: Failed to initialize workspace service: %v", err)
		log.Println("  Workspace features will be limited")
	} else {
		// Set workspace service on worker for secret decryption
		w.SetWorkspaceService(workspaceService)
		log.Println("  Registered handlers: workspace:create, workspace:delete, prompt:execute")
	}

	log.Println("✓ Worker initialized successfully")
	log.Println("  Registered handlers: vm:create, vm:execute, vm:delete")
	log.Println("  Listening for tasks on Redis queue...")

	// Start idle VM cleanup worker (checks for idle workspaces and destroys VMs after timeout)
	idleCleanupCtx, idleCleanupCancel := context.WithCancel(context.Background())
	idleCheckInterval := time.Duration(getEnvInt("IDLE_CHECK_INTERVAL_SECONDS", 60)) * time.Second
	w.StartIdleCleanup(idleCleanupCtx, idleCheckInterval)
	log.Printf("  Started idle VM cleanup worker (check interval: %v)", idleCheckInterval)

	// Start processing tasks
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := queue.Start(ctx); err != nil {
		log.Fatalf("Failed to start queue: %v", err)
	}

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down worker...")
	cancel()

	// Stop idle cleanup worker
	idleCleanupCancel()
	log.Println("  Stopped idle VM cleanup worker")

	// Deregister worker from Consul
	if consulAddr != "" {
		deregCtx, deregCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer deregCancel()

		if err := w.Deregister(deregCtx); err != nil {
			log.Printf("Warning: Failed to deregister worker: %v", err)
		} else {
			log.Println("✓ Worker deregistered from Consul")
		}
	}

	// Cleanup all VMs (this will delete per-VM rootfs files)
	log.Println("Cleaning up VMs...")
	cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cleanupCancel()

	vms, err := orchestrator.ListVMs(cleanupCtx)
	if err != nil {
		log.Printf("Warning: Failed to list VMs during cleanup: %v", err)
	} else if len(vms) > 0 {
		log.Printf("  Found %d VMs to cleanup", len(vms))
		for _, vm := range vms {
			if err := orchestrator.DeleteVM(cleanupCtx, vm.ID); err != nil {
				log.Printf("  Warning: Failed to delete VM %s: %v", vm.ID, err)
			} else {
				log.Printf("  ✓ Deleted VM %s and its rootfs", vm.ID)
			}
		}
		log.Println("✓ VM cleanup complete")
	} else {
		log.Println("  No VMs to cleanup")
	}

	// Stop queue
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer stopCancel()
	queue.Stop(stopCtx)

	log.Println("✓ Worker stopped gracefully")
}

// Helper functions

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return fallback
}

func parseLabels(labelsStr string) map[string]string {
	labels := make(map[string]string)
	if labelsStr == "" {
		return labels
	}

	// Simple parsing: env=prod,tier=compute
	// For production, use proper parsing or JSON
	pairs := splitString(labelsStr, ',')
	for _, pair := range pairs {
		kv := splitString(pair, '=')
		if len(kv) == 2 {
			labels[kv[0]] = kv[1]
		}
	}

	return labels
}

func splitString(s string, sep rune) []string {
	var result []string
	var current string
	for _, r := range s {
		if r == sep {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(r)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}
