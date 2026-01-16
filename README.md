# CodingGame

A Civilization/Call to Power II/Factorio-inspired strategy game interface for Claude Code, transforming software development into an intuitive visual experience.

> **Phase 1 (Core Framework) Implemented** ✅

## What is CodingGame?

CodingGame is a **game interface to coding**, not a game about coding. It uses game UI patterns and metaphors to visualize and interact with real software development activities:

- **Map View**: Your codebase as explorable territory
- **Fog of War**: Files Claude hasn't analyzed (real context window)
- **Resource Bar**: Real metrics (context tokens, API cost, coverage)
- **Buildings**: Build targets with actual build times
- **Units**: Tests that fight bugs (PvE metaphor)
- **Belts**: Dependency flow visualization (Factorio-style)
- **Advisors**: Real Claude subagents with focused contexts

**Core Principle**: Everything is real. No fake bonuses, no artificial progression, no made-up stats.

## Quick Start

### Recommended: Nix + Bazel

**Best for**: Reproducible builds, CI/CD, team development

```bash
# Clone the repository
git clone https://github.com/tedks/CodingGame.git
cd CodingGame

# Enter Nix environment
nix develop

# Build with Bazel
bazel build //...

# Run the game
bazel run //:codinggame
```

**With direnv**: Just `cd CodingGame && direnv allow` and the environment loads automatically!

### Alternative: Go Modules

**Best for**: Quick prototyping, if Nix is not available

```bash
# Prerequisites: Go 1.21+, X11 libs on Linux

# Clone and build
git clone https://github.com/tedks/CodingGame.git
cd CodingGame
GOPROXY=direct go mod download
go build -o codinggame .

# Run
./codinggame
```

See [BUILD.md](BUILD.md) for detailed instructions and system requirements.

### Controls (Phase 1)

- **hjkl** or **arrow keys**: Pan camera
- **+/-**: Zoom in/out
- **Tab**: Switch between views (future)
- **i**: Inspect tile (future)
- **g**: Go to file (future)

## Implementation Status

### ✅ Phase 1: Core Framework (Completed)

- [x] **Map Visualization Engine** - Zoomable, pannable 2D tile system
- [x] **Tile & Fog of War** - Files/directories with 3 fog states (Full, Stale, Revealed)
- [x] **Resource Tracking** - Top bar with real metrics (extensible)
- [x] **Claude Tool Interception** - Event-driven architecture for subprocess integration

### 🚧 Phase 0: Foundation (Partial)

- [x] Basic keyboard navigation (hjkl/arrows)
- [ ] Game start screen (model/harness selection)
- [ ] Prompt interaction window

### 📋 Upcoming Phases

- **Phase 2**: Buildings & Units (build systems, test visualization)
- **Phase 3**: Advisors (real subagents with specialized domains)
- **Phase 4**: Belts & Debugging (dataflow visualization)
- **Phase 5**: Capability Inventory (tech tree view)
- **Phase 6**: Production Realm (deployment visualization)
- **Phase 7**: Multi-Agent Future (parallel agents)
- **Phase 8**: Polish (themes, plugins, sound)

See [DESIGN.md](DESIGN.md) for complete roadmap.

## Architecture

**Tech Stack:**
- **Language**: Go (excellent concurrency, LLM-friendly)
- **UI Framework**: Ebitengine (lightweight, efficient, cross-platform)
- **Build System**: Bazel (hermetic builds, advanced caching)
- **Environment**: Nix (reproducible dependencies)
- **Issue Tracking**: Beads (AI-native, lives in repo)

**Key Components:**

```
┌─────────────────────────────────────────┐
│         Resource Bar (40px)              │
├─────────────────────────────────────────┤
│                                          │
│         Map View (tiles + fog)           │
│                                          │
│  ┌──┐ ┌──┐ ┌──┐    ░░░░ ░░░░            │
│  │██│ │██│ │░░│    ░░░░ ░░░░            │
│  └──┘ └──┘ └──┘    ░░░░ ░░░░            │
│   Revealed Files    Fogged Files        │
│                                          │
└─────────────────────────────────────────┘
```

**Event Flow:**

```
Claude subprocess (--output-format json)
        ↓
Claude Interceptor (parse events)
        ↓
Event Handlers (update state)
        ↓
Game Loop (Ebitengine 60fps)
        ↓
Render (tiles, fog, resources)
```

See [ARCHITECTURE.md](ARCHITECTURE.md) for technical details.

## Philosophy

### Everything Must Be Real

- **No fake bonuses**: Build time is actual build time
- **No fake stats**: Tests show real pass/fail/timing
- **No fake progression**: Tech tree shows tools you've configured
- **No fake resources**: Context tokens are actual token usage

### Descriptive, Not Prescriptive

The interface **describes** what exists, it doesn't **prescribe** progression:
- Tech tree shows configured tools, not unlockables
- Advisors are subagents you write, not pre-defined characters
- Buildings are real build targets, not RPG structures

See [PHILOSOPHY.md](PHILOSOPHY.md) for complete principles.

## Development

### Running Tests

**With Bazel (recommended):**

```bash
# All tests
bazel test //...

# Specific packages
bazel test //internal/tile:tile_test
bazel test //internal/claude:claude_test

# With verbose output
bazel test //... --test_output=all
```

**With Go modules:**

```bash
# All tests
go test ./...

# Specific packages
go test ./internal/tile -v
go test ./internal/claude -v

# With race detector
go test -race ./...
```

### Issue Tracking

This project uses [Beads](https://github.com/steveyegge/beads) for AI-native issue tracking:

```bash
# View available work
bd ready

# Show issue
bd show CodingGame-xxx

# Update status
bd update CodingGame-xxx --status in_progress

# Close issue
bd close CodingGame-xxx
```

Issues are stored in `.beads/issues.jsonl` and synced with git.

### Project Structure

```
CodingGame/
├── main.go                    # Entry point
├── internal/
│   ├── game/                  # Core game loop
│   ├── mapview/               # Map visualization engine
│   ├── tile/                  # Tile & fog of war system
│   ├── resources/             # Resource tracking
│   └── claude/                # Claude tool interception
├── DESIGN.md                  # Complete design spec
├── ARCHITECTURE.md            # Technical architecture
├── PHILOSOPHY.md              # Core principles
├── BUILD.md                   # Build instructions
└── .beads/                    # Issue tracking
```

## Contributing

See [AGENTS.md](AGENTS.md) for agent workflow and guidelines.

Key practices:
- Follow "everything must be real" principle
- Keyboard-first interface design
- Real metrics only, no fake gamification
- Go conventions (gofmt, explicit errors)
- Commit early and often

## Documentation

- [DESIGN.md](DESIGN.md) - Complete design specification
- [ARCHITECTURE.md](ARCHITECTURE.md) - Technical architecture
- [PHILOSOPHY.md](PHILOSOPHY.md) - Core design principles
- [BUILD.md](BUILD.md) - Build instructions and troubleshooting
- [AGENTS.md](AGENTS.md) - Agent workflow and guidelines

## License

See LICENSE file for details.

## Acknowledgments

Inspired by:
- Civilization series (Sid Meier)
- Call to Power II (Activision)
- Factorio (Wube Software)
- Claude Code (Anthropic)

---

*"The code is dark and full of terrors... but we shall bring the light of understanding."*
— The Chronicler
