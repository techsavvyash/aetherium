package postgres

import (
	"context"
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
	db         *sqlx.DB
	vms        storage.VMRepository
	tasks      storage.TaskRepository
	jobs       storage.JobRepository
	executions storage.ExecutionRepository
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
		db:         db,
		vms:        &vmRepository{db: db},
		tasks:      &taskRepository{db: db},
		jobs:       &jobRepository{db: db},
		executions: &executionRepository{db: db},
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
