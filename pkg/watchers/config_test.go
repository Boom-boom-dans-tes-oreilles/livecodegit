package watchers

import (
	"os"
	"path/filepath"
	"testing"
)

func createTempConfigFile(t *testing.T) string {
	tempDir, err := os.MkdirTemp("", "livecodegit-config-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	return filepath.Join(tempDir, "watchers.json")
}

func TestDefaultGlobalConfig(t *testing.T) {
	config := DefaultGlobalConfig()

	if config.DefaultLanguage != "sonicpi" {
		t.Errorf("Expected default language 'sonicpi', got '%s'", config.DefaultLanguage)
	}

	if !config.AutoCommit {
		t.Errorf("Expected auto-commit to be enabled by default")
	}

	if config.LogLevel != "info" {
		t.Errorf("Expected log level 'info', got '%s'", config.LogLevel)
	}

	// Check that default watchers are configured
	expectedWatchers := []string{"sonicpi-osc", "sonicpi-files", "tidal-ghci"}
	for _, watcherName := range expectedWatchers {
		if _, exists := config.Watchers[watcherName]; !exists {
			t.Errorf("Expected default watcher '%s' to be configured", watcherName)
		}
	}

	// Check Sonic Pi OSC watcher default config
	sonicpiOSC := config.Watchers["sonicpi-osc"]
	if sonicpiOSC.Language != "sonicpi" {
		t.Errorf("Expected sonicpi-osc language 'sonicpi', got '%s'", sonicpiOSC.Language)
	}

	if sonicpiOSC.Enabled {
		t.Errorf("Expected sonicpi-osc to be disabled by default")
	}

	if sonicpiOSC.Options["osc_port"] != "4559" {
		t.Errorf("Expected default OSC port '4559', got '%s'", sonicpiOSC.Options["osc_port"])
	}
}

func TestNewConfigManager(t *testing.T) {
	configPath := createTempConfigFile(t)
	defer os.RemoveAll(filepath.Dir(configPath))

	manager := NewConfigManager(configPath)

	if manager.configPath != configPath {
		t.Errorf("Expected config path '%s', got '%s'", configPath, manager.configPath)
	}

	// Should start with default config
	if manager.config.DefaultLanguage != "sonicpi" {
		t.Errorf("Expected default language 'sonicpi', got '%s'", manager.config.DefaultLanguage)
	}
}

func TestConfigManagerLoadSaveConfig(t *testing.T) {
	configPath := createTempConfigFile(t)
	defer os.RemoveAll(filepath.Dir(configPath))

	manager := NewConfigManager(configPath)

	// Load config (should create default since file doesn't exist)
	err := manager.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Check that config file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Errorf("Expected config file to be created")
	}

	// Modify config
	config := manager.GetConfig()
	config.DefaultLanguage = "tidal"
	config.AutoCommit = false
	manager.UpdateConfig(config)

	// Save config
	err = manager.SaveConfig()
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Create new manager and load config
	manager2 := NewConfigManager(configPath)
	err = manager2.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}

	// Check that changes were persisted
	loadedConfig := manager2.GetConfig()
	if loadedConfig.DefaultLanguage != "tidal" {
		t.Errorf("Expected loaded default language 'tidal', got '%s'", loadedConfig.DefaultLanguage)
	}

	if loadedConfig.AutoCommit {
		t.Errorf("Expected auto-commit to be disabled in loaded config")
	}
}

func TestConfigManagerWatcherOperations(t *testing.T) {
	configPath := createTempConfigFile(t)
	defer os.RemoveAll(filepath.Dir(configPath))

	manager := NewConfigManager(configPath)
	err := manager.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Test GetWatcherConfig
	config, exists := manager.GetWatcherConfig("sonicpi-osc")
	if !exists {
		t.Errorf("Expected to find sonicpi-osc watcher config")
	}

	if config.Language != "sonicpi" {
		t.Errorf("Expected language 'sonicpi', got '%s'", config.Language)
	}

	// Test SetWatcherConfig
	newConfig := WatcherConfig{
		Language:    "test",
		Environment: "test-env",
		Enabled:     true,
		Options: map[string]string{
			"test_option": "test_value",
		},
	}

	manager.SetWatcherConfig("test-watcher", newConfig)

	retrievedConfig, exists := manager.GetWatcherConfig("test-watcher")
	if !exists {
		t.Errorf("Expected to find newly set watcher config")
	}

	if retrievedConfig.Language != "test" {
		t.Errorf("Expected language 'test', got '%s'", retrievedConfig.Language)
	}

	if retrievedConfig.Options["test_option"] != "test_value" {
		t.Errorf("Expected test_option 'test_value', got '%s'", retrievedConfig.Options["test_option"])
	}
}

func TestConfigManagerEnableDisableWatcher(t *testing.T) {
	configPath := createTempConfigFile(t)
	defer os.RemoveAll(filepath.Dir(configPath))

	manager := NewConfigManager(configPath)
	err := manager.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Initially, sonicpi-osc should be disabled
	config, _ := manager.GetWatcherConfig("sonicpi-osc")
	if config.Enabled {
		t.Errorf("Expected sonicpi-osc to be disabled initially")
	}

	// Enable watcher
	err = manager.EnableWatcher("sonicpi-osc")
	if err != nil {
		t.Fatalf("Failed to enable watcher: %v", err)
	}

	config, _ = manager.GetWatcherConfig("sonicpi-osc")
	if !config.Enabled {
		t.Errorf("Expected sonicpi-osc to be enabled after EnableWatcher")
	}

	// Disable watcher
	err = manager.DisableWatcher("sonicpi-osc")
	if err != nil {
		t.Fatalf("Failed to disable watcher: %v", err)
	}

	config, _ = manager.GetWatcherConfig("sonicpi-osc")
	if config.Enabled {
		t.Errorf("Expected sonicpi-osc to be disabled after DisableWatcher")
	}

	// Test enabling non-existent watcher
	err = manager.EnableWatcher("non-existent")
	if err == nil {
		t.Errorf("Expected error when enabling non-existent watcher")
	}
}

func TestConfigManagerSetWatcherOption(t *testing.T) {
	configPath := createTempConfigFile(t)
	defer os.RemoveAll(filepath.Dir(configPath))

	manager := NewConfigManager(configPath)
	err := manager.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Set option for existing watcher
	err = manager.SetWatcherOption("sonicpi-osc", "osc_port", "4560")
	if err != nil {
		t.Fatalf("Failed to set watcher option: %v", err)
	}

	config, _ := manager.GetWatcherConfig("sonicpi-osc")
	if config.Options["osc_port"] != "4560" {
		t.Errorf("Expected osc_port '4560', got '%s'", config.Options["osc_port"])
	}

	// Test setting option for non-existent watcher
	err = manager.SetWatcherOption("non-existent", "option", "value")
	if err == nil {
		t.Errorf("Expected error when setting option for non-existent watcher")
	}
}

func TestConfigManagerListOperations(t *testing.T) {
	configPath := createTempConfigFile(t)
	defer os.RemoveAll(filepath.Dir(configPath))

	manager := NewConfigManager(configPath)
	err := manager.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Test ListWatchers
	watchers := manager.ListWatchers()
	expectedWatchers := []string{"sonicpi-osc", "sonicpi-files", "tidal-ghci"}

	if len(watchers) != len(expectedWatchers) {
		t.Errorf("Expected %d watchers, got %d", len(expectedWatchers), len(watchers))
	}

	for _, expected := range expectedWatchers {
		found := false
		for _, actual := range watchers {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find watcher '%s' in list", expected)
		}
	}

	// Test GetEnabledWatchers (initially none should be enabled)
	enabled := manager.GetEnabledWatchers()
	if len(enabled) != 0 {
		t.Errorf("Expected 0 enabled watchers initially, got %d", len(enabled))
	}

	// Enable a watcher and test again
	manager.EnableWatcher("sonicpi-osc")
	enabled = manager.GetEnabledWatchers()
	if len(enabled) != 1 {
		t.Errorf("Expected 1 enabled watcher, got %d", len(enabled))
	}

	if enabled[0] != "sonicpi-osc" {
		t.Errorf("Expected enabled watcher 'sonicpi-osc', got '%s'", enabled[0])
	}
}

func TestConfigManagerValidation(t *testing.T) {
	configPath := createTempConfigFile(t)
	defer os.RemoveAll(filepath.Dir(configPath))

	manager := NewConfigManager(configPath)
	err := manager.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Valid config should pass validation
	err = manager.ValidateConfig()
	if err != nil {
		t.Errorf("Expected valid config to pass validation: %v", err)
	}

	// Test invalid log level
	config := manager.GetConfig()
	config.LogLevel = "invalid"
	manager.UpdateConfig(config)

	err = manager.ValidateConfig()
	if err == nil {
		t.Errorf("Expected validation to fail for invalid log level")
	}

	// Reset to valid config
	config.LogLevel = "info"
	manager.UpdateConfig(config)

	// Test invalid watcher config
	invalidWatcherConfig := WatcherConfig{
		Language:    "", // Invalid: empty language
		Environment: "test",
		Enabled:     true,
	}
	manager.SetWatcherConfig("invalid-watcher", invalidWatcherConfig)

	err = manager.ValidateConfig()
	if err == nil {
		t.Errorf("Expected validation to fail for watcher with empty language")
	}
}

func TestGetDefaultConfigPath(t *testing.T) {
	path := GetDefaultConfigPath()

	if path == "" {
		t.Errorf("Expected non-empty default config path")
	}

	// Should end with the expected filename
	if filepath.Base(path) != "watchers.json" {
		t.Errorf("Expected config file name 'watchers.json', got '%s'", filepath.Base(path))
	}

	// Should contain .livecodegit directory
	if !filepath.IsAbs(path) && !filepath.HasPrefix(path, ".livecodegit") {
		t.Errorf("Expected path to contain .livecodegit directory")
	}
}
