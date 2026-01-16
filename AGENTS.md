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

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   bd sync
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds

## Building and Testing

This project uses **Bazel** as the primary build system with **Nix** for reproducible environments.

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

**ALWAYS use Bazel for running tests:**

```bash
# Run all tests
bazel test //...

# Run specific package tests
bazel test //internal/tile:tile_test
bazel test //internal/claude:claude_test
bazel test //internal/mapview:mapview_test
bazel test //internal/game:game_test

# Run with verbose output
bazel test //... --test_output=all

# Run tests showing only errors
bazel test //... --test_output=errors
```

### Quality Gates (Pre-Commit)

Before committing code changes, run:

```bash
# Build everything
bazel build //...

# Run all tests
bazel test //...

# Format Go code
go fmt ./...
```

### Nix Environment

The Nix environment provides all dependencies automatically:

```bash
# Enter Nix shell (if not using direnv)
nix develop

# Or with direnv
direnv allow  # Auto-loads environment
```

**Note**: In environments without Nix installed, Bazel will not be available. Use Go modules instead (see below).

### Alternative: Go Modules

For quick iteration or environments without Nix, Go modules are also supported:

```bash
go build
go test ./...
```

However, **prefer Bazel for CI/CD and final validation** as it provides hermetic builds.

In Nix-enabled environments, always use `bazel test //...` instead of `go test`.

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
- Unit tests for game logic and state management
- Integration tests for Claude subprocess interaction
- Manual testing for visual elements and UX flows
- No fake/mock metrics in tests - use real examples

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
