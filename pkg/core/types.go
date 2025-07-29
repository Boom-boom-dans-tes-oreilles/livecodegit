package core

import (
	"github.com/livecodegit/pkg/storage"
)

// Type aliases for convenience
type Commit = storage.Commit
type ExecutionMetadata = storage.ExecutionMetadata
type Performance = storage.Performance

// Repository represents a livecoding performance repository
type Repository struct {
	Path         string `json:"path"`
	Initialized  bool   `json:"initialized"`
	CurrentBranch string `json:"current_branch"`
	HeadCommit   string `json:"head_commit,omitempty"`
}

// RepositoryInterface defines the core operations for a livecoding repository
type RepositoryInterface interface {
	Init(path string) error
	Commit(content string, message string, metadata ExecutionMetadata) (*Commit, error)
	Log(limit int) ([]*Commit, error)
	GetCommit(hash string) (*Commit, error)
	GetCurrentPerformance() (*Performance, error)
	StartPerformance(name string) (*Performance, error)
	EndPerformance() error
}

// StorageInterface defines the storage operations for commits and metadata
type StorageInterface interface {
	WriteCommit(commit *Commit) error
	ReadCommit(hash string) (*Commit, error)
	WritePerformance(performance *Performance) error
	ReadPerformance(id string) (*Performance, error)
	ListCommits() ([]string, error)
	Exists(hash string) bool
}