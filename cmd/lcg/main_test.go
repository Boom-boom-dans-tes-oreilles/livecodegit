package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Helper function to create a temporary directory for testing
func createTempDir(t *testing.T) string {
	tempDir, err := os.MkdirTemp("", "livecodegit-cli-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	return tempDir
}

// Helper function to build the CLI binary for testing
func buildCLI(t *testing.T) string {
	tmpBinary := filepath.Join(t.TempDir(), "lcg-test")
	cmd := exec.Command("go", "build", "-o", tmpBinary, "./cmd/lcg")
	
	// Find the project root by looking for go.mod
	projectRoot := "."
	for i := 0; i < 5; i++ { // Limit search depth
		if _, err := os.Stat(filepath.Join(projectRoot, "go.mod")); err == nil {
			break
		}
		projectRoot = filepath.Join("..", projectRoot)
	}
	
	cmd.Dir = projectRoot
	
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build CLI binary: %v", err)
	}
	
	return tmpBinary
}

// Helper function to run CLI command
func runCLI(t *testing.T, binary string, args []string, workDir string) (string, string, error) {
	cmd := exec.Command(binary, args...)
	if workDir != "" {
		cmd.Dir = workDir
	}
	
	stdout, err := cmd.Output()
	stderr := ""
	
	if exitError, ok := err.(*exec.ExitError); ok {
		stderr = string(exitError.Stderr)
	}
	
	return string(stdout), stderr, err
}

func TestCLIVersion(t *testing.T) {
	binary := buildCLI(t)
	
	stdout, _, err := runCLI(t, binary, []string{"version"}, "")
	if err != nil {
		t.Fatalf("Failed to run version command: %v", err)
	}
	
	if !strings.Contains(stdout, "LiveCodeGit version") {
		t.Errorf("Expected version output to contain 'LiveCodeGit version', got: %s", stdout)
	}
}

func TestCLIHelp(t *testing.T) {
	binary := buildCLI(t)
	
	// Test help command
	stdout, _, err := runCLI(t, binary, []string{"help"}, "")
	if err != nil {
		t.Fatalf("Failed to run help command: %v", err)
	}
	
	expectedStrings := []string{
		"LiveCodeGit - A Git-like Version Control System for Livecoding",
		"Usage: lcg <command>",
		"init",
		"commit",
		"log",
	}
	
	for _, expected := range expectedStrings {
		if !strings.Contains(stdout, expected) {
			t.Errorf("Expected help output to contain '%s', got: %s", expected, stdout)
		}
	}
}

func TestCLIInit(t *testing.T) {
	binary := buildCLI(t)
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)
	
	stdout, _, err := runCLI(t, binary, []string{"init"}, tempDir)
	if err != nil {
		t.Fatalf("Failed to run init command: %v", err)
	}
	
	if !strings.Contains(stdout, "Initialized empty LiveCodeGit repository") {
		t.Errorf("Expected init output to contain 'Initialized empty LiveCodeGit repository', got: %s", stdout)
	}
	
	// Check that repository directory was created
	repoDir := filepath.Join(tempDir, ".livecodegit")
	if _, err := os.Stat(repoDir); os.IsNotExist(err) {
		t.Errorf("Repository directory was not created")
	}
	
	// Check required subdirectories
	subdirs := []string{"objects", "performances"}
	for _, subdir := range subdirs {
		path := filepath.Join(repoDir, subdir)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Required subdirectory '%s' was not created", subdir)
		}
	}
}

func TestCLIInitTwice(t *testing.T) {
	binary := buildCLI(t)
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)
	
	// Initialize once
	_, _, err := runCLI(t, binary, []string{"init"}, tempDir)
	if err != nil {
		t.Fatalf("Failed to run first init command: %v", err)
	}
	
	// Try to initialize again
	_, stderr, err := runCLI(t, binary, []string{"init"}, tempDir)
	if err == nil {
		t.Errorf("Expected error when initializing existing repository")
	}
	
	if !strings.Contains(stderr, "repository already exists") {
		t.Errorf("Expected error message about existing repository, got: %s", stderr)
	}
}

func TestCLICommit(t *testing.T) {
	binary := buildCLI(t)
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)
	
	// Initialize repository first
	_, _, err := runCLI(t, binary, []string{"init"}, tempDir)
	if err != nil {
		t.Fatalf("Failed to initialize repository: %v", err)
	}
	
	// Create commit
	args := []string{
		"commit",
		"-m", "Test commit message",
		"-c", "live_loop :drums do\n  sample :bd_haus\nend",
		"-l", "sonicpi",
		"-b", "main",
	}
	
	stdout, _, err := runCLI(t, binary, args, tempDir)
	if err != nil {
		t.Fatalf("Failed to run commit command: %v", err)
	}
	
	if !strings.Contains(stdout, "Created commit") {
		t.Errorf("Expected commit output to contain 'Created commit', got: %s", stdout)
	}
	
	if !strings.Contains(stdout, "Test commit message") {
		t.Errorf("Expected commit output to contain commit message, got: %s", stdout)
	}
}

func TestCLICommitWithoutRepo(t *testing.T) {
	binary := buildCLI(t)
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)
	
	// Try to commit without initializing repository
	args := []string{
		"commit",
		"-m", "Test commit",
		"-c", "test code",
	}
	
	_, stderr, err := runCLI(t, binary, args, tempDir)
	if err == nil {
		t.Errorf("Expected error when committing without repository")
	}
	
	if !strings.Contains(stderr, "Make sure you're in a LiveCodeGit repository") {
		t.Errorf("Expected error message about missing repository, got: %s", stderr)
	}
}

func TestCLICommitMissingMessage(t *testing.T) {
	binary := buildCLI(t)
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)
	
	// Initialize repository
	_, _, err := runCLI(t, binary, []string{"init"}, tempDir)
	if err != nil {
		t.Fatalf("Failed to initialize repository: %v", err)
	}
	
	// Try to commit without message
	args := []string{
		"commit",
		"-c", "test code",
	}
	
	_, stderr, err := runCLI(t, binary, args, tempDir)
	if err == nil {
		t.Errorf("Expected error when committing without message")
	}
	
	if !strings.Contains(stderr, "commit message is required") {
		t.Errorf("Expected error message about missing commit message, got: %s", stderr)
	}
}

func TestCLICommitMissingContent(t *testing.T) {
	binary := buildCLI(t)
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)
	
	// Initialize repository
	_, _, err := runCLI(t, binary, []string{"init"}, tempDir)
	if err != nil {
		t.Fatalf("Failed to initialize repository: %v", err)
	}
	
	// Try to commit without content
	args := []string{
		"commit",
		"-m", "Test message",
	}
	
	_, stderr, err := runCLI(t, binary, args, tempDir)
	if err == nil {
		t.Errorf("Expected error when committing without content")
	}
	
	if !strings.Contains(stderr, "code content is required") {
		t.Errorf("Expected error message about missing code content, got: %s", stderr)
	}
}

func TestCLILog(t *testing.T) {
	binary := buildCLI(t)
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)
	
	// Initialize repository
	_, _, err := runCLI(t, binary, []string{"init"}, tempDir)
	if err != nil {
		t.Fatalf("Failed to initialize repository: %v", err)
	}
	
	// Create multiple commits
	commits := []struct {
		message string
		content string
		lang    string
		buffer  string
	}{
		{"First commit", "live_loop :drums do\n  sample :bd_haus\nend", "sonicpi", "drums"},
		{"Second commit", "live_loop :bass do\n  synth :tb303\nend", "sonicpi", "bass"},
		{"Third commit", "live_loop :melody do\n  play scale(:c, :major).choose\nend", "sonicpi", "melody"},
	}
	
	for _, commit := range commits {
		args := []string{
			"commit",
			"-m", commit.message,
			"-c", commit.content,
			"-l", commit.lang,
			"-b", commit.buffer,
		}
		
		_, _, err := runCLI(t, binary, args, tempDir)
		if err != nil {
			t.Fatalf("Failed to create commit '%s': %v", commit.message, err)
		}
	}
	
	// Get log
	stdout, _, err := runCLI(t, binary, []string{"log"}, tempDir)
	if err != nil {
		t.Fatalf("Failed to run log command: %v", err)
	}
	
	// Check that all commits are present in reverse chronological order
	expectedMessages := []string{"Third commit", "Second commit", "First commit"}
	for _, expected := range expectedMessages {
		if !strings.Contains(stdout, expected) {
			t.Errorf("Expected log to contain '%s', got: %s", expected, stdout)
		}
	}
	
	// Check that commit information is displayed
	expectedInfo := []string{
		"commit ",
		"Date: ",
		"Author: livecoder",
		"Language: sonicpi",
		"Buffer: ",
	}
	
	for _, expected := range expectedInfo {
		if !strings.Contains(stdout, expected) {
			t.Errorf("Expected log to contain '%s', got: %s", expected, stdout)
		}
	}
}

func TestCLILogWithLimit(t *testing.T) {
	binary := buildCLI(t)
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)
	
	// Initialize repository
	_, _, err := runCLI(t, binary, []string{"init"}, tempDir)
	if err != nil {
		t.Fatalf("Failed to initialize repository: %v", err)
	}
	
	// Create multiple commits
	for i := 1; i <= 5; i++ {
		args := []string{
			"commit",
			"-m", fmt.Sprintf("Commit %d", i),
			"-c", "test code",
			"-l", "sonicpi",
		}
		
		_, _, err := runCLI(t, binary, args, tempDir)
		if err != nil {
			t.Fatalf("Failed to create commit %d: %v", i, err)
		}
	}
	
	// Get limited log
	stdout, _, err := runCLI(t, binary, []string{"log", "-n", "3"}, tempDir)
	if err != nil {
		t.Fatalf("Failed to run log command with limit: %v", err)
	}
	
	// Count number of commits in output
	commitCount := strings.Count(stdout, "commit ")
	if commitCount != 3 {
		t.Errorf("Expected 3 commits in limited log, got %d", commitCount)
	}
	
	// Check that most recent commits are shown
	if !strings.Contains(stdout, "Commit 5") {
		t.Errorf("Expected most recent commit (Commit 5) in limited log")
	}
	
	if strings.Contains(stdout, "Commit 1") {
		t.Errorf("Should not contain oldest commit (Commit 1) in limited log")
	}
}

func TestCLILogEmptyRepository(t *testing.T) {
	binary := buildCLI(t)
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)
	
	// Initialize repository
	_, _, err := runCLI(t, binary, []string{"init"}, tempDir)
	if err != nil {
		t.Fatalf("Failed to initialize repository: %v", err)
	}
	
	// Get log from empty repository
	stdout, _, err := runCLI(t, binary, []string{"log"}, tempDir)
	if err != nil {
		t.Fatalf("Failed to run log command on empty repository: %v", err)
	}
	
	if !strings.Contains(stdout, "No commits found") {
		t.Errorf("Expected 'No commits found' message, got: %s", stdout)
	}
}

func TestCLILogWithoutRepo(t *testing.T) {
	binary := buildCLI(t)
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)
	
	// Try to get log without repository
	_, stderr, err := runCLI(t, binary, []string{"log"}, tempDir)
	if err == nil {
		t.Errorf("Expected error when getting log without repository")
	}
	
	if !strings.Contains(stderr, "Make sure you're in a LiveCodeGit repository") {
		t.Errorf("Expected error message about missing repository, got: %s", stderr)
	}
}

func TestCLIUnknownCommand(t *testing.T) {
	binary := buildCLI(t)
	
	_, stderr, err := runCLI(t, binary, []string{"unknown"}, "")
	if err == nil {
		t.Errorf("Expected error for unknown command")
	}
	
	if !strings.Contains(stderr, "Unknown command: unknown") {
		t.Errorf("Expected error message about unknown command, got: %s", stderr)
	}
}

func TestCLINoCommand(t *testing.T) {
	binary := buildCLI(t)
	
	_, stderr, err := runCLI(t, binary, []string{}, "")
	if err == nil {
		t.Errorf("Expected error when no command provided")
	}
	
	// Should show usage information
	if !strings.Contains(stderr, "Usage: lcg <command>") {
		t.Errorf("Expected usage information when no command provided, got: %s", stderr)
	}
}