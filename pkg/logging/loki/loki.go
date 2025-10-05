package loki

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/aetherium/aetherium/pkg/logging"
	"github.com/aetherium/aetherium/pkg/types"
)

// LokiLogger implements the Logger interface for Grafana Loki
type LokiLogger struct {
	config     *Config
	client     *http.Client
	batch      []*types.LogEntry
	batchMu    sync.Mutex
	stopChan   chan struct{}
	flushTicker *time.Ticker
}

// Config holds Loki-specific configuration
type Config struct {
	URL           string        // Loki push URL (e.g., http://localhost:3100/loki/api/v1/push)
	BatchSize     int           // Number of logs to batch before sending
	BatchInterval time.Duration // How often to flush logs
	Timeout       time.Duration // HTTP request timeout
	Labels        map[string]string // Global labels to attach to all logs
}

// NewLokiLogger creates a new Loki logger
func NewLokiLogger(config *Config) (*LokiLogger, error) {
	if config.URL == "" {
		return nil, fmt.Errorf("loki URL is required")
	}

	if config.BatchSize == 0 {
		config.BatchSize = 100
	}

	if config.BatchInterval == 0 {
		config.BatchInterval = 5 * time.Second
	}

	if config.Timeout == 0 {
		config.Timeout = 10 * time.Second
	}

	if config.Labels == nil {
		config.Labels = make(map[string]string)
	}

	// Ensure service label exists
	if _, exists := config.Labels["service"]; !exists {
		config.Labels["service"] = "aetherium"
	}

	logger := &LokiLogger{
		config:      config,
		client:      &http.Client{Timeout: config.Timeout},
		batch:       make([]*types.LogEntry, 0, config.BatchSize),
		stopChan:    make(chan struct{}),
		flushTicker: time.NewTicker(config.BatchInterval),
	}

	// Start background flusher
	go logger.backgroundFlusher()

	return logger, nil
}

// Log writes a log entry
func (l *LokiLogger) Log(ctx context.Context, level types.LogLevel, message string, fields map[string]interface{}) error {
	entry := &types.LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Fields:    fields,
	}

	// Extract common fields
	if taskID, ok := fields["task_id"].(string); ok {
		entry.TaskID = taskID
	}
	if vmID, ok := fields["vm_id"].(string); ok {
		entry.VMID = vmID
	}

	l.batchMu.Lock()
	l.batch = append(l.batch, entry)
	shouldFlush := len(l.batch) >= l.config.BatchSize
	l.batchMu.Unlock()

	if shouldFlush {
		return l.flush()
	}

	return nil
}

// Debug logs a debug message
func (l *LokiLogger) Debug(ctx context.Context, message string, fields map[string]interface{}) error {
	return l.Log(ctx, types.LogLevelDebug, message, fields)
}

// Info logs an info message
func (l *LokiLogger) Info(ctx context.Context, message string, fields map[string]interface{}) error {
	return l.Log(ctx, types.LogLevelInfo, message, fields)
}

// Warn logs a warning message
func (l *LokiLogger) Warn(ctx context.Context, message string, fields map[string]interface{}) error {
	return l.Log(ctx, types.LogLevelWarn, message, fields)
}

// Error logs an error message
func (l *LokiLogger) Error(ctx context.Context, message string, fields map[string]interface{}) error {
	return l.Log(ctx, types.LogLevelError, message, fields)
}

// flush sends batched logs to Loki
func (l *LokiLogger) flush() error {
	l.batchMu.Lock()
	if len(l.batch) == 0 {
		l.batchMu.Unlock()
		return nil
	}

	entries := l.batch
	l.batch = make([]*types.LogEntry, 0, l.config.BatchSize)
	l.batchMu.Unlock()

	return l.sendToLoki(entries)
}

// sendToLoki sends log entries to Loki
func (l *LokiLogger) sendToLoki(entries []*types.LogEntry) error {
	if len(entries) == 0 {
		return nil
	}

	// Group entries by label set
	streams := make(map[string]*lokiStream)

	for _, entry := range entries {
		labels := l.buildLabels(entry)
		labelKey := serializeLabels(labels)

		stream, exists := streams[labelKey]
		if !exists {
			stream = &lokiStream{
				Stream: labels,
				Values: [][]string{},
			}
			streams[labelKey] = stream
		}

		// Format: [timestamp_ns, log_line]
		timestamp := fmt.Sprintf("%d", entry.Timestamp.UnixNano())
		logLine := l.formatLogLine(entry)
		stream.Values = append(stream.Values, []string{timestamp, logLine})
	}

	// Build Loki push request
	streamList := make([]*lokiStream, 0, len(streams))
	for _, stream := range streams {
		streamList = append(streamList, stream)
	}

	payload := lokiPushRequest{
		Streams: streamList,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal loki payload: %w", err)
	}

	req, err := http.NewRequest("POST", l.config.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := l.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send logs to loki: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("loki returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// buildLabels creates label set for a log entry
func (l *LokiLogger) buildLabels(entry *types.LogEntry) map[string]string {
	labels := make(map[string]string)

	// Copy global labels
	for k, v := range l.config.Labels {
		labels[k] = v
	}

	// Add level
	labels["level"] = string(entry.Level)

	// Add task_id if present
	if entry.TaskID != "" {
		labels["task_id"] = entry.TaskID
	}

	// Add vm_id if present
	if entry.VMID != "" {
		labels["vm_id"] = entry.VMID
	}

	// Add component if present in fields
	if component, ok := entry.Fields["component"].(string); ok {
		labels["component"] = component
	}

	return labels
}

// formatLogLine formats a log entry as a single line
func (l *LokiLogger) formatLogLine(entry *types.LogEntry) string {
	// Create structured log line
	logData := map[string]interface{}{
		"timestamp": entry.Timestamp.Format(time.RFC3339Nano),
		"level":     entry.Level,
		"message":   entry.Message,
	}

	// Add fields
	if len(entry.Fields) > 0 {
		logData["fields"] = entry.Fields
	}

	jsonLine, _ := json.Marshal(logData)
	return string(jsonLine)
}

// backgroundFlusher periodically flushes logs
func (l *LokiLogger) backgroundFlusher() {
	for {
		select {
		case <-l.flushTicker.C:
			l.flush()
		case <-l.stopChan:
			l.flush() // Final flush
			return
		}
	}
}

// Stream returns a channel that streams log entries matching the query
func (l *LokiLogger) Stream(ctx context.Context, query *logging.Query) (<-chan *types.LogEntry, error) {
	// TODO: Implement log streaming via Loki query API
	logChan := make(chan *types.LogEntry)
	close(logChan)
	return logChan, fmt.Errorf("log streaming not yet implemented")
}

// Query retrieves historical log entries
func (l *LokiLogger) Query(ctx context.Context, query *logging.Query) ([]*types.LogEntry, error) {
	// Build LogQL query
	logQL := l.buildLogQLQuery(query)

	// Query Loki
	queryURL := fmt.Sprintf("%s/loki/api/v1/query_range", l.config.URL)

	req, err := http.NewRequest("GET", queryURL, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("query", logQL)
	if query.StartTime != nil {
		q.Add("start", fmt.Sprintf("%d", *query.StartTime))
	}
	if query.EndTime != nil {
		q.Add("end", fmt.Sprintf("%d", *query.EndTime))
	}
	if query.Limit > 0 {
		q.Add("limit", fmt.Sprintf("%d", query.Limit))
	}
	req.URL.RawQuery = q.Encode()

	resp, err := l.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("loki query failed: %s", string(body))
	}

	var result lokiQueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	// Convert to log entries
	entries := make([]*types.LogEntry, 0)
	for _, stream := range result.Data.Result {
		for _, value := range stream.Values {
			if len(value) >= 2 {
				entry := &types.LogEntry{
					Message: value[1],
				}
				// Parse timestamp
				if ts, err := parseTimestamp(value[0]); err == nil {
					entry.Timestamp = ts
				}
				entries = append(entries, entry)
			}
		}
	}

	return entries, nil
}

// buildLogQLQuery builds a LogQL query string
func (l *LokiLogger) buildLogQLQuery(query *logging.Query) string {
	// Start with service label
	logQL := fmt.Sprintf("{service=\"%s\"}", l.config.Labels["service"])

	// Add label filters
	for k, v := range query.Labels {
		logQL = fmt.Sprintf("{%s, %s=\"%s\"}", logQL[1:len(logQL)-1], k, v)
	}

	// Add level filter
	if query.Level != nil {
		logQL = fmt.Sprintf("{%s, level=\"%s\"}", logQL[1:len(logQL)-1], *query.Level)
	}

	// Add text search filter
	if query.SearchText != nil && *query.SearchText != "" {
		logQL = fmt.Sprintf("%s |= `%s`", logQL, *query.SearchText)
	}

	return logQL
}

// Health returns the health status of the logger
func (l *LokiLogger) Health(ctx context.Context) error {
	// Try to query Loki ready endpoint
	healthURL := fmt.Sprintf("%s/ready", l.config.URL)
	resp, err := l.client.Get(healthURL)
	if err != nil {
		return fmt.Errorf("loki health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("loki not ready: status %d", resp.StatusCode)
	}

	return nil
}

// Close closes the logger connection
func (l *LokiLogger) Close() error {
	close(l.stopChan)
	l.flushTicker.Stop()
	return l.flush() // Final flush
}

// Helper types for Loki API

type lokiStream struct {
	Stream map[string]string `json:"stream"`
	Values [][]string        `json:"values"`
}

type lokiPushRequest struct {
	Streams []*lokiStream `json:"streams"`
}

type lokiQueryResponse struct {
	Data struct {
		Result []struct {
			Stream map[string]string `json:"stream"`
			Values [][]string        `json:"values"`
		} `json:"result"`
	} `json:"data"`
}

// serializeLabels creates a consistent string key from labels
func serializeLabels(labels map[string]string) string {
	var buf bytes.Buffer
	buf.WriteString("{")
	first := true
	for k, v := range labels {
		if !first {
			buf.WriteString(",")
		}
		buf.WriteString(fmt.Sprintf("%s=\"%s\"", k, v))
		first = false
	}
	buf.WriteString("}")
	return buf.String()
}

// parseTimestamp parses a Loki timestamp string
func parseTimestamp(ts string) (time.Time, error) {
	nsec := int64(0)
	fmt.Sscanf(ts, "%d", &nsec)
	return time.Unix(0, nsec), nil
}
