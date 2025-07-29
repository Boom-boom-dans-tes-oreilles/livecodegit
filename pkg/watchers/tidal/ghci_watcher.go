package tidal

import (
	"bufio"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/livecodegit/pkg/watchers/common"
)

// GHCiWatcher monitors TidalCycles through GHCi interaction
type GHCiWatcher struct {
	config   common.WatcherConfig
	running  bool
	mutex    sync.RWMutex
	callback func(common.ExecutionEvent)

	// GHCi process management
	cmd    *exec.Cmd
	stdin  *bufio.Writer
	stdout *bufio.Reader
	stderr *bufio.Reader

	// Tidal-specific state
	currentCPS  float64
	startTime   time.Time
	connections map[string]string // Track active connections (d1, d2, etc.)

	// Pattern tracking
	lastPatterns map[string]string
}

// NewGHCiWatcher creates a new TidalCycles GHCi watcher
func NewGHCiWatcher() *GHCiWatcher {
	return &GHCiWatcher{
		config: common.WatcherConfig{
			Language:    "tidal",
			Environment: "tidal-cycles",
			Enabled:     true,
			Options: map[string]string{
				"ghci_command": "ghci",
				"boot_file":    "BootTidal.hs",
			},
		},
		running:      false,
		currentCPS:   0.5625, // Default Tidal CPS
		connections:  make(map[string]string),
		lastPatterns: make(map[string]string),
	}
}

// Start begins monitoring TidalCycles through GHCi
func (w *GHCiWatcher) Start(callback func(common.ExecutionEvent)) error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if w.running {
		return fmt.Errorf("GHCi watcher is already running")
	}

	w.callback = callback
	w.startTime = time.Now()

	// Start GHCi process
	ghciCmd := w.config.Options["ghci_command"]
	w.cmd = exec.Command(ghciCmd)

	// Set up pipes for communication
	stdin, err := w.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := w.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := w.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	w.stdin = bufio.NewWriter(stdin)
	w.stdout = bufio.NewReader(stdout)
	w.stderr = bufio.NewReader(stderr)

	// Start GHCi
	if err := w.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start GHCi: %w", err)
	}

	w.running = true

	// Initialize Tidal in separate goroutine
	go w.initializeTidal()

	// Start monitoring output
	go w.monitorOutput()
	go w.monitorErrors()

	return nil
}

// Stop stops the GHCi watcher
func (w *GHCiWatcher) Stop() error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if !w.running {
		return nil
	}

	w.running = false

	// Send quit command to GHCi
	if w.stdin != nil {
		w.stdin.WriteString(":quit\n")
		w.stdin.Flush()
	}

	// Kill the process if it doesn't exit gracefully
	if w.cmd != nil && w.cmd.Process != nil {
		w.cmd.Process.Kill()
		w.cmd.Wait()
	}

	return nil
}

// IsRunning returns true if the watcher is active
func (w *GHCiWatcher) IsRunning() bool {
	w.mutex.RLock()
	defer w.mutex.RUnlock()
	return w.running
}

// GetConfig returns the watcher configuration
func (w *GHCiWatcher) GetConfig() common.WatcherConfig {
	return w.config
}

// GetLanguage returns "tidal"
func (w *GHCiWatcher) GetLanguage() string {
	return "tidal"
}

// GetEnvironment returns "tidal-cycles"
func (w *GHCiWatcher) GetEnvironment() string {
	return "tidal-cycles"
}

// initializeTidal sends initialization commands to set up TidalCycles
func (w *GHCiWatcher) initializeTidal() {
	// Wait a bit for GHCi to start
	time.Sleep(1 * time.Second)

	initCommands := []string{
		":set -XOverloadedStrings",
		":set prompt \"tidal> \"",
		"import Sound.Tidal.Context",
		"(cps, nudger, d1, d2, d3, d4, d5, d6, d7, d8, d9) <- dirtStream",
		"let bps x = cps (x/4)",
		"let hush = mapM_ ($ silence) [d1,d2,d3,d4,d5,d6,d7,d8,d9]",
	}

	for _, cmd := range initCommands {
		w.sendCommand(cmd)
		time.Sleep(100 * time.Millisecond) // Small delay between commands
	}
}

// sendCommand sends a command to GHCi
func (w *GHCiWatcher) sendCommand(command string) error {
	if w.stdin == nil {
		return fmt.Errorf("GHCi stdin not available")
	}

	_, err := w.stdin.WriteString(command + "\n")
	if err != nil {
		return err
	}

	return w.stdin.Flush()
}

// monitorOutput monitors GHCi stdout for execution events
func (w *GHCiWatcher) monitorOutput() {
	scanner := bufio.NewScanner(w.stdout)

	for scanner.Scan() && w.IsRunning() {
		line := scanner.Text()
		w.processOutputLine(line)
	}
}

// monitorErrors monitors GHCi stderr for error messages
func (w *GHCiWatcher) monitorErrors() {
	scanner := bufio.NewScanner(w.stderr)

	for scanner.Scan() && w.IsRunning() {
		line := scanner.Text()
		w.processErrorLine(line)
	}
}

// processOutputLine analyzes GHCi output for execution events
func (w *GHCiWatcher) processOutputLine(line string) {
	line = strings.TrimSpace(line)

	// Skip empty lines and prompts
	if line == "" || strings.HasPrefix(line, "tidal>") {
		return
	}

	// Check for pattern evaluations
	if w.isPatternEvaluation(line) {
		event := w.createPatternExecutionEvent(line, true, "")
		if w.callback != nil {
			w.callback(event)
		}
	}

	// Check for CPS changes
	if w.isCPSChange(line) {
		w.updateCPS(line)
	}
}

// processErrorLine analyzes GHCi errors
func (w *GHCiWatcher) processErrorLine(line string) {
	line = strings.TrimSpace(line)

	if line == "" {
		return
	}

	// Create error event
	event := w.createPatternExecutionEvent(line, false, line)
	if w.callback != nil {
		w.callback(event)
	}
}

// isPatternEvaluation checks if the line indicates a pattern was evaluated
func (w *GHCiWatcher) isPatternEvaluation(line string) bool {
	// TidalCycles pattern indicators
	patterns := []string{
		"d1 $",
		"d2 $",
		"d3 $",
		"d4 $",
		"d5 $",
		"d6 $",
		"d7 $",
		"d8 $",
		"d9 $",
		"hush",
		"silence",
	}

	for _, pattern := range patterns {
		if strings.Contains(line, pattern) {
			return true
		}
	}

	return false
}

// isCPSChange checks if the line indicates a CPS (cycles per second) change
func (w *GHCiWatcher) isCPSChange(line string) bool {
	return strings.Contains(line, "cps") || strings.Contains(line, "bps")
}

// updateCPS extracts and updates the current CPS from output
func (w *GHCiWatcher) updateCPS(line string) {
	// Look for CPS values in the line
	cpsRegex := regexp.MustCompile(`(?:cps|bps)\s*\(?\s*(\d+(?:\.\d+)?)\s*\)?`)
	matches := cpsRegex.FindStringSubmatch(line)

	if len(matches) > 1 {
		if cps, err := strconv.ParseFloat(matches[1], 64); err == nil {
			// If it's BPS, convert to CPS
			if strings.Contains(line, "bps") {
				cps = cps / 4.0
			}
			w.currentCPS = cps
		}
	}
}

// createPatternExecutionEvent creates an execution event for Tidal patterns
func (w *GHCiWatcher) createPatternExecutionEvent(content string, success bool, errorMessage string) common.ExecutionEvent {
	now := time.Now()

	// Extract connection (d1, d2, etc.) from content
	connection := w.extractConnection(content)

	// Calculate cycles from start
	cyclesFromStart := w.calculateCyclesFromStart(now)

	// Store the pattern for this connection
	if success && connection != "" {
		w.lastPatterns[connection] = content
	}

	return common.ExecutionEvent{
		Timestamp:      now,
		Content:        content,
		Buffer:         connection,
		Language:       "tidal",
		Environment:    "tidal-cycles",
		Success:        success,
		ErrorMessage:   errorMessage,
		BPM:            w.currentCPS * 60,          // Convert CPS to BPM approximation
		BeatsFromStart: int64(cyclesFromStart * 4), // Convert cycles to beats
		ExtraData: map[string]string{
			"connection": connection,
			"cps":        fmt.Sprintf("%.4f", w.currentCPS),
		},
	}
}

// extractConnection extracts the connection name (d1, d2, etc.) from Tidal code
func (w *GHCiWatcher) extractConnection(content string) string {
	// Look for d1, d2, etc. in the content
	connectionRegex := regexp.MustCompile(`\b(d\d+)\b`)
	matches := connectionRegex.FindStringSubmatch(content)

	if len(matches) > 1 {
		return matches[1]
	}

	// Check for special commands
	if strings.Contains(content, "hush") {
		return "all"
	}

	return "unknown"
}

// calculateCyclesFromStart calculates how many Tidal cycles have passed since start
func (w *GHCiWatcher) calculateCyclesFromStart(timestamp time.Time) float64 {
	elapsed := timestamp.Sub(w.startTime)
	cyclesPerSecond := w.currentCPS
	totalCycles := elapsed.Seconds() * cyclesPerSecond
	return totalCycles
}

// ExecutePattern sends a pattern to TidalCycles for execution
func (w *GHCiWatcher) ExecutePattern(pattern string) error {
	if !w.IsRunning() {
		return fmt.Errorf("watcher is not running")
	}

	return w.sendCommand(pattern)
}

// GetActivePatterns returns the currently active patterns
func (w *GHCiWatcher) GetActivePatterns() map[string]string {
	w.mutex.RLock()
	defer w.mutex.RUnlock()

	patterns := make(map[string]string)
	for k, v := range w.lastPatterns {
		patterns[k] = v
	}

	return patterns
}

// Hush stops all active patterns (equivalent to Tidal's hush command)
func (w *GHCiWatcher) Hush() error {
	return w.ExecutePattern("hush")
}
