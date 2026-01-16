# CodingGame Build Instructions

## Overview

This document describes how to build and run CodingGame, including system requirements and troubleshooting.

## System Requirements

### Required Dependencies

#### Linux (Ubuntu/Debian)
```bash
sudo apt-get install -y \
    libgl1-mesa-dev \
    libxrandr-dev \
    libxcursor-dev \
    libxinerama-dev \
    libxi-dev \
    libxxf86vm-dev \
    libasound2-dev \
    pkg-config
```

#### macOS
```bash
# No additional dependencies required
```

#### Windows
```bash
# No additional dependencies required
```

### Go Version

Go 1.21 or later is required.

## Building

### Standard Build

```bash
# Install dependencies
GOPROXY=direct go mod download

# Build the binary
go build -o codinggame .
```

### Running

```bash
# Run in current directory
./codinggame

# Run on a specific project
./codinggame /path/to/project
```

## Development

### Running Tests

```bash
# Run all tests
go test ./...

# Run specific package tests
go test ./internal/tile -v
go test ./internal/claude -v
go test ./internal/resources -v

# Run with race detector
go test -race ./...
```

### Building in Headless Environment

If you're in a headless environment without X11, you can still run the unit tests for non-UI components:

```bash
# Test individual packages that don't require UI
go test ./internal/tile -v
go test ./internal/claude -v
```

The main game and resource visualization require X11 to build/run.

## Architecture

See [ARCHITECTURE.md](ARCHITECTURE.md) for detailed technical documentation.

## Phase 1 Implementation Status

Phase 1 (Core Framework) has been implemented:

✅ **Map Visualization Engine** (CodingGame-2f5)
- Zoomable, pannable 2D tile system
- 5 zoom levels: World → Region → City → Street → Interior
- Grid-based layout with camera controls
- Keyboard navigation (hjkl/arrows, +/- for zoom)

✅ **Tile and Fog of War System** (CodingGame-m25)
- Tile abstraction for files/directories
- Fog states: Full, Stale, Revealed
- Reveal tracking with timestamps and counts
- Highlight support for file edits
- Thread-safe concurrent access

✅ **Resource Tracking** (CodingGame-17h)
- Top bar with real metrics
- Default resources: Context tokens, API cost, Test coverage
- Extensible system for custom resources
- Visual meter display with progress bars

✅ **Claude Tool Interception** (CodingGame-5b6)
- Event-driven architecture for Claude subprocess
- Event types: FileRead, FileWrite, FileEdit, BuildRun, TestRun, SubagentRun
- Handler registration system
- Automatic fog reveal on file reads
- Tile highlighting on file edits

## Testing

Comprehensive test coverage for Phase 1:

- `internal/tile/tile_test.go` - 7 tests, all passing
- `internal/claude/interceptor_test.go` - 8 tests, all passing
- `internal/resources/resources_test.go` - 6 tests (requires UI build)

## Known Issues

1. **X11 Dependency**: Building on Linux requires X11 development libraries
2. **Headless Testing**: UI components cannot be tested in headless environments
3. **Network Proxy**: Use `GOPROXY=direct` if standard proxy has issues

## Next Steps

Phase 2 (Buildings & Units) will add:
- Build system adapters (npm, Bazel, cargo)
- Real build metrics display
- Test visualization with actual results

See [DESIGN.md](DESIGN.md) for complete roadmap.
