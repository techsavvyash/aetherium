package stdout

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/aetherium/aetherium/libs/common/pkg/logging"
	"github.com/aetherium/aetherium/libs/types/pkg/domain"
)

// StdoutLogger is a simple logger that writes to stdout
type StdoutLogger struct {
	colorize bool
	mu       sync.Mutex
}

// NewStdoutLogger creates a new stdout logger
func NewStdoutLogger(colorize bool) *StdoutLogger {
	return &StdoutLogger{
		colorize: colorize,
	}
}

// Log writes a log entry to stdout
func (s *StdoutLogger) Log(ctx context.Context, level types.LogLevel, message string, fields map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	levelStr := s.formatLevel(level)

	// Build log line
	logLine := fmt.Sprintf("[%s] %s: %s", timestamp, levelStr, message)

	// Add fields if present
	if len(fields) > 0 {
		logLine += " |"
		for key, value := range fields {
			logLine += fmt.Sprintf(" %s=%v", key, value)
		}
	}

	fmt.Fprintln(os.Stdout, logLine)
	return nil
}

// Debug logs a debug message
func (s *StdoutLogger) Debug(ctx context.Context, message string, fields map[string]interface{}) error {
	return s.Log(ctx, types.LogLevelDebug, message, fields)
}

// Info logs an info message
func (s *StdoutLogger) Info(ctx context.Context, message string, fields map[string]interface{}) error {
	return s.Log(ctx, types.LogLevelInfo, message, fields)
}

// Warn logs a warning message
func (s *StdoutLogger) Warn(ctx context.Context, message string, fields map[string]interface{}) error {
	return s.Log(ctx, types.LogLevelWarn, message, fields)
}

// Error logs an error message
func (s *StdoutLogger) Error(ctx context.Context, message string, fields map[string]interface{}) error {
	return s.Log(ctx, types.LogLevelError, message, fields)
}

// Stream returns a channel that streams log entries matching the query
func (s *StdoutLogger) Stream(ctx context.Context, query *logging.Query) (<-chan *types.LogEntry, error) {
	// Stdout logger doesn't support streaming (it's write-only)
	return nil, fmt.Errorf("streaming not supported for stdout logger")
}

// Query retrieves historical log entries (not supported for stdout)
func (s *StdoutLogger) Query(ctx context.Context, query *logging.Query) ([]*types.LogEntry, error) {
	// Stdout logger doesn't store logs, so querying is not supported
	return nil, fmt.Errorf("querying not supported for stdout logger")
}

// Health checks if the logger is operational
func (s *StdoutLogger) Health(ctx context.Context) error {
	// Always healthy if we can write to stdout
	return nil
}

// Close closes the logger
func (s *StdoutLogger) Close() error {
	return nil
}

func (s *StdoutLogger) formatLevel(level types.LogLevel) string {
	if !s.colorize {
		return string(level)
	}

	// ANSI color codes
	const (
		colorReset  = "\033[0m"
		colorRed    = "\033[31m"
		colorYellow = "\033[33m"
		colorBlue   = "\033[34m"
		colorGray   = "\033[90m"
	)

	switch level {
	case types.LogLevelError:
		return colorRed + "ERROR" + colorReset
	case types.LogLevelWarn:
		return colorYellow + "WARN" + colorReset
	case types.LogLevelInfo:
		return colorBlue + "INFO" + colorReset
	case types.LogLevelDebug:
		return colorGray + "DEBUG" + colorReset
	default:
		return string(level)
	}
}

// Ensure StdoutLogger implements Logger interface
var _ logging.Logger = (*StdoutLogger)(nil)
