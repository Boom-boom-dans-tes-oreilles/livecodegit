package storage

import (
	"os"
	"testing"
	"time"
)

func TestNewIndex(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	storage := NewFileSystemStorage(tempDir)
	index := NewIndex(storage)

	if index.storage != storage {
		t.Errorf("Index storage reference not set correctly")
	}

	if len(index.Entries) != 0 {
		t.Errorf("Expected empty entries, got %d entries", len(index.Entries))
	}
}

func TestLoadIndexEmpty(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	storage := NewFileSystemStorage(tempDir)
	err := storage.InitializeRepository()
	if err != nil {
		t.Fatalf("Failed to initialize repository: %v", err)
	}

	index := NewIndex(storage)
	err = index.LoadIndex()
	if err != nil {
		t.Fatalf("Failed to load empty index: %v", err)
	}

	if len(index.Entries) != 0 {
		t.Errorf("Expected empty entries, got %d entries", len(index.Entries))
	}
}

func TestAddEntry(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	storage := NewFileSystemStorage(tempDir)
	err := storage.InitializeRepository()
	if err != nil {
		t.Fatalf("Failed to initialize repository: %v", err)
	}

	index := NewIndex(storage)
	err = index.LoadIndex()
	if err != nil {
		t.Fatalf("Failed to load index: %v", err)
	}

	// Add first entry
	timestamp1 := time.Now()
	err = index.AddEntry("abc123", "First commit", "", timestamp1)
	if err != nil {
		t.Fatalf("Failed to add entry: %v", err)
	}

	if len(index.Entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(index.Entries))
	}

	entry := index.Entries[0]
	if entry.Hash != "abc123" {
		t.Errorf("Expected hash 'abc123', got '%s'", entry.Hash)
	}

	if entry.Message != "First commit" {
		t.Errorf("Expected message 'First commit', got '%s'", entry.Message)
	}

	if entry.Parent != "" {
		t.Errorf("Expected empty parent, got '%s'", entry.Parent)
	}

	// Add second entry with parent
	timestamp2 := time.Now().Add(time.Second)
	err = index.AddEntry("def456", "Second commit", "abc123", timestamp2)
	if err != nil {
		t.Fatalf("Failed to add second entry: %v", err)
	}

	if len(index.Entries) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(index.Entries))
	}

	entry2 := index.Entries[1]
	if entry2.Parent != "abc123" {
		t.Errorf("Expected parent 'abc123', got '%s'", entry2.Parent)
	}
}

func TestSaveAndLoadIndex(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	storage := NewFileSystemStorage(tempDir)
	err := storage.InitializeRepository()
	if err != nil {
		t.Fatalf("Failed to initialize repository: %v", err)
	}

	// Create index and add entries
	index1 := NewIndex(storage)
	err = index1.LoadIndex()
	if err != nil {
		t.Fatalf("Failed to load index: %v", err)
	}

	timestamp := time.Now()
	err = index1.AddEntry("abc123", "Test commit", "", timestamp)
	if err != nil {
		t.Fatalf("Failed to add entry: %v", err)
	}

	// Create new index and load from disk
	index2 := NewIndex(storage)
	err = index2.LoadIndex()
	if err != nil {
		t.Fatalf("Failed to load index from disk: %v", err)
	}

	if len(index2.Entries) != 1 {
		t.Errorf("Expected 1 entry after loading, got %d", len(index2.Entries))
	}

	entry := index2.Entries[0]
	if entry.Hash != "abc123" {
		t.Errorf("Expected hash 'abc123', got '%s'", entry.Hash)
	}

	if entry.Message != "Test commit" {
		t.Errorf("Expected message 'Test commit', got '%s'", entry.Message)
	}
}

func TestGetOrderedCommits(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	storage := NewFileSystemStorage(tempDir)
	err := storage.InitializeRepository()
	if err != nil {
		t.Fatalf("Failed to initialize repository: %v", err)
	}

	index := NewIndex(storage)
	err = index.LoadIndex()
	if err != nil {
		t.Fatalf("Failed to load index: %v", err)
	}

	// Add entries in chronological order
	baseTime := time.Now()
	entries := []struct {
		hash    string
		message string
		parent  string
		time    time.Time
	}{
		{"abc123", "First commit", "", baseTime},
		{"def456", "Second commit", "abc123", baseTime.Add(time.Second)},
		{"ghi789", "Third commit", "def456", baseTime.Add(2 * time.Second)},
	}

	for _, entry := range entries {
		err = index.AddEntry(entry.hash, entry.message, entry.parent, entry.time)
		if err != nil {
			t.Fatalf("Failed to add entry %s: %v", entry.hash, err)
		}
	}

	// Get ordered commits (should return most recent first)
	ordered := index.GetOrderedCommits(10)
	if len(ordered) != 3 {
		t.Errorf("Expected 3 ordered commits, got %d", len(ordered))
	}

	// Check reverse chronological order (most recent first)
	expectedOrder := []string{"ghi789", "def456", "abc123"}
	for i, expected := range expectedOrder {
		if ordered[i].Hash != expected {
			t.Errorf("Expected commit %d to be '%s', got '%s'", i, expected, ordered[i].Hash)
		}
	}

	// Test with limit
	limited := index.GetOrderedCommits(2)
	if len(limited) != 2 {
		t.Errorf("Expected 2 limited commits, got %d", len(limited))
	}

	if limited[0].Hash != "ghi789" {
		t.Errorf("Expected first limited commit to be 'ghi789', got '%s'", limited[0].Hash)
	}
}

func TestGetEntry(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	storage := NewFileSystemStorage(tempDir)
	err := storage.InitializeRepository()
	if err != nil {
		t.Fatalf("Failed to initialize repository: %v", err)
	}

	index := NewIndex(storage)
	err = index.LoadIndex()
	if err != nil {
		t.Fatalf("Failed to load index: %v", err)
	}

	// Add entry
	timestamp := time.Now()
	err = index.AddEntry("abc123", "Test commit", "", timestamp)
	if err != nil {
		t.Fatalf("Failed to add entry: %v", err)
	}

	// Get existing entry
	entry := index.GetEntry("abc123")
	if entry == nil {
		t.Fatalf("Expected to find entry, got nil")
	}

	if entry.Hash != "abc123" {
		t.Errorf("Expected hash 'abc123', got '%s'", entry.Hash)
	}

	// Get non-existing entry
	nonExistent := index.GetEntry("nonexistent")
	if nonExistent != nil {
		t.Errorf("Expected nil for non-existent entry, got %v", nonExistent)
	}
}

func TestGetHead(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	storage := NewFileSystemStorage(tempDir)
	err := storage.InitializeRepository()
	if err != nil {
		t.Fatalf("Failed to initialize repository: %v", err)
	}

	index := NewIndex(storage)
	err = index.LoadIndex()
	if err != nil {
		t.Fatalf("Failed to load index: %v", err)
	}

	// Empty index should return empty head
	head := index.GetHead()
	if head != "" {
		t.Errorf("Expected empty head for empty index, got '%s'", head)
	}

	// Add entries
	baseTime := time.Now()
	err = index.AddEntry("abc123", "First commit", "", baseTime)
	if err != nil {
		t.Fatalf("Failed to add first entry: %v", err)
	}

	head = index.GetHead()
	if head != "abc123" {
		t.Errorf("Expected head 'abc123', got '%s'", head)
	}

	err = index.AddEntry("def456", "Second commit", "abc123", baseTime.Add(time.Second))
	if err != nil {
		t.Fatalf("Failed to add second entry: %v", err)
	}

	head = index.GetHead()
	if head != "def456" {
		t.Errorf("Expected head 'def456', got '%s'", head)
	}
}

func TestRebuildIndex(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	storage := NewFileSystemStorage(tempDir)
	err := storage.InitializeRepository()
	if err != nil {
		t.Fatalf("Failed to initialize repository: %v", err)
	}

	// Write commits directly to storage
	commits := []*Commit{
		{Hash: "abc123", Message: "First commit", Author: "user", Timestamp: time.Now(), Content: "code1", Metadata: ExecutionMetadata{Language: "sonicpi"}},
		{Hash: "def456", Message: "Second commit", Author: "user", Timestamp: time.Now().Add(time.Second), Content: "code2", Metadata: ExecutionMetadata{Language: "sonicpi"}, Parent: "abc123"},
	}

	for _, commit := range commits {
		err = storage.WriteCommit(commit)
		if err != nil {
			t.Fatalf("Failed to write commit %s: %v", commit.Hash, err)
		}
	}

	// Create index and rebuild from storage
	index := NewIndex(storage)
	err = index.RebuildIndex()
	if err != nil {
		t.Fatalf("Failed to rebuild index: %v", err)
	}

	if len(index.Entries) != 2 {
		t.Errorf("Expected 2 entries after rebuild, got %d", len(index.Entries))
	}

	// Check chronological order (entries should be sorted by timestamp)
	if index.Entries[0].Hash != "abc123" {
		t.Errorf("Expected first entry to be 'abc123', got '%s'", index.Entries[0].Hash)
	}

	if index.Entries[1].Hash != "def456" {
		t.Errorf("Expected second entry to be 'def456', got '%s'", index.Entries[1].Hash)
	}
}