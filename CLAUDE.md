# Claude Agent Instructions for CodingGame

This document provides guidance for Claude Code agents working on the CodingGame project.

## Project Overview

CodingGame is a strategy game interface for Claude Code, inspired by Civilization/Call to Power II/Factorio. It transforms software development into an intuitive visual experience where the codebase becomes a strategic map.

**Core Philosophy**: This is a game interface to coding, not a game about coding. Everything shown is real - no fake bonuses, stats, or artificial progression.

## Key Principles

### 1. Everything Must Be Real
- Build times are actual build times from the build system
- Test metrics show real pass rates, execution times, and coverage
- Resource tracking displays actual context tokens, API costs, and CI status
- No fake RPG elements (HP, ATK, levels, etc.)

### 2. Descriptive, Not Prescriptive
- The interface describes what exists in the project
- Tech trees show configured tools and MCPs, not unlockable upgrades
- Advisors are real subagents you configure, not pre-defined characters

### 3. Implementation Context
- **Language**: Go
- **GUI Framework**: Ebitengine (2D game engine)
- **Integration**: Wraps Claude Code via `claude --output-format json`
- **Platform**: Linux first, cross-platform later

## Architecture

See [ARCHITECTURE.md](ARCHITECTURE.md) for detailed technical specifications.

Key points:
- Goroutines handle subprocess streaming without blocking the game loop
- JSON streaming provides structured access to tool calls and responses
- Event-driven architecture updates visualizations in real-time

## Development Workflow

### Working with Beads
This project uses [Beads](https://github.com/steveyegge/beads) for issue tracking. Issues live in `.beads/issues.jsonl` and use the prefix `CodingGame-XXX`.

**Common commands**:
```bash
bd ready              # Find available work
bd show CodingGame-abc  # View issue details (use actual issue ID)
bd update CodingGame-abc --status in_progress  # Claim work
bd close CodingGame-abc  # Complete work
```

See [AGENTS.md](AGENTS.md) for detailed agent workflow instructions.

### Code Style

**Go Conventions**:
- Follow standard Go formatting (`gofmt`)
- Use meaningful variable names (Go favors clarity over brevity)
- Document exported functions and types
- Handle errors explicitly, don't ignore them
- Keep goroutines and channels simple and obvious

**Game Interface Guidelines**:
- Prioritize keyboard accessibility (vim-style navigation)
- Every visual element must represent something real
- Performance matters - target low-end hardware
- Fail gracefully, never crash on bad data

### Testing Strategy

- Unit tests for game logic and state management
- Integration tests for Claude subprocess interaction
- Manual testing for visual elements and UX flows
- No fake/mock metrics in tests - use real examples

## Visual Metaphors

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

## Common Tasks

### Adding New Visualizations
1. Define what real data it represents
2. Implement data source integration (build system, test runner, etc.)
3. Create rendering logic in Ebitengine
4. Wire up to Claude JSON stream events
5. Add keyboard controls

### Integrating Build Systems
- Create adapter in appropriate module
- Extract real metrics (timing, success rate, dependencies)
- Map to building visualization
- Avoid inventing fake "building levels" or bonuses

### Working with Claude Streaming
- Parse `claude --output-format json` output
- Handle `tool_use` events to trigger visualizations
- Use goroutines for non-blocking reads
- Gracefully handle malformed JSON or subprocess crashes

## Review Guidelines

When reviewing PRs, check for:

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
- [AGENTS.md](AGENTS.md) - Agent workflow and session management
- [.beads/README.md](.beads/README.md) - Beads issue tracking guide

## Questions or Clarifications?

When uncertain:
1. Check if it violates "everything must be real"
2. Ask: "Does this describe what exists or prescribe a progression?"
3. Verify it matches the stated metaphor
4. Look for similar patterns in existing design docs

Remember: This is a visualization tool for real development activities, not a traditional game with artificial progression systems.
