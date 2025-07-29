package storage

import (
	"testing"
	"time"
)

func TestCommitCreation(t *testing.T) {
	metadata := ExecutionMetadata{
		Buffer:      "main",
		Language:    "sonicpi",
		BPM:         120.0,
		BeatsFromStart: 16,
		Success:     true,
		Environment: "sonic-pi",
	}

	commit := Commit{
		Hash:      "abc123",
		Parent:    "def456",
		Timestamp: time.Now(),
		Message:   "Test commit",
		Author:    "testuser",
		Content:   "live_loop :drums do\n  sample :bd_haus\nend",
		Metadata:  metadata,
	}

	if commit.Hash != "abc123" {
		t.Errorf("Expected hash 'abc123', got '%s'", commit.Hash)
	}

	if commit.Metadata.Language != "sonicpi" {
		t.Errorf("Expected language 'sonicpi', got '%s'", commit.Metadata.Language)
	}

	if commit.Metadata.BPM != 120.0 {
		t.Errorf("Expected BPM 120.0, got %f", commit.Metadata.BPM)
	}
}

func TestPerformanceCreation(t *testing.T) {
	startTime := time.Now()
	performance := Performance{
		ID:          "perf-123",
		Name:        "Test Performance",
		StartTime:   startTime,
		CommitCount: 5,
		HeadCommit:  "abc123",
		Branch:      "main",
		Author:      "testuser",
		Description: "A test performance",
	}

	if performance.ID != "perf-123" {
		t.Errorf("Expected ID 'perf-123', got '%s'", performance.ID)
	}

	if performance.CommitCount != 5 {
		t.Errorf("Expected commit count 5, got %d", performance.CommitCount)
	}

	if performance.StartTime != startTime {
		t.Errorf("Expected start time to match, got different time")
	}
}

func TestExecutionMetadata(t *testing.T) {
	metadata := ExecutionMetadata{
		Buffer:        "bass",
		Language:      "tidal",
		BPM:           140.5,
		BeatsFromStart: 32,
		Success:       false,
		ErrorMessage:  "Syntax error on line 3",
		Environment:   "tidal-cycles",
	}

	if metadata.Buffer != "bass" {
		t.Errorf("Expected buffer 'bass', got '%s'", metadata.Buffer)
	}

	if metadata.Success != false {
		t.Errorf("Expected success to be false, got %t", metadata.Success)
	}

	if metadata.ErrorMessage != "Syntax error on line 3" {
		t.Errorf("Expected error message 'Syntax error on line 3', got '%s'", metadata.ErrorMessage)
	}
}