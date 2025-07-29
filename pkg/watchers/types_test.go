package watchers

import (
	"testing"
	"time"
)

func TestExecutionEvent(t *testing.T) {
	event := ExecutionEvent{
		Timestamp:      time.Now(),
		Content:        "live_loop :drums do\n  sample :bd_haus\nend",
		Buffer:         "drums",
		Language:       "sonicpi",
		Environment:    "sonic-pi",
		Success:        true,
		BPM:            120.0,
		BeatsFromStart: 16,
		FilePath:       "/path/to/file.rb",
		LineNumber:     1,
		ProcessID:      12345,
		ExtraData: map[string]string{
			"test_key": "test_value",
		},
	}

	if event.Language != "sonicpi" {
		t.Errorf("Expected language 'sonicpi', got '%s'", event.Language)
	}

	if event.BPM != 120.0 {
		t.Errorf("Expected BPM 120.0, got %f", event.BPM)
	}

	if event.ExtraData["test_key"] != "test_value" {
		t.Errorf("Expected extra data 'test_value', got '%s'", event.ExtraData["test_key"])
	}
}

func TestExecutionEventToExecutionMetadata(t *testing.T) {
	event := ExecutionEvent{
		Buffer:         "main",
		Language:       "sonicpi",
		BPM:            130.0,
		BeatsFromStart: 24,
		Success:        true,
		ErrorMessage:   "",
		Environment:    "sonic-pi",
	}

	metadata := event.ToExecutionMetadata()

	if metadata.Buffer != event.Buffer {
		t.Errorf("Expected buffer '%s', got '%s'", event.Buffer, metadata.Buffer)
	}

	if metadata.Language != event.Language {
		t.Errorf("Expected language '%s', got '%s'", event.Language, metadata.Language)
	}

	if metadata.BPM != event.BPM {
		t.Errorf("Expected BPM %f, got %f", event.BPM, metadata.BPM)
	}

	if metadata.BeatsFromStart != event.BeatsFromStart {
		t.Errorf("Expected beats from start %d, got %d", event.BeatsFromStart, metadata.BeatsFromStart)
	}

	if metadata.Success != event.Success {
		t.Errorf("Expected success %t, got %t", event.Success, metadata.Success)
	}

	if metadata.Environment != event.Environment {
		t.Errorf("Expected environment '%s', got '%s'", event.Environment, metadata.Environment)
	}
}

func TestWatcherConfig(t *testing.T) {
	config := WatcherConfig{
		Language:    "tidal",
		Environment: "tidal-cycles",
		Enabled:     true,
		Options: map[string]string{
			"ghci_command": "ghci",
			"boot_file":    "BootTidal.hs",
		},
	}

	if config.Language != "tidal" {
		t.Errorf("Expected language 'tidal', got '%s'", config.Language)
	}

	if !config.Enabled {
		t.Errorf("Expected config to be enabled")
	}

	if config.Options["ghci_command"] != "ghci" {
		t.Errorf("Expected ghci_command 'ghci', got '%s'", config.Options["ghci_command"])
	}
}

func TestNewWatcherManager(t *testing.T) {
	manager := NewWatcherManager()

	if manager == nil {
		t.Fatalf("Expected non-nil watcher manager")
	}

	if manager.watchers == nil {
		t.Errorf("Expected non-nil watchers map")
	}

	if manager.running {
		t.Errorf("Expected manager to not be running initially")
	}

	if len(manager.ListWatchers()) != 0 {
		t.Errorf("Expected empty watcher list initially")
	}
}

func TestWatcherManagerRegisterWatcher(t *testing.T) {
	manager := NewWatcherManager()

	// Create a mock watcher
	mockWatcher := &MockWatcher{
		config: WatcherConfig{
			Language:    "test",
			Environment: "test-env",
			Enabled:     true,
		},
	}

	manager.RegisterWatcher("test-watcher", mockWatcher)

	watchers := manager.ListWatchers()
	if len(watchers) != 1 {
		t.Errorf("Expected 1 watcher, got %d", len(watchers))
	}

	if watchers[0] != "test-watcher" {
		t.Errorf("Expected watcher name 'test-watcher', got '%s'", watchers[0])
	}

	retrieved, exists := manager.GetWatcher("test-watcher")
	if !exists {
		t.Errorf("Expected to find registered watcher")
	}

	if retrieved != mockWatcher {
		t.Errorf("Expected to get back the same watcher instance")
	}
}

func TestWatcherManagerCallback(t *testing.T) {
	manager := NewWatcherManager()

	callbackCalled := false
	var receivedEvent ExecutionEvent

	callback := func(event ExecutionEvent) {
		callbackCalled = true
		receivedEvent = event
	}

	manager.SetCallback(callback)

	// Test that callback is set
	if manager.callback == nil {
		t.Errorf("Expected callback to be set")
	}

	// Simulate calling the callback
	testEvent := ExecutionEvent{
		Language: "test",
		Buffer:   "test-buffer",
		Success:  true,
	}

	manager.callback(testEvent)

	if !callbackCalled {
		t.Errorf("Expected callback to be called")
	}

	if receivedEvent.Language != "test" {
		t.Errorf("Expected received event language 'test', got '%s'", receivedEvent.Language)
	}
}

func TestWatcherManagerIsRunning(t *testing.T) {
	manager := NewWatcherManager()

	if manager.IsRunning() {
		t.Errorf("Expected manager to not be running initially")
	}

	// Simulate starting
	manager.running = true

	if !manager.IsRunning() {
		t.Errorf("Expected manager to be running after setting running=true")
	}
}

// MockWatcher is a test implementation of ExecutionWatcher
type MockWatcher struct {
	config   WatcherConfig
	running  bool
	callback func(ExecutionEvent)
}

func (m *MockWatcher) Start(callback func(ExecutionEvent)) error {
	m.running = true
	m.callback = callback
	return nil
}

func (m *MockWatcher) Stop() error {
	m.running = false
	return nil
}

func (m *MockWatcher) IsRunning() bool {
	return m.running
}

func (m *MockWatcher) GetConfig() WatcherConfig {
	return m.config
}

func (m *MockWatcher) GetLanguage() string {
	return m.config.Language
}

func (m *MockWatcher) GetEnvironment() string {
	return m.config.Environment
}

// TriggerEvent simulates an execution event for testing
func (m *MockWatcher) TriggerEvent(event ExecutionEvent) {
	if m.callback != nil {
		m.callback(event)
	}
}
