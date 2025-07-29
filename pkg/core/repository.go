package core

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/livecodegit/pkg/storage"
)

// LiveCodeRepository implements the RepositoryInterface for livecoding version control
type LiveCodeRepository struct {
	path               string
	storage            StorageInterface
	index              *storage.Index
	currentPerformance *Performance
}

// NewRepository creates a new LiveCodeGit repository instance
func NewRepository(path string) *LiveCodeRepository {
	fsStorage := storage.NewFileSystemStorage(path)
	index := storage.NewIndex(fsStorage)
	
	return &LiveCodeRepository{
		path:    path,
		storage: fsStorage,
		index:   index,
	}
}

// Init initializes a new LiveCodeGit repository
func (repo *LiveCodeRepository) Init(path string) error {
	repo.path = path
	
	// Check if repository already exists
	repoDir := filepath.Join(path, storage.RepoDir)
	if _, err := os.Stat(repoDir); err == nil {
		return fmt.Errorf("repository already exists at %s", path)
	}

	// Initialize storage
	fsStorage := storage.NewFileSystemStorage(path)
	if err := fsStorage.InitializeRepository(); err != nil {
		return fmt.Errorf("failed to initialize repository: %w", err)
	}

	// Initialize index
	repo.storage = fsStorage
	repo.index = storage.NewIndex(fsStorage)
	
	if err := repo.index.LoadIndex(); err != nil {
		return fmt.Errorf("failed to initialize index: %w", err)
	}

	return nil
}

// Commit creates a new commit with the given content and metadata
func (repo *LiveCodeRepository) Commit(content string, message string, metadata ExecutionMetadata) (*Commit, error) {
	if !repo.IsInitialized() {
		return nil, fmt.Errorf("repository not initialized")
	}

	// Load index if not already loaded
	if repo.index == nil {
		repo.index = storage.NewIndex(repo.storage.(*storage.FileSystemStorage))
		if err := repo.index.LoadIndex(); err != nil {
			return nil, fmt.Errorf("failed to load index: %w", err)
		}
	}

	// Generate hash from content
	hash := storage.GenerateHash(content + message + time.Now().String())
	
	// Get parent commit
	parentHash := repo.index.GetHead()

	// Create commit
	commit := &Commit{
		Hash:      hash,
		Parent:    parentHash,
		Timestamp: time.Now(),
		Message:   message,
		Author:    "livecoder", // TODO: Get from config
		Content:   content,
		Metadata:  metadata,
	}

	// Store commit
	if err := repo.storage.WriteCommit(commit); err != nil {
		return nil, fmt.Errorf("failed to write commit: %w", err)
	}

	// Update index
	if err := repo.index.AddEntry(hash, message, parentHash, commit.Timestamp); err != nil {
		return nil, fmt.Errorf("failed to update index: %w", err)
	}

	// Update HEAD
	if fsStorage, ok := repo.storage.(*storage.FileSystemStorage); ok {
		if err := fsStorage.WriteHead(hash); err != nil {
			return nil, fmt.Errorf("failed to update HEAD: %w", err)
		}
	}

	// Update current performance if active
	if repo.currentPerformance != nil {
		repo.currentPerformance.CommitCount++
		repo.currentPerformance.HeadCommit = hash
		if err := repo.storage.WritePerformance(repo.currentPerformance); err != nil {
			return nil, fmt.Errorf("failed to update performance: %w", err)
		}
	}

	return commit, nil
}

// Log returns the commit history with optional limit
func (repo *LiveCodeRepository) Log(limit int) ([]*Commit, error) {
	if !repo.IsInitialized() {
		return nil, fmt.Errorf("repository not initialized")
	}

	// Load index if not already loaded
	if repo.index == nil {
		repo.index = storage.NewIndex(repo.storage.(*storage.FileSystemStorage))
		if err := repo.index.LoadIndex(); err != nil {
			return nil, fmt.Errorf("failed to load index: %w", err)
		}
	}

	if limit <= 0 {
		limit = 50 // Default limit
	}

	entries := repo.index.GetOrderedCommits(limit)
	commits := make([]*Commit, 0, len(entries))

	for _, entry := range entries {
		commit, err := repo.storage.ReadCommit(entry.Hash)
		if err != nil {
			return nil, fmt.Errorf("failed to read commit %s: %w", entry.Hash, err)
		}
		commits = append(commits, commit)
	}

	return commits, nil
}

// GetCommit retrieves a specific commit by hash
func (repo *LiveCodeRepository) GetCommit(hash string) (*Commit, error) {
	if repo.storage == nil {
		return nil, fmt.Errorf("repository not initialized")
	}

	return repo.storage.ReadCommit(hash)
}

// GetCurrentPerformance returns the active performance session
func (repo *LiveCodeRepository) GetCurrentPerformance() (*Performance, error) {
	return repo.currentPerformance, nil
}

// StartPerformance begins a new performance session
func (repo *LiveCodeRepository) StartPerformance(name string) (*Performance, error) {
	if repo.storage == nil {
		return nil, fmt.Errorf("repository not initialized")
	}

	// End current performance if active
	if repo.currentPerformance != nil {
		if err := repo.EndPerformance(); err != nil {
			return nil, fmt.Errorf("failed to end current performance: %w", err)
		}
	}

	// Create new performance
	performance := &Performance{
		ID:          fmt.Sprintf("perf-%d", time.Now().Unix()),
		Name:        name,
		StartTime:   time.Now(),
		CommitCount: 0,
		Branch:      "main", // TODO: Support branches
		Author:      "livecoder", // TODO: Get from config
	}

	if err := repo.storage.WritePerformance(performance); err != nil {
		return nil, fmt.Errorf("failed to write performance: %w", err)
	}

	repo.currentPerformance = performance
	return performance, nil
}

// EndPerformance concludes the current performance session
func (repo *LiveCodeRepository) EndPerformance() error {
	if repo.currentPerformance == nil {
		return fmt.Errorf("no active performance session")
	}

	repo.currentPerformance.EndTime = time.Now()
	if err := repo.storage.WritePerformance(repo.currentPerformance); err != nil {
		return fmt.Errorf("failed to update performance end time: %w", err)
	}

	repo.currentPerformance = nil
	return nil
}

// IsInitialized checks if the repository is properly initialized
func (repo *LiveCodeRepository) IsInitialized() bool {
	repoDir := filepath.Join(repo.path, storage.RepoDir)
	_, err := os.Stat(repoDir)
	return err == nil
}

// LoadRepository loads an existing repository from the given path
func LoadRepository(path string) (*LiveCodeRepository, error) {
	repo := NewRepository(path)
	
	if !repo.IsInitialized() {
		return nil, fmt.Errorf("no repository found at %s", path)
	}

	// Load index
	if err := repo.index.LoadIndex(); err != nil {
		return nil, fmt.Errorf("failed to load repository index: %w", err)
	}

	return repo, nil
}