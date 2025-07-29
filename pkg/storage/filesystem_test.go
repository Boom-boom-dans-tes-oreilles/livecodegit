package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func createTempDir(t *testing.T) string {
	tempDir, err := os.MkdirTemp("", "livecodegit-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	return tempDir
}

func createTestCommit() *Commit {
	return &Commit{
		Hash:      "abc123def456",
		Parent:    "parent123",
		Timestamp: time.Now(),
		Message:   "Test commit",
		Author:    "testuser",
		Content:   "live_loop :drums do\n  sample :bd_haus\nend",
		Metadata: ExecutionMetadata{
			Buffer:      "main",
			Language:    "sonicpi",
			BPM:         120.0,
			Success:     true,
			Environment: "test",
		},
	}
}

func createTestPerformance() *Performance {
	return &Performance{
		ID:          "perf-123",
		Name:        "Test Performance",
		StartTime:   time.Now(),
		CommitCount: 1,
		HeadCommit:  "abc123def456",
		Branch:      "main",
		Author:      "testuser",
	}
}

func TestNewFileSystemStorage(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	storage := NewFileSystemStorage(tempDir)
	if storage.repoPath != tempDir {
		t.Errorf("Expected repo path '%s', got '%s'", tempDir, storage.repoPath)
	}
}

func TestInitializeRepository(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	storage := NewFileSystemStorage(tempDir)
	err := storage.InitializeRepository()
	if err != nil {
		t.Fatalf("Failed to initialize repository: %v", err)
	}

	// Check that required directories exist
	dirs := []string{
		filepath.Join(tempDir, RepoDir),
		filepath.Join(tempDir, RepoDir, ObjectsDir),
		filepath.Join(tempDir, RepoDir, PerformanceDir),
	}

	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("Directory %s was not created", dir)
		}
	}

	// Check that index file exists
	indexPath := filepath.Join(tempDir, RepoDir, IndexFile)
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		t.Errorf("Index file was not created")
	}
}

func TestWriteAndReadCommit(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	storage := NewFileSystemStorage(tempDir)
	err := storage.InitializeRepository()
	if err != nil {
		t.Fatalf("Failed to initialize repository: %v", err)
	}

	commit := createTestCommit()

	// Write commit
	err = storage.WriteCommit(commit)
	if err != nil {
		t.Fatalf("Failed to write commit: %v", err)
	}

	// Read commit back
	readCommit, err := storage.ReadCommit(commit.Hash)
	if err != nil {
		t.Fatalf("Failed to read commit: %v", err)
	}

	// Verify commit data
	if readCommit.Hash != commit.Hash {
		t.Errorf("Expected hash '%s', got '%s'", commit.Hash, readCommit.Hash)
	}

	if readCommit.Message != commit.Message {
		t.Errorf("Expected message '%s', got '%s'", commit.Message, readCommit.Message)
	}

	if readCommit.Content != commit.Content {
		t.Errorf("Expected content '%s', got '%s'", commit.Content, readCommit.Content)
	}

	if readCommit.Metadata.Language != commit.Metadata.Language {
		t.Errorf("Expected language '%s', got '%s'", commit.Metadata.Language, readCommit.Metadata.Language)
	}
}

func TestWriteAndReadPerformance(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	storage := NewFileSystemStorage(tempDir)
	err := storage.InitializeRepository()
	if err != nil {
		t.Fatalf("Failed to initialize repository: %v", err)
	}

	performance := createTestPerformance()

	// Write performance
	err = storage.WritePerformance(performance)
	if err != nil {
		t.Fatalf("Failed to write performance: %v", err)
	}

	// Read performance back
	readPerformance, err := storage.ReadPerformance(performance.ID)
	if err != nil {
		t.Fatalf("Failed to read performance: %v", err)
	}

	// Verify performance data
	if readPerformance.ID != performance.ID {
		t.Errorf("Expected ID '%s', got '%s'", performance.ID, readPerformance.ID)
	}

	if readPerformance.Name != performance.Name {
		t.Errorf("Expected name '%s', got '%s'", performance.Name, readPerformance.Name)
	}

	if readPerformance.CommitCount != performance.CommitCount {
		t.Errorf("Expected commit count %d, got %d", performance.CommitCount, readPerformance.CommitCount)
	}
}

func TestExists(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	storage := NewFileSystemStorage(tempDir)
	err := storage.InitializeRepository()
	if err != nil {
		t.Fatalf("Failed to initialize repository: %v", err)
	}

	commit := createTestCommit()

	// Should not exist initially
	if storage.Exists(commit.Hash) {
		t.Errorf("Commit should not exist initially")
	}

	// Write commit
	err = storage.WriteCommit(commit)
	if err != nil {
		t.Fatalf("Failed to write commit: %v", err)
	}

	// Should exist now
	if !storage.Exists(commit.Hash) {
		t.Errorf("Commit should exist after writing")
	}
}

func TestListCommits(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	storage := NewFileSystemStorage(tempDir)
	err := storage.InitializeRepository()
	if err != nil {
		t.Fatalf("Failed to initialize repository: %v", err)
	}

	// Write multiple commits
	commits := []*Commit{
		{Hash: "abc123", Message: "First commit", Author: "user", Timestamp: time.Now(), Content: "code1", Metadata: ExecutionMetadata{Language: "sonicpi"}},
		{Hash: "def456", Message: "Second commit", Author: "user", Timestamp: time.Now(), Content: "code2", Metadata: ExecutionMetadata{Language: "sonicpi"}},
		{Hash: "ghi789", Message: "Third commit", Author: "user", Timestamp: time.Now(), Content: "code3", Metadata: ExecutionMetadata{Language: "sonicpi"}},
	}

	for _, commit := range commits {
		err = storage.WriteCommit(commit)
		if err != nil {
			t.Fatalf("Failed to write commit %s: %v", commit.Hash, err)
		}
	}

	// List commits
	hashes, err := storage.ListCommits()
	if err != nil {
		t.Fatalf("Failed to list commits: %v", err)
	}

	if len(hashes) != 3 {
		t.Errorf("Expected 3 commits, got %d", len(hashes))
	}

	// Check that all hashes are present
	expectedHashes := map[string]bool{"abc123": true, "def456": true, "ghi789": true}
	for _, hash := range hashes {
		if !expectedHashes[hash] {
			t.Errorf("Unexpected hash '%s' in list", hash)
		}
		delete(expectedHashes, hash)
	}

	if len(expectedHashes) > 0 {
		t.Errorf("Missing hashes in list: %v", expectedHashes)
	}
}

func TestGenerateHash(t *testing.T) {
	content1 := "test content"
	content2 := "different content"

	hash1 := GenerateHash(content1)
	hash2 := GenerateHash(content2)

	if hash1 == hash2 {
		t.Errorf("Different content should produce different hashes")
	}

	// Same content should produce same hash
	hash3 := GenerateHash(content1)
	if hash1 != hash3 {
		t.Errorf("Same content should produce same hash")
	}

	// Hash should be 40 characters (SHA-1)
	if len(hash1) != 40 {
		t.Errorf("Expected hash length 40, got %d", len(hash1))
	}
}

func TestWriteAndReadHead(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	storage := NewFileSystemStorage(tempDir)
	err := storage.InitializeRepository()
	if err != nil {
		t.Fatalf("Failed to initialize repository: %v", err)
	}

	commitHash := "abc123def456"

	// Write HEAD
	err = storage.WriteHead(commitHash)
	if err != nil {
		t.Fatalf("Failed to write HEAD: %v", err)
	}

	// Read HEAD
	readHash, err := storage.ReadHead()
	if err != nil {
		t.Fatalf("Failed to read HEAD: %v", err)
	}

	if readHash != commitHash {
		t.Errorf("Expected HEAD '%s', got '%s'", commitHash, readHash)
	}
}
