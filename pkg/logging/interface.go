package logging

import (
	"context"

	"github.com/aetherium/aetherium/pkg/types"
)

// Logger defines the interface for logging implementations
type Logger interface {
	// Log writes a log entry
	Log(ctx context.Context, level types.LogLevel, message string, fields map[string]interface{}) error

	// Debug logs a debug message
	Debug(ctx context.Context, message string, fields map[string]interface{}) error

	// Info logs an info message
	Info(ctx context.Context, message string, fields map[string]interface{}) error

	// Warn logs a warning message
	Warn(ctx context.Context, message string, fields map[string]interface{}) error

	// Error logs an error message
	Error(ctx context.Context, message string, fields map[string]interface{}) error

	// Stream returns a channel that streams log entries matching the query
	Stream(ctx context.Context, query *Query) (<-chan *types.LogEntry, error)

	// Query retrieves historical log entries
	Query(ctx context.Context, query *Query) ([]*types.LogEntry, error)

	// Health returns the health status of the logger
	Health(ctx context.Context) error

	// Close closes the logger connection
	Close() error
}

// Query represents a log query
type Query struct {
	// Labels to filter by (e.g., task_id, vm_id, project_id)
	Labels map[string]string

	// Log level filter
	Level *types.LogLevel

	// Time range
	StartTime *int64
	EndTime   *int64

	// Limit number of results
	Limit int

	// Text search in message
	SearchText *string
}

// Config represents logging configuration
type Config struct {
	// Provider-specific configuration
	Options map[string]interface{}
}
