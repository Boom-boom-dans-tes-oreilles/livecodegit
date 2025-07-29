package sonicpi

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"github.com/livecodegit/pkg/watchers/common"
)

// FileWatcher monitors Sonic Pi workspace files for changes
type FileWatcher struct {
	config        common.WatcherConfig
	workspacePath string
	running       bool
	mutex         sync.RWMutex
	callback      func(common.ExecutionEvent)
	lastModified  map[string]time.Time
	stopChan      chan struct{}
	
	// Polling interval for file changes
	pollInterval time.Duration
}

// NewFileWatcher creates a new file system watcher for Sonic Pi
func NewFileWatcher(workspacePath string) *FileWatcher {
	return &FileWatcher{
		config: common.WatcherConfig{
			Language:    "sonicpi",
			Environment: "sonic-pi-files",
			Enabled:     true,
			Options: map[string]string{
				"workspace_path":  workspacePath,
				"poll_interval":   "1s",
			},
		},
		workspacePath: workspacePath,
		running:       false,
		lastModified:  make(map[string]time.Time),
		pollInterval:  1 * time.Second,
	}
}

// Start begins monitoring workspace files
func (w *FileWatcher) Start(callback func(common.ExecutionEvent)) error {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	
	if w.running {
		return fmt.Errorf("file watcher is already running")
	}
	
	// Check if workspace path exists
	if _, err := os.Stat(w.workspacePath); os.IsNotExist(err) {
		return fmt.Errorf("workspace path does not exist: %s", w.workspacePath)
	}
	
	w.callback = callback
	w.running = true
	w.stopChan = make(chan struct{})
	
	// Initialize file modification times
	w.scanWorkspaceFiles()
	
	// Start monitoring in a goroutine
	go w.monitorFiles()
	
	return nil
}

// Stop stops the file watcher
func (w *FileWatcher) Stop() error {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	
	if !w.running {
		return nil
	}
	
	w.running = false
	close(w.stopChan)
	
	return nil
}

// IsRunning returns true if the watcher is active
func (w *FileWatcher) IsRunning() bool {
	w.mutex.RLock()
	defer w.mutex.RUnlock()
	return w.running
}

// GetConfig returns the watcher configuration
func (w *FileWatcher) GetConfig() common.WatcherConfig {
	return w.config
}

// GetLanguage returns "sonicpi"
func (w *FileWatcher) GetLanguage() string {
	return "sonicpi"
}

// GetEnvironment returns "sonic-pi-files"
func (w *FileWatcher) GetEnvironment() string {
	return "sonic-pi-files"
}

// monitorFiles continuously monitors workspace files for changes
func (w *FileWatcher) monitorFiles() {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-w.stopChan:
			return
		case <-ticker.C:
			w.checkForChanges()
		}
	}
}

// scanWorkspaceFiles initializes the file modification time map
func (w *FileWatcher) scanWorkspaceFiles() {
	filepath.WalkDir(w.workspacePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Continue on errors
		}
		
		if w.isSonicPiFile(path) {
			if info, err := d.Info(); err == nil {
				w.lastModified[path] = info.ModTime()
			}
		}
		
		return nil
	})
}

// checkForChanges scans for file modifications
func (w *FileWatcher) checkForChanges() {
	filepath.WalkDir(w.workspacePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Continue on errors
		}
		
		if !w.isSonicPiFile(path) {
			return nil
		}
		
		info, err := d.Info()
		if err != nil {
			return nil
		}
		
		currentModTime := info.ModTime()
		lastModTime, exists := w.lastModified[path]
		
		// Check if file was modified
		if !exists || currentModTime.After(lastModTime) {
			w.lastModified[path] = currentModTime
			
			// Only trigger event if file existed before (not for new files on first scan)
			if exists {
				event := w.createExecutionEvent(path, currentModTime)
				if w.callback != nil {
					w.callback(event)
				}
			}
		}
		
		return nil
	})
}

// isSonicPiFile checks if a file is a Sonic Pi workspace file
func (w *FileWatcher) isSonicPiFile(path string) bool {
	// Sonic Pi workspace files are typically named like:
	// - workspace_0, workspace_1, etc.
	// - *.rb files
	// - buffer_* files
	
	name := filepath.Base(path)
	
	patterns := []string{
		`^workspace_\d+$`,
		`^buffer_\d+$`,
		`\.rb$`,
		`\.sonic$`,
	}
	
	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, name); matched {
			return true
		}
	}
	
	return false
}

// createExecutionEvent creates an execution event from a file change
func (w *FileWatcher) createExecutionEvent(filePath string, modTime time.Time) common.ExecutionEvent {
	content, err := os.ReadFile(filePath)
	contentStr := ""
	success := true
	errorMessage := ""
	
	if err != nil {
		success = false
		errorMessage = fmt.Sprintf("Failed to read file: %v", err)
		contentStr = fmt.Sprintf("# Error reading file: %s", filePath)
	} else {
		contentStr = string(content)
	}
	
	// Extract buffer name from file path
	fileName := filepath.Base(filePath)
	buffer := w.extractBufferName(fileName)
	
	return common.ExecutionEvent{
		Timestamp:    modTime,
		Content:      contentStr,
		Buffer:       buffer,
		Language:     "sonicpi",
		Environment:  "sonic-pi-files",
		Success:      success,
		ErrorMessage: errorMessage,
		FilePath:     filePath,
		ExtraData: map[string]string{
			"file_name":    fileName,
			"trigger_type": "file_change",
		},
	}
}

// extractBufferName extracts a buffer name from a file name
func (w *FileWatcher) extractBufferName(fileName string) string {
	// Extract buffer number from workspace files
	if matched, _ := regexp.MatchString(`^workspace_(\d+)$`, fileName); matched {
		return fileName
	}
	
	// Extract buffer number from buffer files
	if matched, _ := regexp.MatchString(`^buffer_(\d+)$`, fileName); matched {
		return fileName
	}
	
	// For .rb files, use the file name without extension
	if filepath.Ext(fileName) == ".rb" {
		return fileName[:len(fileName)-3]
	}
	
	// Default to file name
	return fileName
}

// SetPollInterval changes the polling interval for file changes
func (w *FileWatcher) SetPollInterval(interval time.Duration) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	
	w.pollInterval = interval
	w.config.Options["poll_interval"] = interval.String()
}