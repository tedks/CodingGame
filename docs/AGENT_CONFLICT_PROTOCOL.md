# Agent Conflict Resolution Protocol

This document defines the protocol for multiple agents working on related code simultaneously.

## Overview

When multiple agents work on test discovery implementations, they may modify overlapping files. This protocol ensures conflicts are detected early and resolved efficiently.

## Conflict-Prone Module Groups

| Group | Modules | Shared Code Areas |
|-------|---------|-------------------|
| **Lifecycle** | harness, game, multiagent | Event handling, start/stop semantics |
| **Watchers** | capability, production | File watching, registry patterns |
| **Rendering** | ui, mapview, tile, resources | Drawing, layout, visual components |
| **Testing** | systemtest, testutil | Test infrastructure, input simulation |
| **Parsing** | claude, build, dependency | JSON/file parsing patterns |

## Protocol Steps

### 1. Proactive Merge Testing

Before making significant changes, and periodically during work:

```bash
# Fetch latest from all branches
git fetch origin

# Check for conflicts with refactoring base
git merge-tree $(git merge-base HEAD origin/refactoring) HEAD origin/refactoring

# If working on related modules, check their branches too
git fetch origin test-discovery/<related-module>
git merge origin/test-discovery/<related-module> --no-commit --no-ff
git merge --abort  # If just testing
```

### 2. Conflict Detection

When you detect a merge conflict:

1. **Identify the conflicting agent** - Check which module owns the conflicting code
2. **Assess complexity**:
   - Simple: Your changes can be rebased/adjusted
   - Complex: Requires coordination

### 3. Resolution Strategies

#### Strategy A: Fix On Your Side (Preferred)

If your changes can be adjusted without breaking your tests:

```bash
# Merge their changes first
git fetch origin test-discovery/<other-module>
git merge origin/test-discovery/<other-module>
# Resolve conflicts favoring their changes where possible
# Re-run your tests to verify
```

#### Strategy B: Fix Common Code + Notify

If you must modify shared code:

1. **Fix the conflict** in a way that works for both use cases
2. **Notify the other agent** via tmux:

```bash
# Send message to other agent's tmux window
~/.claude/skills/spawn-agent/scripts/claude-send.sh "CodingGame:review-<module>" \
  "CONFLICT NOTIFICATION: I fixed a merge conflict in <file>. Please run: git fetch origin && git merge origin/test-discovery/<your-module> --no-edit"
```

3. **Wait for acknowledgment** before proceeding with PR

#### Strategy C: Sequential Execution

For deeply intertwined changes:

1. One agent completes and merges first
2. Other agent rebases on merged changes
3. Coordinate via tmux messages

### 4. Communication Format

When sending conflict notifications:

```
CONFLICT NOTIFICATION:
- File(s): <list of files>
- Your branch: test-discovery/<your-module>
- My branch: test-discovery/<my-module>
- Resolution: <what you did>
- Action needed: <what they should do>
```

### 5. Beads Task Annotation

For tasks that might conflict, add to the beads description:

```
CONFLICT RISK: This task modifies shared code in <package>.
Related agents: review-<module1>, review-<module2>
Protocol: See docs/AGENT_CONFLICT_PROTOCOL.md
```

## Tmux Window Reference

All review agents are in the `CodingGame` tmux session:

| Window | Module |
|--------|--------|
| review-advisor | advisor |
| review-belt | belt |
| review-build | build |
| review-building | building |
| review-capability | capability |
| review-claude | claude |
| review-connection | connection |
| review-debug | debug |
| review-dependency | dependency |
| review-game | game |
| review-harness | harness |
| review-input | input |
| review-mapview | mapview |
| review-multiagent | multiagent |
| review-production | production |
| review-resources | resources |
| review-systemtest | systemtest |
| review-testutil | testutil |
| review-tile | tile |
| review-ui | ui |
| review-unit | unit |

## Example Workflow

1. Agent A (harness) starts implementing lifecycle tests
2. Agent B (game) starts implementing scene transition tests
3. Agent B runs proactive merge check, sees Agent A touched `harness.go`
4. Agent B adjusts their test to work with Agent A's changes
5. Agent B sends: `"FYI: I adapted my game tests to work with your harness changes"`
6. Both agents continue independently
