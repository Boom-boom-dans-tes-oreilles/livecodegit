package common

import (
	"time"

	"github.com/livecodegit/pkg/storage"
)

// ExecutionEvent represents a code execution detected by a watcher
type ExecutionEvent struct {
	Timestamp   time.Time `json:"timestamp"`
	Content     string    `json:"content"`
	Buffer      string    `json:"buffer"`
	Language    string    `json:"language"`
	Environment string    `json:"environment"`
	Success     bool      `json:"success"`
	ErrorMessage string   `json:"error_message,omitempty"`
	
	// Music-specific metadata
	BPM            float64 `json:"bpm,omitempty"`
	BeatsFromStart int64   `json:"beats_from_start,omitempty"`
	
	// File-specific metadata
	FilePath     string `json:"file_path,omitempty"`
	LineNumber   int    `json:"line_number,omitempty"`
	
	// Environment-specific metadata
	ProcessID    int               `json:"process_id,omitempty"`
	ExtraData    map[string]string `json:"extra_data,omitempty"`
}

// WatcherConfig holds configuration for a watcher
type WatcherConfig struct {
	Language    string            `json:"language"`
	Environment string            `json:"environment"`
	Enabled     bool              `json:"enabled"`
	Options     map[string]string `json:"options"`
}

// ExecutionWatcher defines the interface for detecting code executions
type ExecutionWatcher interface {
	// Start begins watching for executions and calls the callback for each event
	Start(callback func(event ExecutionEvent)) error
	
	// Stop stops the watcher
	Stop() error
	
	// IsRunning returns true if the watcher is currently active
	IsRunning() bool
	
	// GetConfig returns the watcher's configuration
	GetConfig() WatcherConfig
	
	// GetLanguage returns the programming language this watcher monitors
	GetLanguage() string
	
	// GetEnvironment returns the environment name (e.g., "sonic-pi", "tidal-cycles")
	GetEnvironment() string
}

// ToExecutionMetadata converts an ExecutionEvent to storage.ExecutionMetadata
func (event ExecutionEvent) ToExecutionMetadata() storage.ExecutionMetadata {
	return storage.ExecutionMetadata{
		Buffer:         event.Buffer,
		Language:       event.Language,
		BPM:            event.BPM,
		BeatsFromStart: event.BeatsFromStart,
		Success:        event.Success,
		ErrorMessage:   event.ErrorMessage,
		Environment:    event.Environment,
	}
}