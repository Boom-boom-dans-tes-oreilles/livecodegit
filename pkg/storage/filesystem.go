package storage

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	RepoDir        = ".livecodegit"
	ObjectsDir     = "objects"
	PerformanceDir = "performances"
	IndexFile      = "index"
	HeadFile       = "HEAD"
)

// Commit represents a single execution state in a livecoding performance
type Commit struct {
	Hash      string            `json:"hash"`
	Parent    string            `json:"parent,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
	Message   string            `json:"message"`
	Author    string            `json:"author"`
	Content   string            `json:"content"`
	Metadata  ExecutionMetadata `json:"metadata"`
}

// ExecutionMetadata contains performance-specific information about code execution
type ExecutionMetadata struct {
	Buffer        string  `json:"buffer"`
	Language      string  `json:"language"`
	BPM           float64 `json:"bpm,omitempty"`
	BeatsFromStart int64  `json:"beats_from_start,omitempty"`
	Success       bool    `json:"success"`
	ErrorMessage  string  `json:"error_message,omitempty"`
	Environment   string  `json:"environment,omitempty"`
}

// Performance represents a complete livecoding session
type Performance struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time,omitempty"`
	CommitCount int       `json:"commit_count"`
	HeadCommit  string    `json:"head_commit"`
	Branch      string    `json:"branch"`
	Author      string    `json:"author"`
	Description string    `json:"description,omitempty"`
}

// FileSystemStorage implements git-like object storage for livecoding commits
type FileSystemStorage struct {
	repoPath string
}

// NewFileSystemStorage creates a new filesystem-based storage instance
func NewFileSystemStorage(repoPath string) *FileSystemStorage {
	return &FileSystemStorage{
		repoPath: repoPath,
	}
}

// WriteCommit stores a commit object using content-addressable storage
func (fs *FileSystemStorage) WriteCommit(commit *Commit) error {
	objectsPath := filepath.Join(fs.repoPath, RepoDir, ObjectsDir)
	if err := os.MkdirAll(objectsPath, 0755); err != nil {
		return fmt.Errorf("failed to create objects directory: %w", err)
	}

	// Create hash-based directory structure (first 2 chars as subdirectory)
	hashPrefix := commit.Hash[:2]
	hashSuffix := commit.Hash[2:]
	objDir := filepath.Join(objectsPath, hashPrefix)
	
	if err := os.MkdirAll(objDir, 0755); err != nil {
		return fmt.Errorf("failed to create object subdirectory: %w", err)
	}

	objPath := filepath.Join(objDir, hashSuffix)
	
	// Serialize commit to JSON
	data, err := json.MarshalIndent(commit, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal commit: %w", err)
	}

	return os.WriteFile(objPath, data, 0644)
}

// ReadCommit retrieves a commit object by its hash
func (fs *FileSystemStorage) ReadCommit(hash string) (*Commit, error) {
	objPath := fs.getObjectPath(hash)
	
	data, err := os.ReadFile(objPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read commit %s: %w", hash, err)
	}

	var commit Commit
	if err := json.Unmarshal(data, &commit); err != nil {
		return nil, fmt.Errorf("failed to unmarshal commit %s: %w", hash, err)
	}

	return &commit, nil
}

// WritePerformance stores performance metadata
func (fs *FileSystemStorage) WritePerformance(performance *Performance) error {
	perfDir := filepath.Join(fs.repoPath, RepoDir, PerformanceDir)
	if err := os.MkdirAll(perfDir, 0755); err != nil {
		return fmt.Errorf("failed to create performances directory: %w", err)
	}

	perfPath := filepath.Join(perfDir, performance.ID+".json")
	
	data, err := json.MarshalIndent(performance, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal performance: %w", err)
	}

	return os.WriteFile(perfPath, data, 0644)
}

// ReadPerformance retrieves performance metadata by ID
func (fs *FileSystemStorage) ReadPerformance(id string) (*Performance, error) {
	perfPath := filepath.Join(fs.repoPath, RepoDir, PerformanceDir, id+".json")
	
	data, err := os.ReadFile(perfPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read performance %s: %w", id, err)
	}

	var performance Performance
	if err := json.Unmarshal(data, &performance); err != nil {
		return nil, fmt.Errorf("failed to unmarshal performance %s: %w", id, err)
	}

	return &performance, nil
}

// ListCommits returns all commit hashes in the repository
func (fs *FileSystemStorage) ListCommits() ([]string, error) {
	objectsPath := filepath.Join(fs.repoPath, RepoDir, ObjectsDir)
	var commits []string

	err := filepath.WalkDir(objectsPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			// Reconstruct hash from directory structure
			rel, err := filepath.Rel(objectsPath, path)
			if err != nil {
				return err
			}
			
			parts := strings.Split(rel, string(filepath.Separator))
			if len(parts) == 2 {
				hash := parts[0] + parts[1]
				commits = append(commits, hash)
			}
		}

		return nil
	})

	return commits, err
}

// Exists checks if a commit object exists
func (fs *FileSystemStorage) Exists(hash string) bool {
	objPath := fs.getObjectPath(hash)
	_, err := os.Stat(objPath)
	return err == nil
}

// GenerateHash creates a SHA-1 hash for commit content
func GenerateHash(content string) string {
	hash := sha1.Sum([]byte(content))
	return fmt.Sprintf("%x", hash)
}

// WriteHead updates the HEAD reference
func (fs *FileSystemStorage) WriteHead(commitHash string) error {
	headPath := filepath.Join(fs.repoPath, RepoDir, HeadFile)
	return os.WriteFile(headPath, []byte(commitHash), 0644)
}

// ReadHead reads the current HEAD reference
func (fs *FileSystemStorage) ReadHead() (string, error) {
	headPath := filepath.Join(fs.repoPath, RepoDir, HeadFile)
	data, err := os.ReadFile(headPath)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// InitializeRepository creates the basic repository structure
func (fs *FileSystemStorage) InitializeRepository() error {
	repoDir := filepath.Join(fs.repoPath, RepoDir)
	
	dirs := []string{
		repoDir,
		filepath.Join(repoDir, ObjectsDir),
		filepath.Join(repoDir, PerformanceDir),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Create empty index file
	indexPath := filepath.Join(repoDir, IndexFile)
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		if err := os.WriteFile(indexPath, []byte("{}"), 0644); err != nil {
			return fmt.Errorf("failed to create index file: %w", err)
		}
	}

	return nil
}

// getObjectPath constructs the file path for a commit object
func (fs *FileSystemStorage) getObjectPath(hash string) string {
	hashPrefix := hash[:2]
	hashSuffix := hash[2:]
	return filepath.Join(fs.repoPath, RepoDir, ObjectsDir, hashPrefix, hashSuffix)
}