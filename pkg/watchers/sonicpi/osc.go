package sonicpi

import (
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/livecodegit/pkg/watchers/common"
)

// OSCWatcher monitors Sonic Pi's OSC messages for code execution events
type OSCWatcher struct {
	config   common.WatcherConfig
	conn     *net.UDPConn
	running  bool
	mutex    sync.RWMutex
	callback func(common.ExecutionEvent)
	
	// Sonic Pi specific settings
	oscPort      int
	workspacePath string
	currentBPM   float64
	startTime    time.Time
}

// NewOSCWatcher creates a new Sonic Pi OSC watcher
func NewOSCWatcher(port int, workspacePath string) *OSCWatcher {
	return &OSCWatcher{
		config: common.WatcherConfig{
			Language:    "sonicpi",
			Environment: "sonic-pi",
			Enabled:     true,
			Options: map[string]string{
				"osc_port":       strconv.Itoa(port),
				"workspace_path": workspacePath,
			},
		},
		oscPort:       port,
		workspacePath: workspacePath,
		currentBPM:    120.0, // Default BPM
		running:       false,
	}
}

// Start begins monitoring OSC messages from Sonic Pi
func (w *OSCWatcher) Start(callback func(common.ExecutionEvent)) error {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	
	if w.running {
		return fmt.Errorf("watcher is already running")
	}
	
	w.callback = callback
	w.startTime = time.Now()
	
	// Listen for OSC messages on UDP
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", w.oscPort))
	if err != nil {
		return fmt.Errorf("failed to resolve UDP address: %w", err)
	}
	
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on UDP port %d: %w", w.oscPort, err)
	}
	
	w.conn = conn
	w.running = true
	
	// Start listening for messages in a goroutine
	go w.listenForMessages()
	
	return nil
}

// Stop stops the OSC watcher
func (w *OSCWatcher) Stop() error {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	
	if !w.running {
		return nil
	}
	
	w.running = false
	
	if w.conn != nil {
		return w.conn.Close()
	}
	
	return nil
}

// IsRunning returns true if the watcher is active
func (w *OSCWatcher) IsRunning() bool {
	w.mutex.RLock()
	defer w.mutex.RUnlock()
	return w.running
}

// GetConfig returns the watcher configuration
func (w *OSCWatcher) GetConfig() common.WatcherConfig {
	return w.config
}

// GetLanguage returns "sonicpi"
func (w *OSCWatcher) GetLanguage() string {
	return "sonicpi"
}

// GetEnvironment returns "sonic-pi"
func (w *OSCWatcher) GetEnvironment() string {
	return "sonic-pi"
}

// listenForMessages continuously listens for OSC messages
func (w *OSCWatcher) listenForMessages() {
	buffer := make([]byte, 4096)
	
	for w.IsRunning() {
		w.conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, err := w.conn.Read(buffer)
		
		if err != nil {
			if netError, ok := err.(net.Error); ok && netError.Timeout() {
				continue // Timeout is expected, continue listening
			}
			if w.IsRunning() {
				fmt.Printf("Error reading OSC message: %v\n", err)
			}
			continue
		}
		
		message := string(buffer[:n])
		w.processOSCMessage(message)
	}
}

// processOSCMessage parses and handles incoming OSC messages
func (w *OSCWatcher) processOSCMessage(message string) {
	// Sonic Pi OSC messages for execution events typically look like:
	// "/run-code" followed by parameters
	// "/error" for errors
	// "/info" for info messages
	
	lines := strings.Split(message, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		if w.isExecutionMessage(line) {
			event := w.parseExecutionEvent(line)
			if w.callback != nil {
				w.callback(event)
			}
		} else if w.isBPMMessage(line) {
			w.updateBPM(line)
		}
	}
}

// isExecutionMessage checks if the message indicates code execution
func (w *OSCWatcher) isExecutionMessage(message string) bool {
	executionPatterns := []string{
		"/run-code",
		"/stop-all",
		"/start-recording",
		"/buffer-update",
	}
	
	for _, pattern := range executionPatterns {
		if strings.Contains(message, pattern) {
			return true
		}
	}
	
	return false
}

// isBPMMessage checks if the message contains BPM information
func (w *OSCWatcher) isBPMMessage(message string) bool {
	return strings.Contains(message, "/bpm") || strings.Contains(message, "use_bpm")
}

// updateBPM extracts and updates the current BPM from OSC messages
func (w *OSCWatcher) updateBPM(message string) {
	// Look for BPM values in the message
	bpmRegex := regexp.MustCompile(`(?:bpm|BPM)[\s:=]*(\d+(?:\.\d+)?)`)
	matches := bpmRegex.FindStringSubmatch(message)
	
	if len(matches) > 1 {
		if bpm, err := strconv.ParseFloat(matches[1], 64); err == nil {
			w.currentBPM = bpm
		}
	}
}

// parseExecutionEvent creates an ExecutionEvent from an OSC message
func (w *OSCWatcher) parseExecutionEvent(message string) common.ExecutionEvent {
	now := time.Now()
	
	// Extract buffer name if present
	buffer := "workspace-0" // Default buffer
	bufferRegex := regexp.MustCompile(`buffer[:\s]+(\w+)`)
	if matches := bufferRegex.FindStringSubmatch(message); len(matches) > 1 {
		buffer = matches[1]
	}
	
	// Determine if this was a successful execution
	success := !strings.Contains(message, "/error")
	errorMessage := ""
	if !success {
		errorMessage = w.extractErrorMessage(message)
	}
	
	// Calculate beats from start
	beatsFromStart := w.calculateBeatsFromStart(now)
	
	// Try to read current buffer content
	content := w.readBufferContent(buffer)
	
	return common.ExecutionEvent{
		Timestamp:      now,
		Content:        content,
		Buffer:         buffer,
		Language:       "sonicpi",
		Environment:    "sonic-pi",
		Success:        success,
		ErrorMessage:   errorMessage,
		BPM:            w.currentBPM,
		BeatsFromStart: beatsFromStart,
		ExtraData: map[string]string{
			"osc_message": message,
		},
	}
}

// extractErrorMessage extracts error information from OSC error messages
func (w *OSCWatcher) extractErrorMessage(message string) string {
	// Simple error extraction - in a real implementation, this would be more sophisticated
	if strings.Contains(message, "/error") {
		parts := strings.Split(message, "/error")
		if len(parts) > 1 {
			return strings.TrimSpace(parts[1])
		}
	}
	return "Unknown error"
}

// calculateBeatsFromStart calculates how many beats have passed since start
func (w *OSCWatcher) calculateBeatsFromStart(timestamp time.Time) int64 {
	elapsed := timestamp.Sub(w.startTime)
	beatsPerSecond := w.currentBPM / 60.0
	totalBeats := elapsed.Seconds() * beatsPerSecond
	return int64(totalBeats)
}

// readBufferContent attempts to read the current content of a Sonic Pi buffer
func (w *OSCWatcher) readBufferContent(bufferName string) string {
	// In a real implementation, this would read from Sonic Pi's workspace files
	// For now, return a placeholder that indicates we detected execution
	if w.workspacePath == "" {
		return fmt.Sprintf("# Code executed in buffer: %s\n# (content not available without workspace path)", bufferName)
	}
	
	// For now, return a simple placeholder
	// TODO: Implement actual file reading when workspace path is available
	// Sonic Pi typically saves workspace content in files named like "workspace_0", etc.
	return fmt.Sprintf("# Executed at %s\n# Buffer: %s", time.Now().Format("15:04:05"), bufferName)
}