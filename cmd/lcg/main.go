package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/livecodegit/pkg/core"
)

const (
	version = "0.1.0"
)

func main() {
	if len(os.Args) < 2 {
		printUsageToStderr()
		os.Exit(1)
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "init":
		handleInit(args)
	case "commit":
		handleCommit(args)
	case "log":
		handleLog(args)
	case "watch":
		handleWatch(args)
	case "version":
		fmt.Printf("LiveCodeGit version %s\n", version)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsageToStderr()
		os.Exit(1)
	}
}

func handleInit(args []string) {
	var path string

	if len(args) > 0 {
		path = args[0]
	} else {
		var err error
		path, err = os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting current directory: %v\n", err)
			os.Exit(1)
		}
	}

	repo := core.NewRepository(path)
	if err := repo.Init(path); err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing repository: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Initialized empty LiveCodeGit repository in %s\n", path)
}

func handleCommit(args []string) {
	commitFlags := flag.NewFlagSet("commit", flag.ExitOnError)
	message := commitFlags.String("m", "", "Commit message")
	content := commitFlags.String("c", "", "Code content to commit")
	language := commitFlags.String("l", "unknown", "Programming language")
	buffer := commitFlags.String("b", "main", "Buffer name")

	commitFlags.Parse(args)

	if *message == "" {
		fmt.Fprintf(os.Stderr, "Error: commit message is required (-m)\n")
		os.Exit(1)
	}

	if *content == "" {
		fmt.Fprintf(os.Stderr, "Error: code content is required (-c)\n")
		os.Exit(1)
	}

	// Get current directory
	path, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting current directory: %v\n", err)
		os.Exit(1)
	}

	// Load repository
	repo, err := core.LoadRepository(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading repository: %v\n", err)
		fmt.Fprintf(os.Stderr, "Make sure you're in a LiveCodeGit repository (run 'lcg init' first)\n")
		os.Exit(1)
	}

	// Create execution metadata
	metadata := core.ExecutionMetadata{
		Buffer:      *buffer,
		Language:    *language,
		Success:     true,
		Environment: "cli",
	}

	// Create commit
	commit, err := repo.Commit(*content, *message, metadata)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating commit: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Created commit %s\n", commit.Hash[:8])
	fmt.Printf("Message: %s\n", commit.Message)
}

func handleLog(args []string) {
	logFlags := flag.NewFlagSet("log", flag.ExitOnError)
	limit := logFlags.Int("n", 10, "Number of commits to show")

	logFlags.Parse(args)

	// Get current directory
	path, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting current directory: %v\n", err)
		os.Exit(1)
	}

	// Load repository
	repo, err := core.LoadRepository(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading repository: %v\n", err)
		fmt.Fprintf(os.Stderr, "Make sure you're in a LiveCodeGit repository (run 'lcg init' first)\n")
		os.Exit(1)
	}

	// Get commit log
	commits, err := repo.Log(*limit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error retrieving commit log: %v\n", err)
		os.Exit(1)
	}

	if len(commits) == 0 {
		fmt.Println("No commits found")
		return
	}

	// Display commits
	for i, commit := range commits {
		fmt.Printf("commit %s", commit.Hash)
		if commit.Parent != "" {
			fmt.Printf(" (parent: %s)", commit.Parent[:8])
		}
		fmt.Printf("\n")
		fmt.Printf("Date: %s\n", commit.Timestamp.Format("Mon Jan 2 15:04:05 2006"))
		fmt.Printf("Author: %s\n", commit.Author)
		fmt.Printf("Language: %s\n", commit.Metadata.Language)
		fmt.Printf("Buffer: %s\n", commit.Metadata.Buffer)
		fmt.Printf("\n    %s\n", commit.Message)

		if i < len(commits)-1 {
			fmt.Println()
		}
	}
}

func printUsage() {
	fmt.Printf("LiveCodeGit - A Git-like Version Control System for Livecoding\n\n")
	fmt.Printf("Usage: lcg <command> [options]\n\n")
	fmt.Printf("Commands:\n")
	fmt.Printf("  init [path]           Initialize a new repository\n")
	fmt.Printf("  commit                Create a new commit\n")
	fmt.Printf("    -m <message>        Commit message (required)\n")
	fmt.Printf("    -c <content>        Code content (required)\n")
	fmt.Printf("    -l <language>       Programming language (default: unknown)\n")
	fmt.Printf("    -b <buffer>         Buffer name (default: main)\n")
	fmt.Printf("  log                   Show commit history\n")
	fmt.Printf("    -n <number>         Number of commits to show (default: 10)\n")
	fmt.Printf("  watch                 Start watching for code executions\n")
	fmt.Printf("    --lang <language>   Watch specific language (sonicpi, tidal)\n")
	fmt.Printf("    --list              List available watchers\n")
	fmt.Printf("    --status            Show watcher status\n")
	fmt.Printf("    --enable <name>     Enable a watcher\n")
	fmt.Printf("    --disable <name>    Disable a watcher\n")
	fmt.Printf("  version               Show version information\n")
	fmt.Printf("  help                  Show this help message\n\n")
	fmt.Printf("Examples:\n")
	fmt.Printf("  lcg init                                    # Initialize repository in current directory\n")
	fmt.Printf("  lcg init /path/to/project                   # Initialize repository in specific path\n")
	fmt.Printf("  lcg commit -m \"Add bass line\" -c \"bass.play\" -l sonicpi\n")
	fmt.Printf("  lcg log -n 5                                # Show last 5 commits\n")
	fmt.Printf("  lcg watch --lang sonicpi                    # Start watching Sonic Pi executions\n")
	fmt.Printf("  lcg watch --list                            # List available watchers\n")
	fmt.Printf("  lcg watch --enable sonicpi-osc              # Enable Sonic Pi OSC watcher\n")
}

func printUsageToStderr() {
	fmt.Fprintf(os.Stderr, "LiveCodeGit - A Git-like Version Control System for Livecoding\n\n")
	fmt.Fprintf(os.Stderr, "Usage: lcg <command> [options]\n\n")
	fmt.Fprintf(os.Stderr, "Commands:\n")
	fmt.Fprintf(os.Stderr, "  init [path]           Initialize a new repository\n")
	fmt.Fprintf(os.Stderr, "  commit                Create a new commit\n")
	fmt.Fprintf(os.Stderr, "    -m <message>        Commit message (required)\n")
	fmt.Fprintf(os.Stderr, "    -c <content>        Code content (required)\n")
	fmt.Fprintf(os.Stderr, "    -l <language>       Programming language (default: unknown)\n")
	fmt.Fprintf(os.Stderr, "    -b <buffer>         Buffer name (default: main)\n")
	fmt.Fprintf(os.Stderr, "  log                   Show commit history\n")
	fmt.Fprintf(os.Stderr, "    -n <number>         Number of commits to show (default: 10)\n")
	fmt.Fprintf(os.Stderr, "  watch                 Start watching for code executions\n")
	fmt.Fprintf(os.Stderr, "    --lang <language>   Watch specific language (sonicpi, tidal)\n")
	fmt.Fprintf(os.Stderr, "    --list              List available watchers\n")
	fmt.Fprintf(os.Stderr, "    --status            Show watcher status\n")
	fmt.Fprintf(os.Stderr, "    --enable <name>     Enable a watcher\n")
	fmt.Fprintf(os.Stderr, "    --disable <name>    Disable a watcher\n")
	fmt.Fprintf(os.Stderr, "  version               Show version information\n")
	fmt.Fprintf(os.Stderr, "  help                  Show this help message\n\n")
	fmt.Fprintf(os.Stderr, "Examples:\n")
	fmt.Fprintf(os.Stderr, "  lcg init                                    # Initialize repository in current directory\n")
	fmt.Fprintf(os.Stderr, "  lcg init /path/to/project                   # Initialize repository in specific path\n")
	fmt.Fprintf(os.Stderr, "  lcg commit -m \"Add bass line\" -c \"bass.play\" -l sonicpi\n")
	fmt.Fprintf(os.Stderr, "  lcg log -n 5                                # Show last 5 commits\n")
	fmt.Fprintf(os.Stderr, "  lcg watch --lang sonicpi                    # Start watching Sonic Pi executions\n")
	fmt.Fprintf(os.Stderr, "  lcg watch --list                            # List available watchers\n")
	fmt.Fprintf(os.Stderr, "  lcg watch --enable sonicpi-osc              # Enable Sonic Pi OSC watcher\n")
}
