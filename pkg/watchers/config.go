package watchers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// GlobalConfig holds configuration for all watchers
type GlobalConfig struct {
	Watchers        map[string]WatcherConfig `json:"watchers"`
	DefaultLanguage string                   `json:"default_language"`
	AutoCommit      bool                     `json:"auto_commit"`
	CommitMessage   string                   `json:"commit_message"`
	WorkspacePath   string                   `json:"workspace_path"`
	LogLevel        string                   `json:"log_level"`
}

// DefaultGlobalConfig returns a default configuration
func DefaultGlobalConfig() GlobalConfig {
	return GlobalConfig{
		Watchers: map[string]WatcherConfig{
			"sonicpi-osc": {
				Language:    "sonicpi",
				Environment: "sonic-pi",
				Enabled:     false,
				Options: map[string]string{
					"osc_port":       "4559",
					"workspace_path": "",
				},
			},
			"sonicpi-files": {
				Language:    "sonicpi",
				Environment: "sonic-pi-files",
				Enabled:     false,
				Options: map[string]string{
					"workspace_path": "",
					"poll_interval":  "1s",
				},
			},
			"tidal-ghci": {
				Language:    "tidal",
				Environment: "tidal-cycles",
				Enabled:     false,
				Options: map[string]string{
					"ghci_command": "ghci",
					"boot_file":    "BootTidal.hs",
				},
			},
		},
		DefaultLanguage: "sonicpi",
		AutoCommit:      true,
		CommitMessage:   "Auto-commit: {{.Language}} execution in {{.Buffer}}",
		WorkspacePath:   "",
		LogLevel:        "info",
	}
}

// ConfigManager handles loading, saving, and managing watcher configurations
type ConfigManager struct {
	configPath string
	config     GlobalConfig
}

// NewConfigManager creates a new configuration manager
func NewConfigManager(configPath string) *ConfigManager {
	return &ConfigManager{
		configPath: configPath,
		config:     DefaultGlobalConfig(),
	}
}

// LoadConfig loads configuration from file
func (cm *ConfigManager) LoadConfig() error {
	if _, err := os.Stat(cm.configPath); os.IsNotExist(err) {
		// Config file doesn't exist, use defaults
		return cm.SaveConfig()
	}

	data, err := os.ReadFile(cm.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	if err := json.Unmarshal(data, &cm.config); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	return nil
}

// SaveConfig saves the current configuration to file
func (cm *ConfigManager) SaveConfig() error {
	// Ensure config directory exists
	configDir := filepath.Dir(cm.configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(cm.config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(cm.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetConfig returns the current global configuration
func (cm *ConfigManager) GetConfig() GlobalConfig {
	return cm.config
}

// UpdateConfig updates the global configuration
func (cm *ConfigManager) UpdateConfig(config GlobalConfig) {
	cm.config = config
}

// GetWatcherConfig returns configuration for a specific watcher
func (cm *ConfigManager) GetWatcherConfig(name string) (WatcherConfig, bool) {
	config, exists := cm.config.Watchers[name]
	return config, exists
}

// SetWatcherConfig updates configuration for a specific watcher
func (cm *ConfigManager) SetWatcherConfig(name string, config WatcherConfig) {
	if cm.config.Watchers == nil {
		cm.config.Watchers = make(map[string]WatcherConfig)
	}
	cm.config.Watchers[name] = config
}

// EnableWatcher enables a specific watcher
func (cm *ConfigManager) EnableWatcher(name string) error {
	config, exists := cm.config.Watchers[name]
	if !exists {
		return fmt.Errorf("watcher '%s' not found", name)
	}

	config.Enabled = true
	cm.config.Watchers[name] = config

	return nil
}

// DisableWatcher disables a specific watcher
func (cm *ConfigManager) DisableWatcher(name string) error {
	config, exists := cm.config.Watchers[name]
	if !exists {
		return fmt.Errorf("watcher '%s' not found", name)
	}

	config.Enabled = false
	cm.config.Watchers[name] = config

	return nil
}

// SetWatcherOption sets a specific option for a watcher
func (cm *ConfigManager) SetWatcherOption(watcherName, optionName, optionValue string) error {
	config, exists := cm.config.Watchers[watcherName]
	if !exists {
		return fmt.Errorf("watcher '%s' not found", watcherName)
	}

	if config.Options == nil {
		config.Options = make(map[string]string)
	}

	config.Options[optionName] = optionValue
	cm.config.Watchers[watcherName] = config

	return nil
}

// ListWatchers returns all configured watcher names
func (cm *ConfigManager) ListWatchers() []string {
	names := make([]string, 0, len(cm.config.Watchers))
	for name := range cm.config.Watchers {
		names = append(names, name)
	}
	return names
}

// GetEnabledWatchers returns names of all enabled watchers
func (cm *ConfigManager) GetEnabledWatchers() []string {
	var enabled []string
	for name, config := range cm.config.Watchers {
		if config.Enabled {
			enabled = append(enabled, name)
		}
	}
	return enabled
}

// ValidateConfig validates the current configuration
func (cm *ConfigManager) ValidateConfig() error {
	config := cm.config

	// Validate log level
	validLogLevels := []string{"debug", "info", "warn", "error"}
	validLevel := false
	for _, level := range validLogLevels {
		if config.LogLevel == level {
			validLevel = true
			break
		}
	}
	if !validLevel {
		return fmt.Errorf("invalid log level: %s", config.LogLevel)
	}

	// Validate watcher configurations
	for name, watcherConfig := range config.Watchers {
		if err := cm.validateWatcherConfig(name, watcherConfig); err != nil {
			return fmt.Errorf("invalid config for watcher '%s': %w", name, err)
		}
	}

	return nil
}

// validateWatcherConfig validates a specific watcher configuration
func (cm *ConfigManager) validateWatcherConfig(name string, config WatcherConfig) error {
	// Validate required fields
	if config.Language == "" {
		return fmt.Errorf("language is required")
	}

	if config.Environment == "" {
		return fmt.Errorf("environment is required")
	}

	// Validate specific watcher types
	switch name {
	case "sonicpi-osc":
		return cm.validateSonicPiOSCConfig(config)
	case "sonicpi-files":
		return cm.validateSonicPiFilesConfig(config)
	case "tidal-ghci":
		return cm.validateTidalGHCiConfig(config)
	}

	return nil
}

// validateSonicPiOSCConfig validates Sonic Pi OSC watcher configuration
func (cm *ConfigManager) validateSonicPiOSCConfig(config WatcherConfig) error {
	if portStr, exists := config.Options["osc_port"]; exists {
		if portStr == "" {
			return fmt.Errorf("osc_port cannot be empty")
		}
		// Could add port range validation here
	}

	return nil
}

// validateSonicPiFilesConfig validates Sonic Pi file watcher configuration
func (cm *ConfigManager) validateSonicPiFilesConfig(config WatcherConfig) error {
	if workspacePath, exists := config.Options["workspace_path"]; exists && workspacePath != "" {
		if _, err := os.Stat(workspacePath); os.IsNotExist(err) {
			return fmt.Errorf("workspace_path does not exist: %s", workspacePath)
		}
	}

	return nil
}

// validateTidalGHCiConfig validates Tidal GHCi watcher configuration
func (cm *ConfigManager) validateTidalGHCiConfig(config WatcherConfig) error {
	if ghciCmd, exists := config.Options["ghci_command"]; exists {
		if ghciCmd == "" {
			return fmt.Errorf("ghci_command cannot be empty")
		}
	}

	return nil
}

// GetDefaultConfigPath returns the default configuration file path
func GetDefaultConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ".livecodegit/watchers.json"
	}

	return filepath.Join(homeDir, ".livecodegit", "watchers.json")
}
