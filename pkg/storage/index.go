package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// IndexEntry represents a single entry in the repository index
type IndexEntry struct {
	Hash      string    `json:"hash"`
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
	Parent    string    `json:"parent,omitempty"`
}

// Index manages the repository index for fast commit lookups
type Index struct {
	Entries []IndexEntry `json:"entries"`
	storage *FileSystemStorage
}

// NewIndex creates a new index manager
func NewIndex(storage *FileSystemStorage) *Index {
	return &Index{
		Entries: make([]IndexEntry, 0),
		storage: storage,
	}
}

// LoadIndex reads the index from disk
func (idx *Index) LoadIndex() error {
	indexPath := filepath.Join(idx.storage.repoPath, RepoDir, IndexFile)
	
	data, err := os.ReadFile(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Index doesn't exist yet, start with empty index
			idx.Entries = make([]IndexEntry, 0)
			return nil
		}
		return fmt.Errorf("failed to read index: %w", err)
	}

	if len(data) == 0 || string(data) == "{}" {
		idx.Entries = make([]IndexEntry, 0)
		return nil
	}

	var indexData struct {
		Entries []IndexEntry `json:"entries"`
	}

	if err := json.Unmarshal(data, &indexData); err != nil {
		return fmt.Errorf("failed to unmarshal index: %w", err)
	}

	idx.Entries = indexData.Entries
	return nil
}

// SaveIndex writes the index to disk
func (idx *Index) SaveIndex() error {
	indexPath := filepath.Join(idx.storage.repoPath, RepoDir, IndexFile)
	
	indexData := struct {
		Entries []IndexEntry `json:"entries"`
	}{
		Entries: idx.Entries,
	}

	data, err := json.MarshalIndent(indexData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal index: %w", err)
	}

	return os.WriteFile(indexPath, data, 0644)
}

// AddEntry adds a new commit to the index
func (idx *Index) AddEntry(hash, message, parent string, timestamp time.Time) error {
	entry := IndexEntry{
		Hash:      hash,
		Timestamp: timestamp,
		Message:   message,
		Parent:    parent,
	}

	idx.Entries = append(idx.Entries, entry)
	return idx.SaveIndex()
}

// GetOrderedCommits returns commits in chronological order
func (idx *Index) GetOrderedCommits(limit int) []IndexEntry {
	// Since entries are added chronologically, we can return them in reverse order
	// for most recent first
	entries := make([]IndexEntry, 0)
	
	start := len(idx.Entries) - limit
	if start < 0 {
		start = 0
	}

	for i := len(idx.Entries) - 1; i >= start; i-- {
		entries = append(entries, idx.Entries[i])
	}

	return entries
}

// GetEntry retrieves an index entry by hash
func (idx *Index) GetEntry(hash string) *IndexEntry {
	for _, entry := range idx.Entries {
		if entry.Hash == hash {
			return &entry
		}
	}
	return nil
}

// GetHead returns the most recent commit hash
func (idx *Index) GetHead() string {
	if len(idx.Entries) == 0 {
		return ""
	}
	return idx.Entries[len(idx.Entries)-1].Hash
}

// RebuildIndex reconstructs the index from all commits in storage
func (idx *Index) RebuildIndex() error {
	hashes, err := idx.storage.ListCommits()
	if err != nil {
		return fmt.Errorf("failed to list commits: %w", err)
	}

	idx.Entries = make([]IndexEntry, 0, len(hashes))

	// Load all commits and build index entries
	for _, hash := range hashes {
		commit, err := idx.storage.ReadCommit(hash)
		if err != nil {
			return fmt.Errorf("failed to read commit %s: %w", hash, err)
		}

		entry := IndexEntry{
			Hash:      commit.Hash,
			Timestamp: commit.Timestamp,
			Message:   commit.Message,
			Parent:    commit.Parent,
		}

		idx.Entries = append(idx.Entries, entry)
	}

	// Sort entries by timestamp to maintain chronological order
	for i := 0; i < len(idx.Entries)-1; i++ {
		for j := i + 1; j < len(idx.Entries); j++ {
			if idx.Entries[i].Timestamp.After(idx.Entries[j].Timestamp) {
				idx.Entries[i], idx.Entries[j] = idx.Entries[j], idx.Entries[i]
			}
		}
	}

	return idx.SaveIndex()
}