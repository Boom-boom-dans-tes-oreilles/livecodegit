# LiveCodeGit

A Git-like version control system designed specifically for capturing and replaying livecoding performances in languages like Sonic Pi, TidalCycles, and SuperCollider.

## Features

- **Automatic Commit Capture**: Seamlessly records code states at each execution
- **Performance Metadata**: Tracks timing, BPM, musical context, and execution success
- **Git-like Storage**: Content-addressable storage with SHA-1 hashing
- **Performance Replay**: Time-accurate playback of coding sessions
- **Multiple Language Support**: Designed for Sonic Pi, TidalCycles, SuperCollider, and more

## Installation

### Build from Source

```bash
# Clone the repository
git clone https://github.com/livecodegit/livecodegit.git
cd livecodegit

# Build the CLI tool
make build

# Install to your PATH
make install
```

### Dependencies

- Go 1.21 or later

## Quick Start

```bash
# Initialize a new LiveCodeGit repository
lcg init

# Create your first commit
lcg commit -m "Initial drums pattern" \
          -c "live_loop :drums do
                sample :bd_haus
                sleep 1
              end" \
          -l sonicpi \
          -b drums

# View your commit history
lcg log

# View recent commits with limit
lcg log -n 5
```

## Usage

### Commands

#### `lcg init [path]`
Initialize a new LiveCodeGit repository in the specified directory (or current directory).

```bash
lcg init                    # Initialize in current directory
lcg init /path/to/project   # Initialize in specific path
```

#### `lcg commit [options]`
Create a new commit with code and metadata.

**Options:**
- `-m <message>`: Commit message (required)
- `-c <content>`: Code content (required)  
- `-l <language>`: Programming language (default: unknown)
- `-b <buffer>`: Buffer name (default: main)

```bash
lcg commit -m "Add bass line" \
          -c "live_loop :bass do
                synth :tb303, note: :c2
                sleep 0.5
              end" \
          -l sonicpi \
          -b bass
```

#### `lcg log [options]`
Display commit history.

**Options:**
- `-n <number>`: Number of commits to show (default: 10)

```bash
lcg log         # Show last 10 commits
lcg log -n 20   # Show last 20 commits
```

### Repository Structure

LiveCodeGit creates a `.livecodegit/` directory with the following structure:

```
.livecodegit/
├── objects/           # Content-addressable commit storage
│   ├── ab/           # First 2 chars of SHA-1 hash
│   │   └── cdef123...# Remaining hash chars
├── performances/      # Performance session metadata
├── index             # Fast commit lookup index
└── HEAD              # Current commit reference
```

## Development

### Building

```bash
# Development build
make build-dev

# Production build with optimizations
make build

# Cross-platform release builds
make release
```

### Testing

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run tests with race detection
make test-race

# Run benchmarks
make bench
```

### Code Quality

```bash
# Format code
make fmt

# Run static analysis
make vet

# Run linter (requires golint)
make lint

# Run all checks
make all
```

### Development Workflow

```bash
# Auto-rebuild on file changes (requires entr)
make dev

# Quick test for changed files
make test-quick
```

## Project Structure

```
livecodegit/
├── cmd/
│   └── lcg/          # CLI application
├── pkg/
│   ├── core/         # Core types and repository logic
│   └── storage/      # Storage layer and file system operations
├── .claude/          # Project documentation
├── Makefile          # Build automation
└── README.md         # This file
```

## Roadmap

### Phase 1: Core Data Structures and Storage ✅
- [x] Basic git-like functionality
- [x] SHA-1 hashing and object storage
- [x] Repository initialization and basic operations
- [x] CLI tool with init, commit, and log commands

### Phase 2: Execution Watchers (Planned)
- [ ] Sonic Pi OSC monitoring
- [ ] TidalCycles GHCi integration
- [ ] SuperCollider document watching
- [ ] Real-time commit creation

### Phase 3: Auto-commit System (Planned)
- [ ] Background commit service
- [ ] Performance session management
- [ ] Non-blocking operations

### Phase 4: Playback Engine (Planned)
- [ ] Time-accurate replay
- [ ] Variable speed playback
- [ ] Language-specific code injection

### Phase 5: Analysis and Visualization (Planned)
- [ ] Performance timeline visualization
- [ ] Code evolution analysis
- [ ] Export to various formats

### Phase 6: User Interface (Planned)
- [ ] Web-based performance browser
- [ ] Editor integrations (VS Code, Emacs, Vim)
- [ ] Live performance display

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature-name`
3. Make your changes and add tests
4. Run the test suite: `make test`
5. Commit your changes: `git commit -am 'Add feature'`
6. Push to the branch: `git push origin feature-name`
7. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments

- Inspired by Git's elegant approach to version control
- Built for the livecoding community
- Special thanks to the creators of Sonic Pi, TidalCycles, and SuperCollider# livecodegit
