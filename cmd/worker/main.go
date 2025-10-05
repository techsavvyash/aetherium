package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aetherium/aetherium/pkg/queue/asynq"
	"github.com/aetherium/aetherium/pkg/storage/postgres"
	"github.com/aetherium/aetherium/pkg/vmm/firecracker"
	"github.com/aetherium/aetherium/pkg/worker"
)

func main() {
	log.Println("Aetherium Worker starting...")

	// Initialize PostgreSQL store
	store, err := postgres.NewStore(postgres.Config{
		Host:         "localhost",
		Port:         5432,
		User:         "aetherium",
		Password:     "aetherium",
		Database:     "aetherium",
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
		RedisAddr: "localhost:6379",
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
		"kernel_path":       "/var/firecracker/vmlinux",
		"rootfs_template":   "/var/firecracker/rootfs.ext4",
		"socket_dir":        "/tmp",
		"default_vcpu":      1,
		"default_memory_mb": 256,
	})
	if err != nil {
		log.Fatalf("Failed to initialize orchestrator: %v", err)
	}

	// Create worker
	w := worker.New(store, orchestrator)

	// Register handlers
	if err := w.RegisterHandlers(queue); err != nil {
		log.Fatalf("Failed to register handlers: %v", err)
	}

	log.Println("Worker initialized successfully")
	log.Println("Registered handlers: vm:create, vm:execute, vm:delete")
	log.Println("Listening for tasks on Redis queue...")

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

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer stopCancel()
	queue.Stop(stopCtx)

	log.Println("Worker stopped")
}
