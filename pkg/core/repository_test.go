package core

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/livecodegit/pkg/storage"
)

func createTempDir(t *testing.T) string {
	tempDir, err := os.MkdirTemp("", "livecodegit-core-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	return tempDir
}

func TestNewRepository(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	repo := NewRepository(tempDir)
	if repo.path != tempDir {
		t.Errorf("Expected path '%s', got '%s'", tempDir, repo.path)
	}

	if repo.storage == nil {
		t.Errorf("Storage should not be nil")
	}

	if repo.index == nil {
		t.Errorf("Index should not be nil")
	}
}

func TestInit(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	repo := NewRepository("")
	err := repo.Init(tempDir)
	if err != nil {
		t.Fatalf("Failed to initialize repository: %v", err)
	}

	if repo.path != tempDir {
		t.Errorf("Expected path '%s', got '%s'", tempDir, repo.path)
	}

	// Check that repository directory structure exists
	repoDir := filepath.Join(tempDir, storage.RepoDir)
	if _, err := os.Stat(repoDir); os.IsNotExist(err) {
		t.Errorf("Repository directory was not created")
	}

	// Test double initialization
	err = repo.Init(tempDir)
	if err == nil {
		t.Errorf("Expected error when initializing existing repository")
	}
}

func TestIsInitialized(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	repo := NewRepository(tempDir)

	// Should not be initialized initially
	if repo.IsInitialized() {
		t.Errorf("Repository should not be initialized initially")
	}

	// Initialize repository
	err := repo.Init(tempDir)
	if err != nil {
		t.Fatalf("Failed to initialize repository: %v", err)
	}

	// Should be initialized now
	if !repo.IsInitialized() {
		t.Errorf("Repository should be initialized after Init()")
	}
}

func TestCommit(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	repo := NewRepository(tempDir)
	err := repo.Init(tempDir)
	if err != nil {
		t.Fatalf("Failed to initialize repository: %v", err)
	}

	// Create first commit
	metadata := ExecutionMetadata{
		Buffer:      "main",
		Language:    "sonicpi",
		BPM:         120.0,
		Success:     true,
		Environment: "test",
	}

	commit1, err := repo.Commit("live_loop :drums do\n  sample :bd_haus\nend", "First commit", metadata)
	if err != nil {
		t.Fatalf("Failed to create first commit: %v", err)
	}

	if commit1.Hash == "" {
		t.Errorf("Commit hash should not be empty")
	}

	if commit1.Parent != "" {
		t.Errorf("First commit should have no parent, got '%s'", commit1.Parent)
	}

	if commit1.Message != "First commit" {
		t.Errorf("Expected message 'First commit', got '%s'", commit1.Message)
	}

	// Create second commit
	commit2, err := repo.Commit("live_loop :bass do\n  synth :tb303\nend", "Add bass", metadata)
	if err != nil {
		t.Fatalf("Failed to create second commit: %v", err)
	}

	if commit2.Parent != commit1.Hash {
		t.Errorf("Expected parent '%s', got '%s'", commit1.Hash, commit2.Parent)
	}

	// Verify commits can be retrieved
	retrieved1, err := repo.GetCommit(commit1.Hash)
	if err != nil {
		t.Fatalf("Failed to retrieve first commit: %v", err)
	}

	if retrieved1.Content != commit1.Content {
		t.Errorf("Expected content '%s', got '%s'", commit1.Content, retrieved1.Content)
	}
}

func TestCommitWithoutInit(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	repo := NewRepository(tempDir)
	metadata := ExecutionMetadata{
		Buffer:   "main",
		Language: "sonicpi",
		Success:  true,
	}

	_, err := repo.Commit("test code", "test commit", metadata)
	if err == nil {
		t.Errorf("Expected error when committing without initialization")
	}
}

func TestLog(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	repo := NewRepository(tempDir)
	err := repo.Init(tempDir)
	if err != nil {
		t.Fatalf("Failed to initialize repository: %v", err)
	}

	metadata := ExecutionMetadata{
		Buffer:   "main",
		Language: "sonicpi",
		Success:  true,
	}

	// Create multiple commits
	commits := []string{
		"First commit",
		"Second commit",
		"Third commit",
	}

	for _, message := range commits {
		_, err := repo.Commit("test code", message, metadata)
		if err != nil {
			t.Fatalf("Failed to create commit '%s': %v", message, err)
		}
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}

	// Get log
	log, err := repo.Log(10)
	if err != nil {
		t.Fatalf("Failed to get log: %v", err)
	}

	if len(log) != 3 {
		t.Errorf("Expected 3 commits in log, got %d", len(log))
	}

	// Check chronological order (most recent first)
	expectedMessages := []string{"Third commit", "Second commit", "First commit"}
	for i, expected := range expectedMessages {
		if log[i].Message != expected {
			t.Errorf("Expected commit %d message '%s', got '%s'", i, expected, log[i].Message)
		}
	}

	// Test with limit
	limitedLog, err := repo.Log(2)
	if err != nil {
		t.Fatalf("Failed to get limited log: %v", err)
	}

	if len(limitedLog) != 2 {
		t.Errorf("Expected 2 commits in limited log, got %d", len(limitedLog))
	}
}

func TestLogWithoutInit(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	repo := NewRepository(tempDir)
	_, err := repo.Log(10)
	if err == nil {
		t.Errorf("Expected error when getting log without initialization")
	}
}

func TestStartAndEndPerformance(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	repo := NewRepository(tempDir)
	err := repo.Init(tempDir)
	if err != nil {
		t.Fatalf("Failed to initialize repository: %v", err)
	}

	// Start performance
	performance, err := repo.StartPerformance("Test Performance")
	if err != nil {
		t.Fatalf("Failed to start performance: %v", err)
	}

	if performance.Name != "Test Performance" {
		t.Errorf("Expected performance name 'Test Performance', got '%s'", performance.Name)
	}

	if performance.CommitCount != 0 {
		t.Errorf("Expected initial commit count 0, got %d", performance.CommitCount)
	}

	// Check current performance
	current, err := repo.GetCurrentPerformance()
	if err != nil {
		t.Fatalf("Failed to get current performance: %v", err)
	}

	if current.ID != performance.ID {
		t.Errorf("Expected current performance ID '%s', got '%s'", performance.ID, current.ID)
	}

	// Create commit during performance
	metadata := ExecutionMetadata{
		Buffer:   "main",
		Language: "sonicpi",
		Success:  true,
	}

	_, err = repo.Commit("test code", "test commit", metadata)
	if err != nil {
		t.Fatalf("Failed to create commit during performance: %v", err)
	}

	// Check that performance was updated
	current, err = repo.GetCurrentPerformance()
	if err != nil {
		t.Fatalf("Failed to get updated performance: %v", err)
	}

	if current.CommitCount != 1 {
		t.Errorf("Expected commit count 1 after commit, got %d", current.CommitCount)
	}

	// End performance
	err = repo.EndPerformance()
	if err != nil {
		t.Fatalf("Failed to end performance: %v", err)
	}

	// Check that there's no current performance
	current, err = repo.GetCurrentPerformance()
	if err != nil {
		t.Fatalf("Failed to get current performance after end: %v", err)
	}

	if current != nil {
		t.Errorf("Expected no current performance after end, got %v", current)
	}
}

func TestEndPerformanceWithoutStart(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	repo := NewRepository(tempDir)
	err := repo.Init(tempDir)
	if err != nil {
		t.Fatalf("Failed to initialize repository: %v", err)
	}

	err = repo.EndPerformance()
	if err == nil {
		t.Errorf("Expected error when ending performance without starting")
	}
}

func TestLoadRepository(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	// Initialize repository
	repo1 := NewRepository(tempDir)
	err := repo1.Init(tempDir)
	if err != nil {
		t.Fatalf("Failed to initialize repository: %v", err)
	}

	// Create a commit
	metadata := ExecutionMetadata{
		Buffer:   "main",
		Language: "sonicpi",
		Success:  true,
	}

	_, err = repo1.Commit("test code", "test commit", metadata)
	if err != nil {
		t.Fatalf("Failed to create commit: %v", err)
	}

	// Load existing repository
	repo2, err := LoadRepository(tempDir)
	if err != nil {
		t.Fatalf("Failed to load repository: %v", err)
	}

	// Check that commits are accessible
	log, err := repo2.Log(10)
	if err != nil {
		t.Fatalf("Failed to get log from loaded repository: %v", err)
	}

	if len(log) != 1 {
		t.Errorf("Expected 1 commit in loaded repository, got %d", len(log))
	}
}

func TestLoadNonExistentRepository(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	_, err := LoadRepository(tempDir)
	if err == nil {
		t.Errorf("Expected error when loading non-existent repository")
	}
}