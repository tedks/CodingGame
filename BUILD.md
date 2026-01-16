# CodingGame Build Instructions

## Overview

CodingGame uses **Nix** for reproducible development environments and **Bazel** for hermetic, scalable builds. Go modules are also supported for simpler workflows.

**Recommended:** Use Nix + Bazel for the best development experience.

---

## Quick Start (Nix + Bazel)

### Automated Setup (Recommended)

The fastest way to get started:

```bash
# Clone the repository
git clone https://github.com/tedks/CodingGame.git
cd CodingGame

# Run the automated setup script
./scripts/setup-dev-env.sh

# The script will install Nix, direnv, and verify everything works
# Then you can immediately build and run:
bazel build //...
bazel run //:codinggame
```

### Manual Setup

If you prefer to install manually or need more control:

**Prerequisites:**
- [Nix](https://nixos.org/download.html) with flakes enabled
- [direnv](https://direnv.net/) (optional but recommended)

**Steps:**

```bash
# Clone the repository
git clone https://github.com/tedks/CodingGame.git
cd CodingGame

# Option 1: Using direnv (automatic)
direnv allow

# Option 2: Manual Nix shell
nix develop

# Build with Bazel
bazel build //...

# Run the game
bazel run //:codinggame

# Run tests
bazel test //...
```

That's it! Nix handles all dependencies (Go, Bazel, X11 libraries, etc.).

---

## Detailed Setup

### 1. Nix Installation

**Tip:** For automated installation, run `./scripts/setup-dev-env.sh` instead of following these manual steps.

#### Install Nix with Flakes

```bash
# Install Nix (if not already installed)
curl --proto '=https' --tlsv1.2 -sSf -L https://install.determinate.systems/nix | sh -s -- install

# Or use the official installer and enable flakes:
sh <(curl -L https://nixos.org/nix/install) --daemon

# Enable flakes (if using official installer)
mkdir -p ~/.config/nix
echo "experimental-features = nix-command flakes" >> ~/.config/nix/nix.conf
```

#### Verify Installation

```bash
nix --version
# Should show: nix (Nix) 2.x.x or later
```

### 2. Enter Development Environment

#### Option A: direnv (Recommended)

direnv automatically loads the Nix environment when you `cd` into the project directory.

```bash
# Install direnv
# macOS:
brew install direnv

# Ubuntu/Debian:
apt-get install direnv

# Add to your shell rc file (~/.bashrc, ~/.zshrc, etc.):
eval "$(direnv hook bash)"  # or 'zsh', 'fish', etc.

# Allow direnv for this project
cd CodingGame
direnv allow
```

Now the environment loads automatically when you enter the directory!

#### Option B: Manual nix develop

```bash
cd CodingGame
nix develop

# You'll see:
# 🎮 CodingGame Development Environment
# ======================================
# Available commands:
#   bazel build //...        - Build all targets
#   bazel test //...         - Run all tests
#   bazel run //:codinggame  - Run the game
# ...
```

### 3. Verify Environment

```bash
# Check tools are available
go version      # Should show: go1.22.x
bazel version   # Should show: bazel 7.0.0
```

---

## Building with Bazel

### Build All Targets

```bash
bazel build //...
```

### Build Specific Targets

```bash
# Build main binary
bazel build //:codinggame

# Build a specific package
bazel build //internal/tile
bazel build //internal/claude
```

### Run the Game

```bash
# Run in current directory
bazel run //:codinggame

# Run on a specific project
bazel run //:codinggame -- /path/to/project
```

### Run Tests

```bash
# Run all tests
bazel test //...

# Run specific package tests
bazel test //internal/tile:tile_test
bazel test //internal/claude:claude_test

# Run with verbose output
bazel test //... --test_output=all

# Run with race detector
bazel test //... --features=race
```

### Clean Build Artifacts

```bash
# Clean Bazel cache
bazel clean

# Clean everything including external dependencies
bazel clean --expunge
```

### Build Modes

```bash
# Optimized release build
bazel build --config=release //:codinggame

# Debug build with symbols
bazel build --config=debug //:codinggame
```

---

## Building with Go Modules (Alternative)

If you prefer not to use Nix/Bazel, you can use standard Go tooling.

### Prerequisites

- Go 1.21 or later
- System dependencies (see below)

### System Dependencies

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
brew install go
```

#### Windows

```bash
# No additional dependencies required
# Install Go from: https://go.dev/dl/
```

### Build

```bash
# Install dependencies
GOPROXY=direct go mod download

# Build the binary
go build -o codinggame .

# Run
./codinggame
```

### Test

```bash
# Run all tests
go test ./...

# Run specific package tests
go test ./internal/tile -v
go test ./internal/claude -v

# Run with race detector
go test -race ./...
```

---

## Development Workflow

### With Nix + Bazel (Recommended)

```bash
# Enter project directory (with direnv)
cd CodingGame

# Make code changes
vim internal/game/game.go

# Build and test
bazel build //...
bazel test //...

# Run
bazel run //:codinggame

# Format Bazel files
buildifier -r .

# Update Go dependencies (if go.mod changes)
bazel run //:gazelle-update-repos

# Update BUILD.bazel files (if new .go files added)
bazel run //:gazelle
```

### With Go Modules

```bash
# Make code changes
vim internal/game/game.go

# Build and test
go build
go test ./...

# Run
./codinggame

# Format code
go fmt ./...
```

---

## Bazel Configuration

### .bazelrc

The `.bazelrc` file contains build configurations:

- Platform-specific settings (Linux, macOS, Windows)
- Compilation modes (release, debug)
- Test output settings
- Nix-specific environment variables

### Custom Configuration

Create `.bazelrc.user` for personal settings (gitignored):

```bash
# Example .bazelrc.user
build --jobs=8
test --test_output=streamed
```

---

## Troubleshooting

### "command not found: bazel"

**Solution**: You're not in the Nix environment.

```bash
# With direnv:
direnv allow

# Without direnv:
nix develop
```

### "cannot find package" errors

**Solution**: Update Bazel's Go dependencies.

```bash
# After modifying go.mod:
bazel run //:gazelle-update-repos
bazel run //:gazelle
```

### X11 build errors on Linux

**Solution**: Nix automatically provides X11 libraries. If using Go modules:

```bash
sudo apt-get install libgl1-mesa-dev libxrandr-dev libxcursor-dev \
    libxinerama-dev libxi-dev libxxf86vm-dev libasound2-dev
```

### "experimental-features" error

**Solution**: Enable Nix flakes.

```bash
mkdir -p ~/.config/nix
echo "experimental-features = nix-command flakes" >> ~/.config/nix/nix.conf
```

### Bazel builds fail in Nix environment

**Solution**: Make sure you're using the `nix` config.

```bash
bazel build --config=nix //...
```

Or add to `.bazelrc.user`:

```bash
build --config=nix
```

### Slow initial Bazel build

This is normal. Bazel downloads and caches dependencies on first build. Subsequent builds are much faster thanks to caching.

---

## CI/CD

### GitHub Actions

Example workflow for Nix + Bazel:

```yaml
name: Build and Test

on: [push, pull_request]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: DeterminateSystems/nix-installer-action@main
      - uses: DeterminateSystems/magic-nix-cache-action@main

      - name: Build
        run: nix develop --command bazel build //...

      - name: Test
        run: nix develop --command bazel test //...
```

---

## Build System Comparison

| Feature | Nix + Bazel | Go Modules |
|---------|-------------|------------|
| **Reproducibility** | ✅ Hermetic | ⚠️ Depends on system |
| **Dependency Management** | ✅ Automatic | ⚠️ Manual install |
| **Caching** | ✅ Advanced | ✅ Basic |
| **Incremental Builds** | ✅ File-level | ✅ Package-level |
| **Cross-platform** | ✅ Consistent | ⚠️ Varies |
| **Setup Complexity** | ⚠️ Initial setup | ✅ Simple |
| **Build Speed** | ✅ Fast (cached) | ✅ Fast |
| **IDE Support** | ⚠️ Requires plugin | ✅ Native |

**Recommendation**: Use Nix + Bazel for development and CI/CD. Use Go modules for quick prototyping or if Nix is not available.

---

## Editor/IDE Setup

### VSCode

Install extensions:
- **Go** (golang.go)
- **Bazel** (BazelBuild.vscode-bazel)
- **Nix** (bbenoist.Nix)

With direnv, VSCode automatically uses the Nix environment.

### GoLand / IntelliJ IDEA

1. Install **Bazel plugin**
2. Import project: File → Open → select CodingGame directory
3. Select "Bazel" as project type
4. Configure to use Nix environment (use terminal or direnv integration)

### Vim/Neovim

```vim
" Add to your config
Plug 'bazelbuild/vim-bazel'
Plug 'fatih/vim-go'
Plug 'LnL7/vim-nix'
```

LSP works automatically with `gopls` from Nix environment.

---

## Performance Tips

### Bazel

- Use `--jobs=auto` (default in `.bazelrc`)
- Enable disk cache: Add to `.bazelrc.user`:
  ```
  build --disk_cache=~/.cache/bazel
  ```
- Use remote cache for teams (advanced)

### Nix

- Use binary caches (enabled by default)
- Keep Nix store clean: `nix-collect-garbage -d`
- Use `nix-direnv` for faster shell activation

---

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

---

## Next Steps

Phase 2 (Buildings & Units) will add:
- Build system adapters (npm, **Bazel**, cargo)
- Real build metrics display
- Test visualization with actual results

See [DESIGN.md](DESIGN.md) for complete roadmap.

---

## References

- [Nix Manual](https://nixos.org/manual/nix/stable/)
- [Bazel Documentation](https://bazel.build/docs)
- [rules_go](https://github.com/bazelbuild/rules_go)
- [direnv](https://direnv.net/)
- [Go Documentation](https://go.dev/doc/)
