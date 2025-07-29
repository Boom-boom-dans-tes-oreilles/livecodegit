package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/livecodegit/pkg/core"
	"github.com/livecodegit/pkg/watchers"
)

func handleWatch(args []string) {
	watchFlags := flag.NewFlagSet("watch", flag.ExitOnError)
	language := watchFlags.String("lang", "", "Language to watch (sonicpi, tidal)")
	configPath := watchFlags.String("config", "", "Path to watcher configuration file")
	listWatchers := watchFlags.Bool("list", false, "List available watchers")
	showStatus := watchFlags.Bool("status", false, "Show watcher status")
	enableWatcher := watchFlags.String("enable", "", "Enable a specific watcher")
	disableWatcher := watchFlags.String("disable", "", "Disable a specific watcher")

	watchFlags.Parse(args)

	// Get current directory
	path, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting current directory: %v\n", err)
		os.Exit(1)
	}

	// Load repository
	repo, err := core.LoadRepository(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading repository: %v\n", err)
		fmt.Fprintf(os.Stderr, "Make sure you're in a LiveCodeGit repository (run 'lcg init' first)\n")
		os.Exit(1)
	}

	// Set default config path if not provided
	if *configPath == "" {
		*configPath = watchers.GetDefaultConfigPath()
	}

	// Create watcher service
	service := watchers.NewWatcherService(repo, *configPath)

	// Initialize service
	if err := service.Initialize(); err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing watcher service: %v\n", err)
		os.Exit(1)
	}

	// Handle different watch commands
	if *listWatchers {
		handleListWatchers(service)
		return
	}

	if *showStatus {
		handleShowStatus(service)
		return
	}

	if *enableWatcher != "" {
		handleEnableWatcher(service, *enableWatcher)
		return
	}

	if *disableWatcher != "" {
		handleDisableWatcher(service, *disableWatcher)
		return
	}

	// Start watching
	if *language != "" {
		handleStartWatchingLanguage(service, *language)
	} else {
		handleStartWatchingAll(service)
	}
}

func handleListWatchers(service *watchers.WatcherService) {
	enabledWatchers := service.GetEnabledWatchers()

	fmt.Printf("Available Watchers:\n\n")

	watchers := []struct {
		name        string
		language    string
		environment string
		description string
	}{
		{"sonicpi-osc", "sonicpi", "sonic-pi", "Monitors Sonic Pi OSC messages for execution events"},
		{"sonicpi-files", "sonicpi", "sonic-pi-files", "Watches Sonic Pi workspace files for changes"},
		{"tidal-ghci", "tidal", "tidal-cycles", "Monitors TidalCycles through GHCi interaction"},
	}

	for _, w := range watchers {
		enabled := contains(enabledWatchers, w.name)
		status := "disabled"
		if enabled {
			status = "enabled"
		}

		fmt.Printf("  %s (%s)\n", w.name, status)
		fmt.Printf("    Language: %s\n", w.language)
		fmt.Printf("    Environment: %s\n", w.environment)
		fmt.Printf("    Description: %s\n", w.description)

		// Show configuration
		if config, exists := service.GetWatcherConfig(w.name); exists {
			fmt.Printf("    Options:\n")
			for key, value := range config.Options {
				fmt.Printf("      %s: %s\n", key, value)
			}
		}
		fmt.Printf("\n")
	}
}

func handleShowStatus(service *watchers.WatcherService) {
	stats := service.GetStats()

	fmt.Printf("Watcher Service Status:\n\n")
	fmt.Printf("  Running: %t\n", stats.Running)
	fmt.Printf("  Active Watchers: %d\n", stats.ActiveWatchers)
	fmt.Printf("  Total Executions: %d\n", stats.TotalExecutions)
	fmt.Printf("  Total Commits: %d\n", stats.TotalCommits)

	if !stats.LastExecution.IsZero() {
		fmt.Printf("  Last Execution: %s\n", stats.LastExecution.Format("2006-01-02 15:04:05"))
	}

	fmt.Printf("\nEnabled Watchers:\n")
	for _, name := range service.GetEnabledWatchers() {
		fmt.Printf("  - %s\n", name)
	}
}

func handleEnableWatcher(service *watchers.WatcherService, watcherName string) {
	if err := service.EnableWatcher(watcherName); err != nil {
		fmt.Fprintf(os.Stderr, "Error enabling watcher: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Enabled watcher: %s\n", watcherName)
}

func handleDisableWatcher(service *watchers.WatcherService, watcherName string) {
	if err := service.DisableWatcher(watcherName); err != nil {
		fmt.Fprintf(os.Stderr, "Error disabling watcher: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Disabled watcher: %s\n", watcherName)
}

func handleStartWatchingLanguage(service *watchers.WatcherService, language string) {
	// Enable watchers for the specified language
	languageWatchers := getWatchersForLanguage(language)
	if len(languageWatchers) == 0 {
		fmt.Fprintf(os.Stderr, "No watchers available for language: %s\n", language)
		fmt.Fprintf(os.Stderr, "Available languages: sonicpi, tidal\n")
		os.Exit(1)
	}

	// Enable relevant watchers
	for _, watcherName := range languageWatchers {
		if err := service.EnableWatcher(watcherName); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to enable watcher %s: %v\n", watcherName, err)
		}
	}

	fmt.Printf("Starting watchers for %s...\n", language)
	startWatcherService(service)
}

func handleStartWatchingAll(service *watchers.WatcherService) {
	enabledWatchers := service.GetEnabledWatchers()
	if len(enabledWatchers) == 0 {
		fmt.Printf("No watchers are enabled. Use 'lcg watch --list' to see available watchers.\n")
		fmt.Printf("Enable a watcher first: lcg watch --enable <watcher-name>\n")
		os.Exit(1)
	}

	fmt.Printf("Starting %d enabled watchers...\n", len(enabledWatchers))
	startWatcherService(service)
}

func startWatcherService(service *watchers.WatcherService) {
	// Start the service
	if err := service.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting watcher service: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Watcher service started. Monitoring for code executions...\n")
	fmt.Printf("Press Ctrl+C to stop.\n\n")

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Print periodic status updates
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-sigChan:
			fmt.Printf("\nShutting down watcher service...\n")
			if err := service.Stop(); err != nil {
				fmt.Fprintf(os.Stderr, "Error stopping service: %v\n", err)
			}

			// Print final stats
			stats := service.GetStats()
			fmt.Printf("Final stats: %d executions, %d commits\n",
				stats.TotalExecutions, stats.TotalCommits)

			return

		case <-ticker.C:
			stats := service.GetStats()
			if stats.TotalExecutions > 0 {
				fmt.Printf("Status: %d executions, %d commits\n",
					stats.TotalExecutions, stats.TotalCommits)
			}
		}
	}
}

func getWatchersForLanguage(language string) []string {
	switch strings.ToLower(language) {
	case "sonicpi", "sonic-pi":
		return []string{"sonicpi-osc", "sonicpi-files"}
	case "tidal", "tidalcycles", "tidal-cycles":
		return []string{"tidal-ghci"}
	default:
		return []string{}
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
