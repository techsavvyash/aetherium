package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/aetherium/aetherium/pkg/api"
	"github.com/aetherium/aetherium/pkg/discovery"
	"github.com/aetherium/aetherium/pkg/discovery/consul"
	"github.com/aetherium/aetherium/pkg/events/redis"
	"github.com/aetherium/aetherium/pkg/integrations"
	githubIntegration "github.com/aetherium/aetherium/pkg/integrations/github"
	"github.com/aetherium/aetherium/pkg/integrations/slack"
	"github.com/aetherium/aetherium/pkg/logging/loki"
	"github.com/aetherium/aetherium/pkg/queue/asynq"
	"github.com/aetherium/aetherium/pkg/service"
	"github.com/aetherium/aetherium/pkg/storage"
	"github.com/aetherium/aetherium/pkg/storage/postgres"
	"github.com/aetherium/aetherium/pkg/vmm"
	"github.com/aetherium/aetherium/pkg/vmm/firecracker"
	"github.com/aetherium/aetherium/pkg/websocket"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
)

type Server struct {
	store            *postgres.Store
	taskService      *service.TaskService
	workerService    *service.WorkerService
	workspaceService *service.WorkspaceService
	integrations     *integrations.Registry
	logger           *loki.LokiLogger
	eventBus         *redis.RedisEventBus
	orchestrator     vmm.VMOrchestrator
	sessionManager   *websocket.SessionManager
}

func main() {
	log.Println("Aetherium API Gateway starting...")

	// Initialize PostgreSQL store
	store, err := postgres.NewStore(postgres.Config{
		Host:         getEnv("POSTGRES_HOST", "localhost"),
		Port:         getEnvInt("POSTGRES_PORT", 5432),
		User:         getEnv("POSTGRES_USER", "aetherium"),
		Password:     getEnv("POSTGRES_PASSWORD", "aetherium"),
		Database:     getEnv("POSTGRES_DB", "aetherium"),
		SSLMode:      "disable",
		MaxOpenConns: 25,
		MaxIdleConns: 5,
	})
	if err != nil {
		log.Fatalf("Failed to initialize store: %v", err)
	}
	defer store.Close()

	// Initialize Redis queue
	queue, err := asynq.NewQueue(asynq.Config{
		RedisAddr: getEnv("REDIS_ADDR", "localhost:6379"),
	})
	if err != nil {
		log.Fatalf("Failed to initialize queue: %v", err)
	}

	// Initialize Loki logger (optional)
	var logger *loki.LokiLogger
	if lokiURL := getEnv("LOKI_URL", ""); lokiURL != "" {
		logger, err = loki.NewLokiLogger(&loki.Config{
			URL:           lokiURL,
			BatchSize:     100,
			BatchInterval: 5 * time.Second,
			Labels: map[string]string{
				"service":   "aetherium-api",
				"component": "api-gateway",
			},
		})
		if err != nil {
			log.Printf("Warning: Failed to initialize Loki logger: %v", err)
		} else {
			defer logger.Close()
		}
	}

	// Initialize Redis event bus (optional)
	var eventBus *redis.RedisEventBus
	if redisAddr := getEnv("REDIS_ADDR", ""); redisAddr != "" {
		eventBus, err = redis.NewRedisEventBus(&redis.Config{
			Addr: redisAddr,
		})
		if err != nil {
			log.Printf("Warning: Failed to initialize event bus: %v", err)
		} else {
			defer eventBus.Close()
		}
	}

	// Initialize integration registry
	registry := integrations.NewRegistry()

	// Register GitHub integration
	if githubToken := getEnv("GITHUB_TOKEN", ""); githubToken != "" {
		gh := githubIntegration.NewGitHubIntegration()
		if err := gh.Initialize(context.Background(), integrations.Config{
			Options: map[string]interface{}{
				"token":          githubToken,
				"webhook_secret": getEnv("GITHUB_WEBHOOK_SECRET", ""),
			},
		}); err != nil {
			log.Printf("Warning: Failed to initialize GitHub integration: %v", err)
		} else {
			registry.Register(gh)
			log.Println("✓ GitHub integration registered")
		}
	}

	// Register Slack integration
	if slackToken := getEnv("SLACK_BOT_TOKEN", ""); slackToken != "" {
		slackInt := slack.NewSlackIntegration()
		if err := slackInt.Initialize(context.Background(), integrations.Config{
			Options: map[string]interface{}{
				"bot_token":      slackToken,
				"signing_secret": getEnv("SLACK_SIGNING_SECRET", ""),
			},
		}); err != nil {
			log.Printf("Warning: Failed to initialize Slack integration: %v", err)
		} else {
			registry.Register(slackInt)
			log.Println("✓ Slack integration registered")
		}
	}

	// Create task service
	taskService := service.NewTaskService(queue, store)

	// Create workspace service
	encryptionKey := getEnv("WORKSPACE_ENCRYPTION_KEY", "")
	workspaceService, err := service.NewWorkspaceService(queue, store, encryptionKey)
	if err != nil {
		log.Fatalf("Failed to initialize workspace service: %v", err)
	}
	log.Println("✓ Workspace service initialized")

	// Initialize Firecracker orchestrator for WebSocket streaming
	var orchestrator vmm.VMOrchestrator
	firecrackerConfig := map[string]interface{}{
		"kernel_path":       getEnv("FIRECRACKER_KERNEL", "/var/firecracker/vmlinux.bin"),
		"rootfs_template":   getEnv("FIRECRACKER_ROOTFS", "/var/firecracker/rootfs.ext4"),
		"socket_dir":        getEnv("FIRECRACKER_SOCKET_DIR", "/tmp"),
		"default_vcpu":      getEnvInt("FIRECRACKER_DEFAULT_VCPU", 1),
		"default_memory_mb": getEnvInt("FIRECRACKER_DEFAULT_MEMORY_MB", 512),
	}

	orchestrator, err = firecracker.NewFirecrackerOrchestrator(firecrackerConfig)
	if err != nil {
		log.Printf("Warning: Failed to initialize Firecracker orchestrator: %v", err)
		log.Println("  WebSocket streaming will not be available")
		log.Println("  Set FIRECRACKER_KERNEL and FIRECRACKER_ROOTFS environment variables")
		orchestrator = nil
	} else {
		log.Println("✓ Firecracker orchestrator initialized for WebSocket streaming")
	}

	// Initialize WebSocket session manager
	var sessionManager *websocket.SessionManager
	if orchestrator != nil {
		sessionManager = websocket.NewSessionManager(store, orchestrator)
		log.Println("✓ WebSocket session manager initialized")
	}

	// Initialize service discovery (optional)
	var workerService *service.WorkerService
	var consulRegistry discovery.ServiceRegistry

	consulAddr := getEnv("CONSUL_ADDR", "")
	if consulAddr != "" {
		// Initialize Consul registry
		consulConfig := &discovery.ConsulConfig{
			Address:     consulAddr,
			Datacenter:  getEnv("CONSUL_DATACENTER", "dc1"),
			Scheme:      getEnv("CONSUL_SCHEME", "http"),
			ServiceName: getEnv("CONSUL_SERVICE_NAME", "aetherium-worker"),
			Token:       getEnv("CONSUL_TOKEN", ""),
		}

		healthCheckConfig := discovery.DefaultHealthCheckConfig()

		reg, err := consul.NewConsulRegistry(consulConfig, healthCheckConfig)
		if err != nil {
			log.Printf("Warning: Failed to initialize Consul registry: %v", err)
			log.Println("Worker service will run without service discovery")
			workerService = service.NewWorkerService(store, nil)
		} else {
			consulRegistry = reg
			workerService = service.NewWorkerService(store, consulRegistry)
			log.Printf("✓ Consul registry initialized (address: %s, datacenter: %s)",
				consulAddr, consulConfig.Datacenter)
		}
	} else {
		workerService = service.NewWorkerService(store, nil)
		log.Println("Worker service initialized (no Consul configured)")
		log.Println("  Set CONSUL_ADDR environment variable to enable service discovery")
	}

	// Create server
	srv := &Server{
		store:            store,
		taskService:      taskService,
		workerService:    workerService,
		workspaceService: workspaceService,
		integrations:     registry,
		logger:           logger,
		eventBus:         eventBus,
		orchestrator:     orchestrator,
		sessionManager:   sessionManager,
	}

	// Setup router
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// CORS
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Static UI
	r.Get("/ui", srv.serveUI)
	r.Get("/", srv.serveUI) // Redirect root to UI

	// Routes
	r.Route("/api/v1", func(r chi.Router) {
		// Smart Execute - Intelligent VM selection
		r.Post("/smart-execute", srv.smartExecute)

		// VMs
		r.Post("/vms", srv.createVM)
		r.Get("/vms", srv.listVMs)
		r.Get("/vms/{id}", srv.getVM)
		r.Delete("/vms/{id}", srv.deleteVM)
		r.Post("/vms/{id}/execute", srv.executeCommand)
		r.Get("/vms/{id}/executions", srv.listExecutions)

		// Workers
		r.Get("/workers", srv.listWorkers)
		r.Get("/workers/{id}", srv.getWorker)
		r.Get("/workers/{id}/vms", srv.getWorkerVMs)
		r.Post("/workers/{id}/drain", srv.drainWorker)
		r.Post("/workers/{id}/activate", srv.activateWorker)

		// Cluster
		r.Get("/cluster/stats", srv.getClusterStats)
		r.Get("/cluster/distribution", srv.getVMDistribution)

		// Tasks
		r.Get("/tasks/{id}", srv.getTask)

		// Logs
		r.Post("/logs/query", srv.queryLogs)
		r.Get("/logs/stream", srv.streamLogs) // WebSocket

		// Integrations
		r.Post("/webhooks/{integration}", srv.handleWebhook)

		// Environments
		r.Post("/environments", srv.createEnvironment)
		r.Get("/environments", srv.listEnvironments)
		r.Get("/environments/{id}", srv.getEnvironment)
		r.Put("/environments/{id}", srv.updateEnvironment)
		r.Delete("/environments/{id}", srv.deleteEnvironment)

		// Workspaces
		r.Post("/workspaces", srv.createWorkspace)
		r.Get("/workspaces", srv.listWorkspaces)
		r.Get("/workspaces/{id}", srv.getWorkspace)
		r.Delete("/workspaces/{id}", srv.deleteWorkspace)
		r.Post("/workspaces/{id}/prompts", srv.submitPrompt)
		r.Get("/workspaces/{id}/prompts", srv.listPrompts)
		r.Get("/workspaces/{id}/prompts/{promptId}", srv.getPrompt)
		r.Post("/workspaces/{id}/secrets", srv.addSecret)
		r.Get("/workspaces/{id}/secrets", srv.listSecrets)
		r.Delete("/workspaces/{id}/secrets/{secretId}", srv.deleteSecret)
		r.Get("/workspaces/{id}/ws", srv.workspaceSession) // WebSocket

		// Health
		r.Get("/health", srv.health)
	})

	// Start server
	port := getEnv("PORT", "8080")
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: r,
	}

	// Graceful shutdown
	go func() {
		log.Printf("API Gateway listening on :%s", port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down API Gateway...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("API Gateway stopped")
}

// Handler functions

func (s *Server) serveUI(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/index.html")
}

func (s *Server) createVM(w http.ResponseWriter, r *http.Request) {
	var req api.CreateVMRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	taskID, err := s.taskService.CreateVMTaskWithTools(
		r.Context(),
		req.Name,
		req.VCPUs,
		req.MemoryMB,
		req.AdditionalTools,
		req.ToolVersions,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create VM task", err)
		return
	}

	respondJSON(w, http.StatusAccepted, api.CreateVMResponse{
		TaskID: taskID,
		Status: "pending",
	})
}

func (s *Server) listVMs(w http.ResponseWriter, r *http.Request) {
	vms, err := s.taskService.ListVMs(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list VMs", err)
		return
	}

	vmResponses := make([]*api.VMResponse, len(vms))
	for i, vm := range vms {
		vmResponses[i] = &api.VMResponse{
			ID:         vm.ID,
			Name:       vm.Name,
			Status:     vm.Status,
			VCPUCount:  vm.VCPUCount,
			MemoryMB:   vm.MemoryMB,
			KernelPath: vm.KernelPath,
			RootFSPath: vm.RootFSPath,
			SocketPath: vm.SocketPath,
			CreatedAt:  vm.CreatedAt,
			StartedAt:  vm.StartedAt,
			StoppedAt:  vm.StoppedAt,
			Metadata:   vm.Metadata,
		}
	}

	respondJSON(w, http.StatusOK, api.ListVMsResponse{
		VMs:   vmResponses,
		Total: len(vmResponses),
	})
}

func (s *Server) getVM(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid VM ID", err)
		return
	}

	vm, err := s.taskService.GetVM(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, "VM not found", err)
		return
	}

	respondJSON(w, http.StatusOK, api.VMResponse{
		ID:         vm.ID,
		Name:       vm.Name,
		Status:     vm.Status,
		VCPUCount:  vm.VCPUCount,
		MemoryMB:   vm.MemoryMB,
		KernelPath: vm.KernelPath,
		RootFSPath: vm.RootFSPath,
		SocketPath: vm.SocketPath,
		CreatedAt:  vm.CreatedAt,
		StartedAt:  vm.StartedAt,
		StoppedAt:  vm.StoppedAt,
		Metadata:   vm.Metadata,
	})
}

func (s *Server) deleteVM(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")

	taskID, err := s.taskService.DeleteVMTask(r.Context(), idStr)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete VM", err)
		return
	}

	respondJSON(w, http.StatusAccepted, api.TaskResponse{
		ID:     taskID,
		Type:   "vm:delete",
		Status: "pending",
	})
}

func (s *Server) executeCommand(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")

	var req api.ExecuteCommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	taskID, err := s.taskService.ExecuteCommandTask(r.Context(), idStr, req.Command, req.Args)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to execute command", err)
		return
	}

	respondJSON(w, http.StatusAccepted, api.ExecuteCommandResponse{
		TaskID: taskID,
		VMID:   idStr,
		Status: "pending",
	})
}

func (s *Server) listExecutions(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid VM ID", err)
		return
	}

	executions, err := s.taskService.GetExecutions(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get executions", err)
		return
	}

	execResponses := make([]*api.ExecutionResponse, len(executions))
	for i, exec := range executions {
		execResponses[i] = &api.ExecutionResponse{
			ID:          exec.ID,
			VMID:        exec.VMID,
			Command:     exec.Command,
			Args:        exec.Args,
			ExitCode:    exec.ExitCode,
			Stdout:      exec.Stdout,
			Stderr:      exec.Stderr,
			Error:       exec.Error,
			StartedAt:   exec.StartedAt,
			CompletedAt: exec.CompletedAt,
			DurationMS:  exec.DurationMS,
			Metadata:    exec.Metadata,
		}
	}

	respondJSON(w, http.StatusOK, api.ListExecutionsResponse{
		Executions: execResponses,
		Total:      len(execResponses),
	})
}

func (s *Server) smartExecute(w http.ResponseWriter, r *http.Request) {
	var req api.SmartExecuteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Set defaults if not provided
	if req.VCPUs == 0 {
		req.VCPUs = 1
	}
	if req.MemoryMB == 0 {
		req.MemoryMB = 512
	}

	var selectedVM *uuid.UUID
	var vmName string
	var vmCreated bool
	var vmReused bool

	// Strategy 1: If specific VM name provided, try to find it
	if req.VMName != "" {
		vm, err := s.taskService.GetVMByName(r.Context(), req.VMName)
		if err == nil && vm != nil && vm.Status == "RUNNING" {
			selectedVM = &vm.ID
			vmName = vm.Name
			vmReused = true
			log.Printf("Smart Execute: Reusing specified VM %s (%s)", vmName, vm.ID)
		}
	}

	// Strategy 2: If prefer existing and no specific VM, find any running VM
	if selectedVM == nil && req.PreferExisting {
		vms, err := s.taskService.ListVMs(r.Context())
		if err == nil && len(vms) > 0 {
			// Find first running VM
			for _, vm := range vms {
				if vm.Status == "RUNNING" {
					selectedVM = &vm.ID
					vmName = vm.Name
					vmReused = true
					log.Printf("Smart Execute: Reusing existing VM %s (%s)", vmName, vm.ID)
					break
				}
			}
		}
	}

	// Strategy 3: No suitable VM found, create a new one
	if selectedVM == nil {
		log.Printf("Smart Execute: Creating new VM with %d vCPUs and %dMB memory", req.VCPUs, req.MemoryMB)

		// Generate name if not provided
		if vmName == "" {
			vmName = fmt.Sprintf("smart-vm-%d", time.Now().Unix())
		} else {
			vmName = req.VMName
		}

		// Create VM task
		taskID, err := s.taskService.CreateVMTaskWithTools(
			r.Context(),
			vmName,
			req.VCPUs,
			req.MemoryMB,
			req.RequiredTools,
			nil, // tool versions
		)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to create VM", err)
			return
		}

		log.Printf("Smart Execute: VM creation task submitted (ID: %s), waiting for VM...", taskID)

		// Wait for VM to be created (poll for up to 30 seconds)
		var newVM *uuid.UUID
		for i := 0; i < 30; i++ {
			time.Sleep(1 * time.Second)

			vm, err := s.taskService.GetVMByName(r.Context(), vmName)
			if err == nil && vm != nil && vm.Status == "RUNNING" {
				newVM = &vm.ID
				break
			}
		}

		if newVM == nil {
			respondError(w, http.StatusInternalServerError, "VM creation timed out", nil)
			return
		}

		selectedVM = newVM
		vmCreated = true
		log.Printf("Smart Execute: New VM created successfully: %s (%s)", vmName, *selectedVM)
	}

	// Execute command on selected VM
	// If args are provided, use them directly. Otherwise, wrap the full command in bash -c
	var taskID uuid.UUID
	var execErr error
	if len(req.Args) > 0 {
		// Command and args provided separately
		taskID, execErr = s.taskService.ExecuteCommandTask(r.Context(), selectedVM.String(), req.Command, req.Args)
	} else {
		// Full command string provided - execute via bash -c
		taskID, execErr = s.taskService.ExecuteCommandTask(r.Context(), selectedVM.String(), "bash", []string{"-c", req.Command})
	}

	if execErr != nil {
		respondError(w, http.StatusInternalServerError, "Failed to execute command", execErr)
		return
	}

	log.Printf("Smart Execute: Command execution task submitted (ID: %s) on VM %s", taskID, *selectedVM)

	respondJSON(w, http.StatusAccepted, api.SmartExecuteResponse{
		ExecutionID: taskID,
		VMID:        *selectedVM,
		VMName:      vmName,
		VMCreated:   vmCreated,
		VMReused:    vmReused,
		Status:      "pending",
		Message:     fmt.Sprintf("Command queued for execution on VM %s", vmName),
	})
}

func (s *Server) getTask(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement task status endpoint
	respondJSON(w, http.StatusOK, map[string]string{"status": "not_implemented"})
}

func (s *Server) queryLogs(w http.ResponseWriter, r *http.Request) {
	if s.logger == nil {
		respondError(w, http.StatusServiceUnavailable, "Logging not configured", nil)
		return
	}

	var req api.LogQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// TODO: Query logs from Loki
	respondJSON(w, http.StatusOK, api.LogQueryResponse{
		Logs:  []*api.LogEntry{},
		Total: 0,
	})
}

func (s *Server) streamLogs(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement WebSocket log streaming
	respondError(w, http.StatusNotImplemented, "Log streaming not yet implemented", nil)
}

func (s *Server) handleWebhook(w http.ResponseWriter, r *http.Request) {
	integrationName := chi.URLParam(r, "integration")

	_, err := s.integrations.Get(integrationName)
	if err != nil {
		respondError(w, http.StatusNotFound, "Integration not found", err)
		return
	}

	// TODO: Verify webhook signature
	// TODO: Parse webhook payload and publish event

	respondJSON(w, http.StatusOK, map[string]string{"status": "received"})
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	components := make(map[string]string)

	// Check integrations
	if s.integrations != nil {
		healthResults := s.integrations.HealthCheck(r.Context())
		for name, err := range healthResults {
			if err != nil {
				components[name] = "unhealthy"
			} else {
				components[name] = "healthy"
			}
		}
	}

	// Check logger
	if s.logger != nil {
		if err := s.logger.Health(r.Context()); err != nil {
			components["loki"] = "unhealthy"
		} else {
			components["loki"] = "healthy"
		}
	}

	// Check event bus
	if s.eventBus != nil {
		if err := s.eventBus.Health(r.Context()); err != nil {
			components["event_bus"] = "unhealthy"
		} else {
			components["event_bus"] = "healthy"
		}
	}

	respondJSON(w, http.StatusOK, api.HealthResponse{
		Status:     "ok",
		Components: components,
		Timestamp:  time.Now(),
	})
}

// Worker management handlers

func (s *Server) listWorkers(w http.ResponseWriter, r *http.Request) {
	// Check for zone filter
	zone := r.URL.Query().Get("zone")

	var workers []*service.WorkerStats
	var err error

	if zone != "" {
		workers, err = s.workerService.ListWorkersByZone(r.Context(), zone)
	} else {
		workers, err = s.workerService.ListWorkers(r.Context())
	}

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list workers", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"workers": workers,
		"total":   len(workers),
	})
}

func (s *Server) getWorker(w http.ResponseWriter, r *http.Request) {
	workerID := chi.URLParam(r, "id")

	worker, err := s.workerService.GetWorker(r.Context(), workerID)
	if err != nil {
		respondError(w, http.StatusNotFound, "Worker not found", err)
		return
	}

	respondJSON(w, http.StatusOK, worker)
}

func (s *Server) getWorkerVMs(w http.ResponseWriter, r *http.Request) {
	workerID := chi.URLParam(r, "id")

	vms, err := s.workerService.GetWorkerVMs(r.Context(), workerID)
	if err != nil {
		respondError(w, http.StatusNotFound, "Failed to get worker VMs", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"worker_id": workerID,
		"vms":       vms,
		"total":     len(vms),
	})
}

func (s *Server) drainWorker(w http.ResponseWriter, r *http.Request) {
	workerID := chi.URLParam(r, "id")

	if err := s.workerService.DrainWorker(r.Context(), workerID); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to drain worker", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"worker_id": workerID,
		"status":    "draining",
		"message":   "Worker marked as draining. It will stop accepting new tasks.",
	})
}

func (s *Server) activateWorker(w http.ResponseWriter, r *http.Request) {
	workerID := chi.URLParam(r, "id")

	if err := s.workerService.ActivateWorker(r.Context(), workerID); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to activate worker", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"worker_id": workerID,
		"status":    "active",
		"message":   "Worker activated. It will resume accepting tasks.",
	})
}

func (s *Server) getClusterStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.workerService.GetClusterStats(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get cluster stats", err)
		return
	}

	respondJSON(w, http.StatusOK, stats)
}

func (s *Server) getVMDistribution(w http.ResponseWriter, r *http.Request) {
	distribution, err := s.workerService.GetVMDistribution(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get VM distribution", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"distribution":  distribution,
		"total_workers": len(distribution),
	})
}

// Workspace handlers

func (s *Server) createWorkspace(w http.ResponseWriter, r *http.Request) {
	var req api.CreateWorkspaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Validate required fields
	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "Name is required", nil)
		return
	}
	if req.VCPUs < 1 {
		req.VCPUs = 1
	}
	if req.MemoryMB < 128 {
		req.MemoryMB = 512
	}
	if req.AIAssistant == "" {
		req.AIAssistant = "claude-code"
	}

	taskID, workspaceID, err := s.workspaceService.CreateWorkspace(r.Context(), &req)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create workspace", err)
		return
	}

	respondJSON(w, http.StatusAccepted, api.CreateWorkspaceResponse{
		TaskID:      taskID,
		WorkspaceID: workspaceID,
		Status:      "creating",
	})
}

func (s *Server) listWorkspaces(w http.ResponseWriter, r *http.Request) {
	workspaces, err := s.workspaceService.ListWorkspaces(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list workspaces", err)
		return
	}

	responses := make([]*api.WorkspaceResponse, len(workspaces))
	for i, ws := range workspaces {
		responses[i] = storageWorkspaceToResponse(ws)
	}

	respondJSON(w, http.StatusOK, api.ListWorkspacesResponse{
		Workspaces: responses,
		Total:      len(responses),
	})
}

func (s *Server) getWorkspace(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid workspace ID", err)
		return
	}

	workspace, err := s.workspaceService.GetWorkspace(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Workspace not found", err)
		return
	}

	respondJSON(w, http.StatusOK, storageWorkspaceToResponse(workspace))
}

func (s *Server) deleteWorkspace(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid workspace ID", err)
		return
	}

	taskID, err := s.workspaceService.DeleteWorkspace(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete workspace", err)
		return
	}

	respondJSON(w, http.StatusAccepted, api.TaskResponse{
		ID:     taskID,
		Type:   "workspace:delete",
		Status: "pending",
	})
}

func (s *Server) submitPrompt(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	workspaceID, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid workspace ID", err)
		return
	}

	var req api.SubmitPromptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if req.Prompt == "" {
		respondError(w, http.StatusBadRequest, "Prompt is required", nil)
		return
	}

	promptID, err := s.workspaceService.SubmitPrompt(r.Context(), workspaceID, &req)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to submit prompt", err)
		return
	}

	respondJSON(w, http.StatusAccepted, api.SubmitPromptResponse{
		PromptID:    promptID,
		WorkspaceID: workspaceID,
		Status:      "pending",
	})
}

func (s *Server) listPrompts(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	workspaceID, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid workspace ID", err)
		return
	}

	prompts, err := s.workspaceService.ListPrompts(r.Context(), workspaceID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list prompts", err)
		return
	}

	responses := make([]*api.PromptResponse, len(prompts))
	for i, p := range prompts {
		responses[i] = storagePromptToResponse(p)
	}

	respondJSON(w, http.StatusOK, api.ListPromptsResponse{
		Prompts: responses,
		Total:   len(responses),
	})
}

func (s *Server) getPrompt(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "promptId")
	promptID, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid prompt ID", err)
		return
	}

	prompt, err := s.workspaceService.GetPrompt(r.Context(), promptID)
	if err != nil {
		respondError(w, http.StatusNotFound, "Prompt not found", err)
		return
	}

	respondJSON(w, http.StatusOK, storagePromptToResponse(prompt))
}

func (s *Server) addSecret(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	workspaceID, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid workspace ID", err)
		return
	}

	var req api.AddSecretRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if req.Name == "" || req.Value == "" {
		respondError(w, http.StatusBadRequest, "Name and value are required", nil)
		return
	}

	secretID, err := s.workspaceService.AddSecret(r.Context(), workspaceID, &req)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to add secret", err)
		return
	}

	respondJSON(w, http.StatusCreated, api.AddSecretResponse{
		SecretID: secretID,
		Name:     req.Name,
		Status:   "created",
	})
}

func (s *Server) listSecrets(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	workspaceID, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid workspace ID", err)
		return
	}

	secrets, err := s.workspaceService.ListSecrets(r.Context(), workspaceID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list secrets", err)
		return
	}

	// Only return metadata, never the actual values
	responses := make([]*api.SecretResponse, len(secrets))
	for i, secret := range secrets {
		description := ""
		if secret.Description != nil {
			description = *secret.Description
		}
		responses[i] = &api.SecretResponse{
			ID:          secret.ID,
			Name:        secret.Name,
			Type:        secret.SecretType,
			Description: description,
			Scope:       secret.Scope,
			CreatedAt:   secret.CreatedAt,
			UpdatedAt:   secret.UpdatedAt,
		}
	}

	respondJSON(w, http.StatusOK, api.ListSecretsResponse{
		Secrets: responses,
		Total:   len(responses),
	})
}

func (s *Server) deleteSecret(w http.ResponseWriter, r *http.Request) {
	secretIDStr := chi.URLParam(r, "secretId")
	secretID, err := uuid.Parse(secretIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid secret ID", err)
		return
	}

	if err := s.workspaceService.DeleteSecret(r.Context(), secretID); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete secret", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) workspaceSession(w http.ResponseWriter, r *http.Request) {
	// Check if session manager is available
	if s.sessionManager == nil {
		respondError(w, http.StatusServiceUnavailable, "WebSocket streaming not available (orchestrator not initialized)", nil)
		return
	}

	// Parse workspace ID
	idStr := chi.URLParam(r, "id")
	workspaceID, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid workspace ID", err)
		return
	}

	// Handle WebSocket upgrade and session
	s.sessionManager.HandleSession(w, r, workspaceID)
}

// Environment handlers

func (s *Server) createEnvironment(w http.ResponseWriter, r *http.Request) {
	var req api.CreateEnvironmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Validate required fields
	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "Name is required", nil)
		return
	}

	// Convert request to storage type
	env := &storage.Environment{
		Name:               req.Name,
		GitRepoURL:         req.GitRepoURL,
		GitBranch:          req.GitBranch,
		WorkingDirectory:   req.WorkingDirectory,
		VCPUs:              req.VCPUs,
		MemoryMB:           req.MemoryMB,
		Tools:              req.Tools,
		EnvVars:            req.EnvVars,
		IdleTimeoutSeconds: req.IdleTimeoutSeconds,
	}

	if req.Description != "" {
		env.Description = &req.Description
	}

	// Convert MCP servers
	if len(req.MCPServers) > 0 {
		env.MCPServers = make([]storage.MCPServerConfig, len(req.MCPServers))
		for i, mcp := range req.MCPServers {
			env.MCPServers[i] = storage.MCPServerConfig{
				Name:    mcp.Name,
				Type:    storage.MCPServerType(mcp.Type),
				Command: mcp.Command,
				Args:    mcp.Args,
				URL:     mcp.URL,
				Headers: mcp.Headers,
				Env:     mcp.Env,
			}
		}
	}

	if err := s.store.Environments().Create(r.Context(), env); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create environment", err)
		return
	}

	respondJSON(w, http.StatusCreated, storageEnvironmentToResponse(env))
}

func (s *Server) listEnvironments(w http.ResponseWriter, r *http.Request) {
	environments, err := s.store.Environments().List(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list environments", err)
		return
	}

	responses := make([]*api.EnvironmentResponse, len(environments))
	for i, env := range environments {
		responses[i] = storageEnvironmentToResponse(env)
	}

	respondJSON(w, http.StatusOK, api.ListEnvironmentsResponse{
		Environments: responses,
		Total:        len(responses),
	})
}

func (s *Server) getEnvironment(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid environment ID", err)
		return
	}

	env, err := s.store.Environments().Get(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Environment not found", err)
		return
	}

	respondJSON(w, http.StatusOK, storageEnvironmentToResponse(env))
}

func (s *Server) updateEnvironment(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid environment ID", err)
		return
	}

	// Get existing environment
	env, err := s.store.Environments().Get(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Environment not found", err)
		return
	}

	var req api.UpdateEnvironmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Update fields if provided
	if req.Name != "" {
		env.Name = req.Name
	}
	if req.Description != "" {
		env.Description = &req.Description
	}
	if req.VCPUs > 0 {
		env.VCPUs = req.VCPUs
	}
	if req.MemoryMB > 0 {
		env.MemoryMB = req.MemoryMB
	}
	if req.GitRepoURL != "" {
		env.GitRepoURL = req.GitRepoURL
	}
	if req.GitBranch != "" {
		env.GitBranch = req.GitBranch
	}
	if req.WorkingDirectory != "" {
		env.WorkingDirectory = req.WorkingDirectory
	}
	if req.Tools != nil {
		env.Tools = req.Tools
	}
	if req.EnvVars != nil {
		env.EnvVars = req.EnvVars
	}
	if req.IdleTimeoutSeconds > 0 {
		env.IdleTimeoutSeconds = req.IdleTimeoutSeconds
	}

	// Update MCP servers if provided
	if req.MCPServers != nil {
		env.MCPServers = make([]storage.MCPServerConfig, len(req.MCPServers))
		for i, mcp := range req.MCPServers {
			env.MCPServers[i] = storage.MCPServerConfig{
				Name:    mcp.Name,
				Type:    storage.MCPServerType(mcp.Type),
				Command: mcp.Command,
				Args:    mcp.Args,
				URL:     mcp.URL,
				Headers: mcp.Headers,
				Env:     mcp.Env,
			}
		}
	}

	if err := s.store.Environments().Update(r.Context(), env); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update environment", err)
		return
	}

	respondJSON(w, http.StatusOK, storageEnvironmentToResponse(env))
}

func (s *Server) deleteEnvironment(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid environment ID", err)
		return
	}

	if err := s.store.Environments().Delete(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete environment", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// Environment response helper
func storageEnvironmentToResponse(env *storage.Environment) *api.EnvironmentResponse {
	resp := &api.EnvironmentResponse{
		ID:                 env.ID,
		Name:               env.Name,
		VCPUs:              env.VCPUs,
		MemoryMB:           env.MemoryMB,
		GitRepoURL:         env.GitRepoURL,
		GitBranch:          env.GitBranch,
		WorkingDirectory:   env.WorkingDirectory,
		Tools:              env.Tools,
		EnvVars:            env.EnvVars,
		IdleTimeoutSeconds: env.IdleTimeoutSeconds,
		CreatedAt:          env.CreatedAt,
		UpdatedAt:          env.UpdatedAt,
	}

	if env.Description != nil {
		resp.Description = *env.Description
	}

	// Convert MCP servers
	if len(env.MCPServers) > 0 {
		resp.MCPServers = make([]api.MCPServerResponse, len(env.MCPServers))
		for i, mcp := range env.MCPServers {
			resp.MCPServers[i] = api.MCPServerResponse{
				Name:    mcp.Name,
				Type:    string(mcp.Type),
				Command: mcp.Command,
				Args:    mcp.Args,
				URL:     mcp.URL,
				Headers: mcp.Headers,
				Env:     mcp.Env,
			}
		}
	}

	return resp
}

// Workspace response helpers

func storageWorkspaceToResponse(ws *storage.Workspace) *api.WorkspaceResponse {
	resp := &api.WorkspaceResponse{
		ID:                ws.ID,
		Name:              ws.Name,
		Status:            ws.Status,
		AIAssistant:       ws.AIAssistant,
		AIAssistantConfig: ws.AIAssistantConfig,
		WorkingDirectory:  ws.WorkingDirectory,
		CreatedAt:         ws.CreatedAt,
		ReadyAt:           ws.ReadyAt,
		StoppedAt:         ws.StoppedAt,
		IdleSince:         ws.IdleSince,
		Metadata:          ws.Metadata,
	}
	if ws.Description != nil {
		resp.Description = *ws.Description
	}
	if ws.VMID != nil {
		resp.VMID = ws.VMID
	}
	if ws.EnvironmentID != nil {
		resp.EnvironmentID = ws.EnvironmentID
	}
	return resp
}

func storagePromptToResponse(p *storage.PromptTask) *api.PromptResponse {
	resp := &api.PromptResponse{
		ID:          p.ID,
		WorkspaceID: p.WorkspaceID,
		Prompt:      p.Prompt,
		Priority:    p.Priority,
		Status:      p.Status,
		CreatedAt:   p.CreatedAt,
		ScheduledAt: p.ScheduledAt,
		StartedAt:   p.StartedAt,
		CompletedAt: p.CompletedAt,
		DurationMS:  p.DurationMS,
		Metadata:    p.Metadata,
	}
	if p.SystemPrompt != nil {
		resp.SystemPrompt = *p.SystemPrompt
	}
	if p.WorkingDirectory != nil {
		resp.WorkingDirectory = *p.WorkingDirectory
	}
	if p.Environment != nil {
		resp.Environment = p.Environment
	}
	if p.ExitCode != nil {
		resp.ExitCode = p.ExitCode
	}
	if p.Stdout != nil {
		resp.Stdout = p.Stdout
	}
	if p.Stderr != nil {
		resp.Stderr = p.Stderr
	}
	if p.Error != nil {
		resp.Error = p.Error
	}
	return resp
}

// Helper functions

func respondJSON(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, code int, message string, err error) {
	errMsg := message
	if err != nil {
		errMsg = fmt.Sprintf("%s: %v", message, err)
	}

	respondJSON(w, code, api.ErrorResponse{
		Error:   http.StatusText(code),
		Message: errMsg,
		Code:    code,
	})
}

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
