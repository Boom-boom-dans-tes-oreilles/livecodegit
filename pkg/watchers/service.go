package watchers

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/livecodegit/pkg/core"
	"github.com/livecodegit/pkg/watchers/sonicpi"
	"github.com/livecodegit/pkg/watchers/tidal"
)

// WatcherService integrates watchers with the LiveCodeGit repository system
type WatcherService struct {
	manager       *WatcherManager
	configManager *ConfigManager
	repository    *core.LiveCodeRepository
	running       bool
	mutex         sync.RWMutex

	// Auto-commit configuration
	autoCommit        bool
	commitMessageTmpl *template.Template

	// Statistics
	totalExecutions int64
	totalCommits    int64
	lastExecution   time.Time
}

// NewWatcherService creates a new watcher service
func NewWatcherService(repo *core.LiveCodeRepository, configPath string) *WatcherService {
	manager := NewWatcherManager()
	configManager := NewConfigManager(configPath)

	service := &WatcherService{
		manager:       manager,
		configManager: configManager,
		repository:    repo,
		running:       false,
		autoCommit:    true,
	}

	// Set up the callback for execution events
	manager.SetCallback(service.handleExecutionEvent)

	return service
}

// Initialize loads configuration and sets up watchers
func (ws *WatcherService) Initialize() error {
	// Load configuration
	if err := ws.configManager.LoadConfig(); err != nil {
		return fmt.Errorf("failed to load watcher configuration: %w", err)
	}

	// Validate configuration
	if err := ws.configManager.ValidateConfig(); err != nil {
		return fmt.Errorf("invalid watcher configuration: %w", err)
	}

	// Set up commit message template
	config := ws.configManager.GetConfig()
	ws.autoCommit = config.AutoCommit

	tmpl, err := template.New("commit-message").Parse(config.CommitMessage)
	if err != nil {
		return fmt.Errorf("invalid commit message template: %w", err)
	}
	ws.commitMessageTmpl = tmpl

	// Register available watchers
	ws.registerWatchers()

	return nil
}

// registerWatchers creates and registers all configured watchers
func (ws *WatcherService) registerWatchers() {
	config := ws.configManager.GetConfig()

	for name, watcherConfig := range config.Watchers {
		var watcher ExecutionWatcher
		var err error

		switch name {
		case "sonicpi-osc":
			watcher, err = ws.createSonicPiOSCWatcher(watcherConfig)
		case "sonicpi-files":
			watcher, err = ws.createSonicPiFileWatcher(watcherConfig)
		case "tidal-ghci":
			watcher, err = ws.createTidalGHCiWatcher(watcherConfig)
		default:
			log.Printf("Unknown watcher type: %s", name)
			continue
		}

		if err != nil {
			log.Printf("Failed to create watcher %s: %v", name, err)
			continue
		}

		// Always register the watcher, but whether it starts depends on enabled status
		ws.manager.RegisterWatcher(name, watcher)
	}
}

// createSonicPiOSCWatcher creates a Sonic Pi OSC watcher
func (ws *WatcherService) createSonicPiOSCWatcher(config WatcherConfig) (ExecutionWatcher, error) {
	port := 4559 // Default Sonic Pi OSC port
	if portStr, exists := config.Options["osc_port"]; exists {
		// Parse port from string (simplified for now)
		if portStr == "4559" {
			port = 4559
		} else if portStr == "4560" {
			port = 4560
		}
		// In a real implementation, use strconv.Atoi
	}

	workspacePath := config.Options["workspace_path"]

	return sonicpi.NewOSCWatcher(port, workspacePath), nil
}

// createSonicPiFileWatcher creates a Sonic Pi file watcher
func (ws *WatcherService) createSonicPiFileWatcher(config WatcherConfig) (ExecutionWatcher, error) {
	workspacePath := config.Options["workspace_path"]
	if workspacePath == "" {
		return nil, fmt.Errorf("workspace_path is required for sonicpi-files watcher")
	}

	return sonicpi.NewFileWatcher(workspacePath), nil
}

// createTidalGHCiWatcher creates a TidalCycles GHCi watcher
func (ws *WatcherService) createTidalGHCiWatcher(config WatcherConfig) (ExecutionWatcher, error) {
	return tidal.NewGHCiWatcher(), nil
}

// Start starts all enabled watchers
func (ws *WatcherService) Start() error {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()

	if ws.running {
		return fmt.Errorf("watcher service is already running")
	}

	// Start only enabled watchers
	for _, name := range ws.configManager.GetEnabledWatchers() {
		if watcher, exists := ws.manager.GetWatcher(name); exists {
			if err := watcher.Start(ws.manager.callback); err != nil {
				return fmt.Errorf("failed to start watcher %s: %w", name, err)
			}
		}
	}

	ws.running = true
	log.Printf("Watcher service started with %d active watchers", len(ws.configManager.GetEnabledWatchers()))

	return nil
}

// Stop stops all running watchers
func (ws *WatcherService) Stop() error {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()

	if !ws.running {
		return nil
	}

	if err := ws.manager.StopAll(); err != nil {
		return fmt.Errorf("failed to stop watchers: %w", err)
	}

	ws.running = false
	log.Printf("Watcher service stopped")

	return nil
}

// IsRunning returns true if the service is running
func (ws *WatcherService) IsRunning() bool {
	ws.mutex.RLock()
	defer ws.mutex.RUnlock()
	return ws.running
}

// handleExecutionEvent processes execution events from watchers
func (ws *WatcherService) handleExecutionEvent(event ExecutionEvent) {
	ws.mutex.Lock()
	ws.totalExecutions++
	ws.lastExecution = event.Timestamp
	ws.mutex.Unlock()

	log.Printf("Execution detected: %s/%s - %s", event.Language, event.Buffer,
		truncateString(event.Content, 50))

	// Create auto-commit if enabled
	if ws.autoCommit {
		if err := ws.createAutoCommit(event); err != nil {
			log.Printf("Failed to create auto-commit: %v", err)
		} else {
			ws.mutex.Lock()
			ws.totalCommits++
			ws.mutex.Unlock()
		}
	}
}

// createAutoCommit creates a commit from an execution event
func (ws *WatcherService) createAutoCommit(event ExecutionEvent) error {
	// Generate commit message from template
	commitMessage, err := ws.generateCommitMessage(event)
	if err != nil {
		return fmt.Errorf("failed to generate commit message: %w", err)
	}

	// Convert event to metadata
	metadata := event.ToExecutionMetadata()

	// Create commit
	_, err = ws.repository.Commit(event.Content, commitMessage, metadata)
	if err != nil {
		return fmt.Errorf("failed to create commit: %w", err)
	}

	return nil
}

// generateCommitMessage generates a commit message from template and event
func (ws *WatcherService) generateCommitMessage(event ExecutionEvent) (string, error) {
	var buf strings.Builder

	// Create template data
	data := struct {
		Language    string
		Environment string
		Buffer      string
		Timestamp   string
		Success     string
	}{
		Language:    event.Language,
		Environment: event.Environment,
		Buffer:      event.Buffer,
		Timestamp:   event.Timestamp.Format("15:04:05"),
		Success: func() string {
			if event.Success {
				return "success"
			}
			return "error"
		}(),
	}

	if err := ws.commitMessageTmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// GetStats returns service statistics
func (ws *WatcherService) GetStats() ServiceStats {
	ws.mutex.RLock()
	defer ws.mutex.RUnlock()

	return ServiceStats{
		TotalExecutions: ws.totalExecutions,
		TotalCommits:    ws.totalCommits,
		LastExecution:   ws.lastExecution,
		ActiveWatchers:  len(ws.configManager.GetEnabledWatchers()),
		Running:         ws.running,
	}
}

// GetEnabledWatchers returns names of enabled watchers
func (ws *WatcherService) GetEnabledWatchers() []string {
	return ws.configManager.GetEnabledWatchers()
}

// EnableWatcher enables a specific watcher
func (ws *WatcherService) EnableWatcher(name string) error {
	if err := ws.configManager.EnableWatcher(name); err != nil {
		return err
	}

	// Save configuration
	return ws.configManager.SaveConfig()
}

// DisableWatcher disables a specific watcher
func (ws *WatcherService) DisableWatcher(name string) error {
	// Stop the watcher if it's running
	if ws.running {
		if watcher, exists := ws.manager.GetWatcher(name); exists && watcher.IsRunning() {
			if err := watcher.Stop(); err != nil {
				return fmt.Errorf("failed to stop watcher: %w", err)
			}
		}
	}

	if err := ws.configManager.DisableWatcher(name); err != nil {
		return err
	}

	// Save configuration
	return ws.configManager.SaveConfig()
}

// UpdateWatcherConfig updates configuration for a specific watcher
func (ws *WatcherService) UpdateWatcherConfig(name string, config WatcherConfig) error {
	ws.configManager.SetWatcherConfig(name, config)
	return ws.configManager.SaveConfig()
}

// GetWatcherConfig returns configuration for a specific watcher
func (ws *WatcherService) GetWatcherConfig(name string) (WatcherConfig, bool) {
	return ws.configManager.GetWatcherConfig(name)
}

// ServiceStats holds statistics about the watcher service
type ServiceStats struct {
	TotalExecutions int64     `json:"total_executions"`
	TotalCommits    int64     `json:"total_commits"`
	LastExecution   time.Time `json:"last_execution"`
	ActiveWatchers  int       `json:"active_watchers"`
	Running         bool      `json:"running"`
}

// truncateString truncates a string to a maximum length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
