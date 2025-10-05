package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aetherium/aetherium/pkg/api"
	"github.com/aetherium/aetherium/pkg/events/redis"
	"github.com/aetherium/aetherium/pkg/integrations"
	githubIntegration "github.com/aetherium/aetherium/pkg/integrations/github"
	"github.com/aetherium/aetherium/pkg/integrations/slack"
	"github.com/aetherium/aetherium/pkg/logging/loki"
	"github.com/aetherium/aetherium/pkg/queue/asynq"
	"github.com/aetherium/aetherium/pkg/service"
	"github.com/aetherium/aetherium/pkg/storage/postgres"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
)

type Server struct {
	taskService  *service.TaskService
	integrations *integrations.Registry
	logger       *loki.LokiLogger
	eventBus     *redis.RedisEventBus
}

func main() {
	log.Println("Aetherium API Gateway starting...")

	// Initialize PostgreSQL store
	store, err := postgres.NewStore(postgres.Config{
		Host:         getEnv("POSTGRES_HOST", "localhost"),
		Port:         5432,
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

	// Create server
	srv := &Server{
		taskService:  taskService,
		integrations: registry,
		logger:       logger,
		eventBus:     eventBus,
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

	// Routes
	r.Route("/api/v1", func(r chi.Router) {
		// VMs
		r.Post("/vms", srv.createVM)
		r.Get("/vms", srv.listVMs)
		r.Get("/vms/{id}", srv.getVM)
		r.Delete("/vms/{id}", srv.deleteVM)
		r.Post("/vms/{id}/execute", srv.executeCommand)
		r.Get("/vms/{id}/executions", srv.listExecutions)

		// Tasks
		r.Get("/tasks/{id}", srv.getTask)

		// Logs
		r.Post("/logs/query", srv.queryLogs)
		r.Get("/logs/stream", srv.streamLogs) // WebSocket

		// Integrations
		r.Post("/webhooks/{integration}", srv.handleWebhook)

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
