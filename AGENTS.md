# Agent Instructions

> For comprehensive Claude Code agent guidance including project context, code style, and review guidelines, see [CLAUDE.md](CLAUDE.md).

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
