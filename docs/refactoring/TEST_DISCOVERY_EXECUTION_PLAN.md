# Test Discovery Execution Plan

## Current State

- **Refactoring Phases 1-5**: Complete (17 PRs merged)
- **PR #70**: Open - refactoring → master
- **Test Discovery Issues**: 21 GitHub issues created (#50-70)
- **Review Agents**: 21 Claude agents in tmux session `CodingGame`
- **Conflict Protocol**: docs/AGENT_CONFLICT_PROTOCOL.md

## Objective

Instruct all 21 review agents to implement the tests they discovered, using a structured approach that survives context compaction.

## Agent Instruction Protocol

### Step 1: Send Planning Mode Instruction

For each agent, send via claude-send.sh:

```bash
~/.claude/skills/spawn-agent/scripts/claude-send.sh "CodingGame:review-<module>" \
"PHASE 2 INSTRUCTIONS: Enter planning mode now. Type /plan to enter planning mode, then create a comprehensive implementation plan."
```

Wait for agent to enter planning mode (watch for plan mode indicator).

### Step 2: Send Detailed Plan Template

After agent is in planning mode, send:

```
Your plan must include these steps IN ORDER:

1. CREATE BEADS EPIC:
   bd create --type=epic --title="Test Discovery Implementation: <module>"
   Link to GitHub issue #<number> in description

2. CREATE CHILD TASKS in beads for each test category discovered

3. CREATE WORKTREE:
   cd /home/tedks/Projects/CodingGame.git
   git worktree add -b test-discovery/<module> ../test-<module> origin/refactoring

4. WRITE TESTS:
   - Write failing tests first (expose the bug/gap)
   - Verify test fails for the right reason
   - Fix any code issues exposed
   - Verify test passes

5. CONFLICT CHECK (if in conflict-prone group):
   See docs/AGENT_CONFLICT_PROTOCOL.md
   Related agents: <list from protocol>

6. GET CODEX REVIEW:
   /ask-agent codex "Review my test implementation in test-discovery/<module>. Check for: test quality, edge cases covered, code style."

7. CREATE PR:
   gh pr create --base refactoring --title "Test Discovery: <module> module"

8. FILE GITHUB ISSUES for anything you cannot solve autonomously

Write your plan now. Be detailed - this plan will survive context compaction.
```

### Step 3: Approve Plan

After verifying the plan is faithful to our vision:

```bash
~/.claude/skills/spawn-agent/scripts/claude-send.sh "CodingGame:review-<module>" "1"
```

This selects option 1 (proceed with plan).

## Conflict-Prone Groups

| Group | Agents | Shared Code |
|-------|--------|-------------|
| Lifecycle | review-harness, review-game, review-multiagent | Event handling, start/stop |
| Watchers | review-capability, review-production | File watching, registry |
| Rendering | review-ui, review-mapview, review-tile, review-resources | Drawing, layout |
| Testing | review-systemtest, review-testutil | Test infrastructure |
| Parsing | review-claude, review-build, review-dependency | Parsing patterns |

Standalone (no conflicts): advisor, belt, building, connection, debug, input, unit

## GitHub Issues Created

| Issue | Module | Key Findings |
|-------|--------|--------------|
| #50 | building | Metrics state machine |
| #51 | build | Adapter fuzz testing |
| #52 | claude | JSON parsing, stream interruption |
| #53 | dependency | Import extraction fuzzing |
| #54 | belt | Dependency visualization stress |
| #55 | advisor | Subagent pool property tests |
| #56 | debug | Debugger lifecycle, snapshots |
| #57 | connection | Graph algorithms, cycle detection |
| #58 | game | Scene transitions, loop determinism |
| #59 | production | File watcher accuracy |
| #60 | input | Vim mode transitions |
| #61 | unit | Coverage edge cases |
| #62 | resources | Metric edge cases |
| #63 | harness | Lifecycle state machine, backpressure |
| #64 | tile | Fog state machine |
| #65 | multiagent | **CRITICAL: TOCTOU race, snapshot consistency** |
| #66 | ui | Layout math edge cases |
| #67 | testutil | Meta-testing infrastructure |
| #68 | mapview | Coordinate math, zoom/pan |
| #69 | systemtest | Missing interactions |

## Critical Bugs Found (Priority)

1. **multiagent TOCTOU race** (#65) - HandoffTask() has check-then-act race
2. **multiagent Snapshots() consistency** (#65) - Agents queried at different times
3. **harness backpressure** (#63) - Buffer full behavior untested

## Execution Order

1. **First wave** (standalone, no conflicts):
   - advisor, belt, building, connection, debug, input, unit

2. **Second wave** (after first merges):
   - Lifecycle group: harness first, then game, then multiagent
   - Watcher group: capability, production (parallel OK)

3. **Third wave**:
   - Rendering group: ui first, then mapview, tile, resources
   - Testing group: testutil first, then systemtest
   - Parsing group: parallel OK

## Session Recovery

If context compacts, resume by:
1. Reading this document
2. Checking `tmux list-windows -t CodingGame` for agent status
3. Checking `bd list --all` for beads progress
4. Checking `gh pr list` for created PRs
5. Continuing from where agents left off
