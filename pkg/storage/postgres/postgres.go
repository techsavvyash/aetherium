package postgres

import (
	"database/sql"
	"fmt"

	"github.com/aetherium/aetherium/pkg/storage"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// Store implements storage.Store using PostgreSQL
type Store struct {
	db              *sqlx.DB
	vms             storage.VMRepository
	tasks           storage.TaskRepository
	jobs            storage.JobRepository
	executions      storage.ExecutionRepository
	workers         storage.WorkerRepository
	workerMetrics   storage.WorkerMetricRepository
	environments    storage.EnvironmentRepository
	workspaces      storage.WorkspaceRepository
	secrets         storage.SecretRepository
	prepSteps       storage.PrepStepRepository
	promptTasks     storage.PromptTaskRepository
	sessions        storage.SessionRepository
	sessionMessages storage.SessionMessageRepository
}

// Config holds PostgreSQL configuration
type Config struct {
	Host         string
	Port         int
	User         string
	Password     string
	Database     string
	SSLMode      string
	MaxOpenConns int
	MaxIdleConns int
}

// NewStore creates a new PostgreSQL store
func NewStore(config Config) (*Store, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.User, config.Password, config.Database, config.SSLMode,
	)

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if config.MaxOpenConns > 0 {
		db.SetMaxOpenConns(config.MaxOpenConns)
	}
	if config.MaxIdleConns > 0 {
		db.SetMaxIdleConns(config.MaxIdleConns)
	}

	store := &Store{
		db:              db,
		vms:             &vmRepository{db: db},
		tasks:           &taskRepository{db: db},
		jobs:            &jobRepository{db: db},
		executions:      &executionRepository{db: db},
		workers:         &workerRepository{db: db},
		workerMetrics:   &workerMetricRepository{db: db},
		environments:    &environmentRepository{db: db},
		workspaces:      &workspaceRepository{db: db},
		secrets:         &secretRepository{db: db},
		prepSteps:       &prepStepRepository{db: db},
		promptTasks:     &promptTaskRepository{db: db},
		sessions:        &sessionRepository{db: db},
		sessionMessages: &sessionMessageRepository{db: db},
	}

	return store, nil
}

// RunMigrations runs database migrations
func (s *Store) RunMigrations(migrationsPath string) error {
	driver, err := postgres.WithInstance(s.db.DB, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", migrationsPath),
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// VMs returns the VM repository
func (s *Store) VMs() storage.VMRepository {
	return s.vms
}

// Tasks returns the task repository
func (s *Store) Tasks() storage.TaskRepository {
	return s.tasks
}

// Jobs returns the job repository
func (s *Store) Jobs() storage.JobRepository {
	return s.jobs
}

// Executions returns the execution repository
func (s *Store) Executions() storage.ExecutionRepository {
	return s.executions
}

// Workers returns the worker repository
func (s *Store) Workers() storage.WorkerRepository {
	return s.workers
}

// WorkerMetrics returns the worker metrics repository
func (s *Store) WorkerMetrics() storage.WorkerMetricRepository {
	return s.workerMetrics
}

// Environments returns the environment repository
func (s *Store) Environments() storage.EnvironmentRepository {
	return s.environments
}

// Workspaces returns the workspace repository
func (s *Store) Workspaces() storage.WorkspaceRepository {
	return s.workspaces
}

// Secrets returns the secret repository
func (s *Store) Secrets() storage.SecretRepository {
	return s.secrets
}

// PrepSteps returns the prep step repository
func (s *Store) PrepSteps() storage.PrepStepRepository {
	return s.prepSteps
}

// PromptTasks returns the prompt task repository
func (s *Store) PromptTasks() storage.PromptTaskRepository {
	return s.promptTasks
}

// Sessions returns the session repository
func (s *Store) Sessions() storage.SessionRepository {
	return s.sessions
}

// SessionMessages returns the session message repository
func (s *Store) SessionMessages() storage.SessionMessageRepository {
	return s.sessionMessages
}

// Close closes the database connection
func (s *Store) Close() error {
	return s.db.Close()
}

// Helper function to convert nil strings to SQL NULL
func toNullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: *s, Valid: true}
}

// Helper function to convert SQL NULL to nil strings
func fromNullString(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	return &ns.String
}
