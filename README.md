# LiveCodeGit

A Git-like version control system designed specifically for capturing and replaying livecoding performances in languages like Sonic Pi, TidalCycles, and SuperCollider.

## Features

- **Automatic Commit Capture**: Seamlessly records code states at each execution
- **Performance Metadata**: Tracks timing, BPM, musical context, and execution success
- **Git-like Storage**: Content-addressable storage with SHA-1 hashing
- **Performance Replay**: Time-accurate playback of coding sessions
- **Multiple Language Support**: Designed for Sonic Pi, TidalCycles, SuperCollider, and more

## Build System

LiveCodeGit uses a comprehensive Makefile for building, testing, and development. All commands should be run from the project root directory.

### Quick Start

```bash
# Build everything (format, vet, test, build)
make all

# Build the binary only
make build

# Run tests
make test
```

### Development Commands

```bash
# Install dependencies
make deps

# Format code using gofmt
make fmt

# Run static analysis with go vet
make vet

# Run linter (requires golint: go install golang.org/x/lint/golint@latest)
make lint

# Development mode - auto-rebuild on file changes (requires entr)
make dev

# Build for development (faster, no optimization)
make build-dev
```

### Testing Commands

```bash
# Run all tests with verbose output
make test

# Run tests with coverage report (generates coverage.html)
make test-coverage

# Run tests with race condition detection
make test-race

# Run quick tests (short mode)
make test-quick

# Run benchmarks
make bench

# Run integration tests (builds binary and tests basic functionality)
make test-integration
```

### Installation and Deployment

```bash
# Install binary to $GOPATH/bin
make install

# Build cross-platform release binaries
make release

# Docker build (if Dockerfile exists)
make docker-build
```

### Maintenance Commands

```bash
# Clean all build artifacts
make clean

# Show all available commands
make help
```

### Binary Usage

After building, the `lcg` binary will be in the `build/` directory:

```bash
# Initialize a new LiveCodeGit repository
./build/lcg init

# Commit current workspace state
./build/lcg commit "Added new beat pattern"

# View commit history
./build/lcg log

# Start execution monitoring for Sonic Pi
./build/lcg watch --lang sonicpi

# List available watchers
./build/lcg watch --list

# Show watcher service status
./build/lcg watch --status
```
