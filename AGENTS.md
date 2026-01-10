# Agent Instructions

## Project Overview

**CodingGame** is a strategy game-style UI wrapper for Claude Code, inspired by Civilization/Factorio. It transforms software development into an intuitive strategy game experience.

**Core Principle:** This is a *game interface to coding*, NOT gamification. Everything is real:
- No fake bonuses or stats - metrics are actual build times, test results, coverage
- Descriptive, not prescriptive - shows what exists, not what you've "unlocked"
- Real subagents, real tools, real data flow

**Visual Metaphors:**
| Game Element | Real Thing |
|-------------|------------|
| Map/Tiles | Codebase (files & directories) |
| Buildings | Build targets (package.json, Cargo.toml) |
| Units | Tests ("fighting" to pass) |
| Advisors | Subagents you configure |
| Belts | Dependency/data flow (Factorio-style) |
| Fog of War | Context boundary |
| Tech Tree | Capability inventory (tools, MCPs) |

**Key Files:**
- `DESIGN.md` - Full design specification
- `PHILOSOPHY.md` - Core principles and metaphors

---

## Beads Issue Tracking

This project uses **bd** (beads) for issue tracking. Run `bd onboard` to get started.

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

```bash
bd ready              # Find available work
bd show <id>          # View issue details
bd update <id> --status in_progress  # Claim work
bd close <id>         # Complete work
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

