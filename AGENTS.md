# Agent Instructions

This document provides comprehensive guidance for Claude Code agents working on the CodingGame project, including workflow management, issue tracking, and session completion requirements.

## Project Overview

CodingGame is a strategy game interface for Claude Code, inspired by Civilization/Call to Power II/Factorio. It transforms software development into an intuitive visual experience where the codebase becomes a strategic map.

**Core Philosophy**: This is a game interface to coding, not a game about coding. Everything shown is real - no fake bonuses, stats, or artificial progression.

**Implementation**: Go + Ebitengine GUI application. See [ARCHITECTURE.md](ARCHITECTURE.md) for technical details and [DESIGN.md](DESIGN.md) for complete specifications.

## Beads Issue Tracking

This project uses **bd** (beads) for issue tracking. Run `bd ready` to get started.

## First-Time Setup

If beads is not yet installed on your machine, run the setup script:

```bash
./scripts/setup-beads.sh
```

This script will:
1. Install the `bd` CLI if not already installed
2. Initialize beads in the repository (if needed)
3. Install git hooks for automatic sync
4. Set up Claude Code integration (if detected)

The script is idempotent - safe to run multiple times.

## Web Claude Code (No Daemon Mode)

When running in web-based Claude Code environments, the beads daemon may have database locking issues. Set the `BEADS_NO_DAEMON` environment variable:

```bash
export BEADS_NO_DAEMON=1
bd ready
```

Or prefix individual commands:

```bash
BEADS_NO_DAEMON=1 bd create --title="New task" --type=task
```

## Quick Reference

Issues in this project use the prefix `CodingGame-XXX` (e.g., `CodingGame-w93`, `CodingGame-66a`).

```bash
bd ready              # Find available work
bd show CodingGame-abc  # View issue details (use actual issue ID)
bd update CodingGame-abc --status in_progress  # Claim work
bd close CodingGame-abc  # Complete work
bd sync               # Sync with git
```

## Landing the Plane (Session Completion)

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **Verify development environment** - Ensure Nix + Bazel are working:
   ```bash
   ./scripts/setup.sh --verify
   # OR manually:
   which bazel && which go  # Must return paths
   ```

2. **File issues for remaining work** - Create issues for anything that needs follow-up

3. **Run quality gates** (if code changed) - Tests, linters, builds:
   ```bash
   # Ensure you're in Nix environment
   which bazel || nix develop

   # Run quality checks
   bazel build //...
   bazel test //...
   go fmt ./...
   ```

4. **Update issue status** - Close finished work, update in-progress items

5. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   bd sync  # If beads is installed
   git push
   git status  # MUST show "up to date with origin"
   ```

6. **Clean up** - Clear stashes, prune remote branches

7. **Verify** - All changes committed AND pushed

8. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds
- ALWAYS verify Nix/Bazel environment before running quality gates

## Building and Testing

This project uses **Nix** for environmental dependencies and **Bazel** for code dependencies and builds.

### Dependency Management Philosophy

- **Nix** manages environmental dependencies: Go toolchain, Bazel, system libraries (X11, etc.)
- **Bazel** manages code dependencies: Go packages, external libraries
- **Result**: Hermetic, reproducible builds across all environments

### First-Time Setup: Complete Development Environment

**If you don't have Nix, Bazel, and beads installed yet**, run the automated setup script:

```bash
./scripts/setup.sh
```

This single script installs everything you need:
1. Nix package manager with flakes support
2. direnv for automatic environment loading (optional)
3. Bazel and Go (via Nix environment)
4. beads issue tracking system
5. All necessary git hooks and configurations

**Partial setup options:**
```bash
./scripts/setup.sh --verify        # Check existing setup
./scripts/setup.sh --skip-direnv   # Skip direnv installation
./scripts/setup.sh --skip-beads    # Skip beads setup
./scripts/setup.sh --beads-only    # Only install beads
```

**Manual installation**: See [BUILD.md](BUILD.md) for detailed step-by-step instructions.

### Environment Setup

**CRITICAL**: You must ALWAYS be in a Nix environment when working on this project.

**Check if in Nix environment:**
```bash
which bazel  # Should return a path (not "command not found")
```

**If Bazel is not found, enter Nix environment:**
```bash
# Method 1: direnv (automatic, recommended)
direnv allow

# Method 2: Manual nix develop
nix develop

# Then verify:
which bazel  # Should now work
```

**Note**: All build and test commands below assume you're in the Nix environment.

### Build Commands

```bash
# Build all targets
bazel build //...

# Build main binary
bazel build //:codinggame

# Build specific package
bazel build //internal/tile
```

### Running Tests

**CRITICAL: ONLY run tests through Nix + Bazel. NEVER use `go test` directly.**

Using `go test` bypasses the hermetic build environment and may produce inconsistent results. The Nix environment ensures all system dependencies (X11, OpenGL, etc.) are correctly configured.

**Recommended approach:** Enter the Nix environment once, then use bazel directly:

```bash
# Enter Nix environment (do this once per session)
nix develop

# Then use bazel directly (saves tokens vs. nix develop --command each time)
bazel test //...
bazel test //internal/tile:tile_test
bazel test //... --test_output=all
```

**Alternative (one-off commands):**
```bash
nix develop --command bazel test //...
```

**WRONG - DO NOT DO THIS:**
```bash
go test ./...           # Bypasses Nix environment
go test ./internal/...  # May fail due to missing system deps
```

**Why this matters:**
- Ebitengine requires X11/OpenGL libraries that Nix provides
- Some tests need `xvfb-run` for headless display (Bazel handles this)
- `go test` may pass locally but fail in CI due to environment differences

### Quality Gates (Pre-Commit)

Before committing code changes, run inside Nix environment:

```bash
# Enter Nix environment (if not already)
nix develop

# Build everything
bazel build //...

# Run all tests
bazel test //...

# Format Go code
go fmt ./...
```

### Why Not Go Modules?

While Go modules work, they bypass our hermetic build system:
- ❌ **Go modules**: System-dependent, version drift, "works on my machine"
- ✅ **Nix + Bazel**: Exact same environment everywhere, reproducible builds

**Always use Bazel** for builds and tests. Go modules (`go.mod`, `go.sum`) exist only for IDE support and dependency tracking.

## Code Style and Development Guidelines

### Go Conventions
- Follow standard Go formatting (`gofmt`)
- Use meaningful variable names (Go favors clarity over brevity)
- Document exported functions and types
- Handle errors explicitly, don't ignore them
- Keep goroutines and channels simple and obvious

### Game Interface Guidelines
- **Keyboard accessibility**: Prioritize vim-style navigation, no mouse required
- **Real data only**: Every visual element must represent something real
- **Performance**: Target low-end hardware, optimize rendering
- **Graceful degradation**: Fail gracefully, never crash on bad data

### Visual Metaphors

Understanding the metaphors helps maintain consistency:

| Metaphor | Represents | Key Point |
|----------|-----------|-----------|
| Map | Codebase structure | Files/directories as territory |
| Fog of War | Context window | What Claude has/hasn't read |
| Buildings | Build targets | Real build metrics only |
| Units | Tests | Real pass/fail, timing, coverage |
| Advisors | Subagents | Real Claude instances with focused contexts |
| Belts | Dependencies | Data flow visualization + debugging |
| Tech Tree | Capabilities | Shows configured tools/MCPs |

### Testing Strategy
- **Always run tests with Bazel**: `bazel test //...` (never `go test`)
- Unit tests for game logic and state management
- Integration tests for Claude subprocess interaction
- Manual testing for visual elements and UX flows
- No fake/mock metrics in tests - use real examples
- Tests must pass in Nix environment before committing

## PR Review Guidelines

When reviewing pull requests, check for:

- **No fake gamification**: Ensure all displayed metrics are real
- **Keyboard accessibility**: Can everything be done without a mouse?
- **Performance**: Will this run on older hardware?
- **Error handling**: Does it fail gracefully?
- **Real data**: Are we showing actual metrics, not placeholders?
- **Go idioms**: Does the Go code follow standard conventions?
- **Documentation**: Are new features explained?

## Related Documentation

- [DESIGN.md](DESIGN.md) - Complete design specification
- [PHILOSOPHY.md](PHILOSOPHY.md) - Core design principles  
- [ARCHITECTURE.md](ARCHITECTURE.md) - Technical architecture
- [.beads/README.md](.beads/README.md) - Beads issue tracking guide

## Questions or Clarifications?

When uncertain:
1. Check if it violates "everything must be real"
2. Ask: "Does this describe what exists or prescribe a progression?"
3. Verify it matches the stated metaphor
4. Look for similar patterns in existing design docs

Remember: This is a visualization tool for real development activities, not a traditional game with artificial progression systems.
