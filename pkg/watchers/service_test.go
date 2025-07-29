package watchers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/livecodegit/pkg/core"
)

func createTestRepository(t *testing.T) *core.LiveCodeRepository {
	tempDir, err := os.MkdirTemp("", "livecodegit-service-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	repo := core.NewRepository(tempDir)
	if err := repo.Init(tempDir); err != nil {
		t.Fatalf("Failed to initialize test repository: %v", err)
	}

	return repo
}

func createTestWatcherService(t *testing.T) (*WatcherService, string) {
	repo := createTestRepository(t)

	configPath := createTempConfigFile(t)
	service := NewWatcherService(repo, configPath)

	return service, filepath.Dir(configPath)
}

func TestNewWatcherService(t *testing.T) {
	service, tempDir := createTestWatcherService(t)
	defer os.RemoveAll(tempDir)

	if service == nil {
		t.Fatalf("Expected non-nil watcher service")
	}

	if service.manager == nil {
		t.Errorf("Expected non-nil watcher manager")
	}

	if service.configManager == nil {
		t.Errorf("Expected non-nil config manager")
	}

	if service.repository == nil {
		t.Errorf("Expected non-nil repository")
	}

	if service.running {
		t.Errorf("Expected service to not be running initially")
	}
}

func TestWatcherServiceInitialize(t *testing.T) {
	service, tempDir := createTestWatcherService(t)
	defer os.RemoveAll(tempDir)

	err := service.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize watcher service: %v", err)
	}

	// Check that commit message template was set
	if service.commitMessageTmpl == nil {
		t.Errorf("Expected commit message template to be set")
	}

	// Check that auto-commit is enabled by default
	if !service.autoCommit {
		t.Errorf("Expected auto-commit to be enabled by default")
	}

	// Check that watchers were registered
	watchers := service.manager.ListWatchers()
	if len(watchers) == 0 {
		t.Errorf("Expected watchers to be registered during initialization")
	}
}

func TestWatcherServiceStartStop(t *testing.T) {
	service, tempDir := createTestWatcherService(t)
	defer os.RemoveAll(tempDir)

	err := service.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize service: %v", err)
	}

	// Should not be running initially
	if service.IsRunning() {
		t.Errorf("Expected service to not be running initially")
	}

	// Disable all real watchers that might have dependencies (before registration)
	config := service.configManager.GetConfig()
	for name, watcherConfig := range config.Watchers {
		watcherConfig.Enabled = false
		service.configManager.SetWatcherConfig(name, watcherConfig)
	}

	// Enable a mock watcher for testing
	mockWatcher := &MockWatcher{
		config: WatcherConfig{
			Language:    "test",
			Environment: "test-env",
			Enabled:     true,
		},
	}
	service.manager.RegisterWatcher("mock-watcher", mockWatcher)
	service.configManager.SetWatcherConfig("mock-watcher", mockWatcher.config)

	// Start service
	err = service.Start()
	if err != nil {
		t.Fatalf("Failed to start service: %v", err)
	}

	if !service.IsRunning() {
		t.Errorf("Expected service to be running after start")
	}

	// Stop service
	err = service.Stop()
	if err != nil {
		t.Fatalf("Failed to stop service: %v", err)
	}

	if service.IsRunning() {
		t.Errorf("Expected service to not be running after stop")
	}
}

func TestWatcherServiceHandleExecutionEvent(t *testing.T) {
	service, tempDir := createTestWatcherService(t)
	defer os.RemoveAll(tempDir)

	err := service.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize service: %v", err)
	}

	// Get initial stats
	initialStats := service.GetStats()
	if initialStats.TotalExecutions != 0 {
		t.Errorf("Expected 0 initial executions, got %d", initialStats.TotalExecutions)
	}

	// Create test execution event
	event := ExecutionEvent{
		Timestamp:   time.Now(),
		Content:     "test code",
		Buffer:      "test-buffer",
		Language:    "sonicpi",
		Environment: "sonic-pi",
		Success:     true,
	}

	// Handle the event
	service.handleExecutionEvent(event)

	// Check updated stats
	stats := service.GetStats()
	if stats.TotalExecutions != 1 {
		t.Errorf("Expected 1 execution after handling event, got %d", stats.TotalExecutions)
	}

	if stats.TotalCommits != 1 {
		t.Errorf("Expected 1 commit after handling event, got %d", stats.TotalCommits)
	}

	if stats.LastExecution.IsZero() {
		t.Errorf("Expected last execution time to be set")
	}
}

func TestWatcherServiceAutoCommitDisabled(t *testing.T) {
	service, tempDir := createTestWatcherService(t)
	defer os.RemoveAll(tempDir)

	err := service.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize service: %v", err)
	}

	// Disable auto-commit
	service.autoCommit = false

	// Create test execution event
	event := ExecutionEvent{
		Timestamp:   time.Now(),
		Content:     "test code",
		Buffer:      "test-buffer",
		Language:    "sonicpi",
		Environment: "sonic-pi",
		Success:     true,
	}

	// Handle the event
	service.handleExecutionEvent(event)

	// Check stats - execution should be counted but no commit should be created
	stats := service.GetStats()
	if stats.TotalExecutions != 1 {
		t.Errorf("Expected 1 execution after handling event, got %d", stats.TotalExecutions)
	}

	if stats.TotalCommits != 0 {
		t.Errorf("Expected 0 commits with auto-commit disabled, got %d", stats.TotalCommits)
	}
}

func TestWatcherServiceGenerateCommitMessage(t *testing.T) {
	service, tempDir := createTestWatcherService(t)
	defer os.RemoveAll(tempDir)

	err := service.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize service: %v", err)
	}

	event := ExecutionEvent{
		Timestamp:   time.Now(),
		Content:     "test code",
		Buffer:      "drums",
		Language:    "sonicpi",
		Environment: "sonic-pi",
		Success:     true,
	}

	message, err := service.generateCommitMessage(event)
	if err != nil {
		t.Fatalf("Failed to generate commit message: %v", err)
	}

	// Check that message contains expected elements
	if message == "" {
		t.Errorf("Expected non-empty commit message")
	}

	// The default template should include language and buffer
	expectedSubstrings := []string{"sonicpi", "drums"}
	for _, expected := range expectedSubstrings {
		if !strings.Contains(message, expected) {
			t.Errorf("Expected commit message to contain '%s', got: %s", expected, message)
		}
	}
}

func TestWatcherServiceGetEnabledWatchers(t *testing.T) {
	service, tempDir := createTestWatcherService(t)
	defer os.RemoveAll(tempDir)

	err := service.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize service: %v", err)
	}

	// Initially no watchers should be enabled
	enabled := service.GetEnabledWatchers()
	if len(enabled) != 0 {
		t.Errorf("Expected 0 enabled watchers initially, got %d", len(enabled))
	}

	// Enable a watcher
	err = service.EnableWatcher("sonicpi-osc")
	if err != nil {
		t.Fatalf("Failed to enable watcher: %v", err)
	}

	enabled = service.GetEnabledWatchers()
	if len(enabled) != 1 {
		t.Errorf("Expected 1 enabled watcher, got %d", len(enabled))
	}

	if enabled[0] != "sonicpi-osc" {
		t.Errorf("Expected enabled watcher 'sonicpi-osc', got '%s'", enabled[0])
	}
}

func TestWatcherServiceWatcherConfigOperations(t *testing.T) {
	service, tempDir := createTestWatcherService(t)
	defer os.RemoveAll(tempDir)

	err := service.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize service: %v", err)
	}

	// Test GetWatcherConfig
	config, exists := service.GetWatcherConfig("sonicpi-osc")
	if !exists {
		t.Errorf("Expected to find sonicpi-osc watcher config")
	}

	if config.Language != "sonicpi" {
		t.Errorf("Expected language 'sonicpi', got '%s'", config.Language)
	}

	// Test UpdateWatcherConfig
	newConfig := WatcherConfig{
		Language:    "test",
		Environment: "test-env",
		Enabled:     true,
		Options: map[string]string{
			"test_option": "test_value",
		},
	}

	err = service.UpdateWatcherConfig("test-watcher", newConfig)
	if err != nil {
		t.Fatalf("Failed to update watcher config: %v", err)
	}

	retrievedConfig, exists := service.GetWatcherConfig("test-watcher")
	if !exists {
		t.Errorf("Expected to find updated watcher config")
	}

	if retrievedConfig.Language != "test" {
		t.Errorf("Expected language 'test', got '%s'", retrievedConfig.Language)
	}
}

func TestWatcherServiceDisableWatcher(t *testing.T) {
	service, tempDir := createTestWatcherService(t)
	defer os.RemoveAll(tempDir)

	err := service.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize service: %v", err)
	}

	// Enable then disable a watcher
	err = service.EnableWatcher("sonicpi-osc")
	if err != nil {
		t.Fatalf("Failed to enable watcher: %v", err)
	}

	enabled := service.GetEnabledWatchers()
	if len(enabled) != 1 {
		t.Errorf("Expected 1 enabled watcher, got %d", len(enabled))
	}

	err = service.DisableWatcher("sonicpi-osc")
	if err != nil {
		t.Fatalf("Failed to disable watcher: %v", err)
	}

	enabled = service.GetEnabledWatchers()
	if len(enabled) != 0 {
		t.Errorf("Expected 0 enabled watchers after disable, got %d", len(enabled))
	}
}

func TestWatcherServiceStats(t *testing.T) {
	service, tempDir := createTestWatcherService(t)
	defer os.RemoveAll(tempDir)

	err := service.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize service: %v", err)
	}

	// Get initial stats
	stats := service.GetStats()

	if stats.TotalExecutions != 0 {
		t.Errorf("Expected 0 initial executions, got %d", stats.TotalExecutions)
	}

	if stats.TotalCommits != 0 {
		t.Errorf("Expected 0 initial commits, got %d", stats.TotalCommits)
	}

	if stats.Running {
		t.Errorf("Expected service to not be running initially")
	}

	if stats.ActiveWatchers != 0 {
		t.Errorf("Expected 0 active watchers initially, got %d", stats.ActiveWatchers)
	}

	if !stats.LastExecution.IsZero() {
		t.Errorf("Expected last execution to be zero initially")
	}

	// Enable a watcher to increase active count
	service.EnableWatcher("sonicpi-osc")
	stats = service.GetStats()

	if stats.ActiveWatchers != 1 {
		t.Errorf("Expected 1 active watcher after enabling, got %d", stats.ActiveWatchers)
	}
}
