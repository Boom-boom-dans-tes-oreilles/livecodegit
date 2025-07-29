package watchers

import (
	"fmt"

	"github.com/livecodegit/pkg/watchers/common"
)

// Type aliases for common types
type ExecutionEvent = common.ExecutionEvent
type WatcherConfig = common.WatcherConfig
type ExecutionWatcher = common.ExecutionWatcher

// WatcherManager manages multiple watchers and coordinates their execution
type WatcherManager struct {
	watchers map[string]ExecutionWatcher
	callback func(event ExecutionEvent)
	running  bool
}

// NewWatcherManager creates a new watcher manager
func NewWatcherManager() *WatcherManager {
	return &WatcherManager{
		watchers: make(map[string]ExecutionWatcher),
		running:  false,
	}
}

// RegisterWatcher adds a watcher to the manager
func (wm *WatcherManager) RegisterWatcher(name string, watcher ExecutionWatcher) {
	wm.watchers[name] = watcher
}

// SetCallback sets the function to call when executions are detected
func (wm *WatcherManager) SetCallback(callback func(event ExecutionEvent)) {
	wm.callback = callback
}

// StartAll starts all registered watchers that are enabled
func (wm *WatcherManager) StartAll() error {
	if wm.callback == nil {
		return fmt.Errorf("no callback function set")
	}

	var startedAny bool
	for name, watcher := range wm.watchers {
		if watcher.GetConfig().Enabled {
			if err := watcher.Start(wm.callback); err != nil {
				return fmt.Errorf("failed to start watcher %s: %w", name, err)
			}
			startedAny = true
		}
	}

	wm.running = startedAny
	return nil
}

// StopAll stops all running watchers
func (wm *WatcherManager) StopAll() error {
	var lastError error

	for name, watcher := range wm.watchers {
		if watcher.IsRunning() {
			if err := watcher.Stop(); err != nil {
				lastError = fmt.Errorf("failed to stop watcher %s: %w", name, err)
			}
		}
	}

	wm.running = false
	return lastError
}

// GetWatcher returns a watcher by name
func (wm *WatcherManager) GetWatcher(name string) (ExecutionWatcher, bool) {
	watcher, exists := wm.watchers[name]
	return watcher, exists
}

// ListWatchers returns all registered watcher names
func (wm *WatcherManager) ListWatchers() []string {
	names := make([]string, 0, len(wm.watchers))
	for name := range wm.watchers {
		names = append(names, name)
	}
	return names
}

// IsRunning returns true if the manager is currently running watchers
func (wm *WatcherManager) IsRunning() bool {
	return wm.running
}
