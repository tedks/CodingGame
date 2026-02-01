# Test Discovery Implementation Plan

## Status
- **Refactoring PR**: #70 (refactoring → master)
- **Test Discovery Issues**: GitHub #50-#69 (21 issues)
- **Review Agents**: 21 Claude agents in tmux session `CodingGame`

## Phase: Instruct Agents to Implement Test Discoveries

### Instructions to Send to Each Agent

Each of the 21 review agents (in `CodingGame:review-<module>`) needs these instructions:

```
IMPLEMENTATION PHASE - Follow this plan exactly:

1. CREATE BEADS EPIC
   cd /home/tedks/Projects/CodingGame-orchestrator
   BEADS_NO_DAEMON=1 bd create --type=epic --title="Test Discovery: <module> implementation"

   Create child tasks for each test category in your GitHub issue.
   Link to GitHub issue #<number> in descriptions.
   Make descriptions DETAILED - they persist across context compaction.

2. CREATE WORKTREE
   cd /home/tedks/Projects/CodingGame.git
   git fetch origin
   git worktree add -b test-discovery/<module> ../test-<module> origin/refactoring

3. IMPLEMENT TESTS
   For each test category:
   a. Write the test (it may fail initially - that's expected)
   b. Run: xvfb-run -a -s "-screen 0 1024x768x24 -ac" nix develop --command bazel test //internal/<module>:...
   c. If test exposes a bug, FIX THE BUG
   d. Verify test passes
   e. Update beads task status

4. CHECK FOR CONFLICTS (if in conflict-prone group)
   See docs/AGENT_CONFLICT_PROTOCOL.md
   Related agents: <list based on group>
   Proactively: git fetch origin && git merge-tree ...

5. GET CODEX REVIEW
   /ask-agent codex "Review my test implementation in test-discovery/<module>.
   Focus on: test coverage, edge cases, assertions quality.
   Files: <list key files>"

6. CREATE PR
   git push -u origin test-discovery/<module>
   gh pr create --base refactoring --title "Test Discovery: <module> tests"

7. FILE ISSUES FOR BLOCKERS
   If you cannot solve something autonomously:
   gh issue create --title "Blocker: <description>" --body "..."
```

### Conflict-Prone Groups

| Group | Modules | Notify Each Other |
|-------|---------|-------------------|
| Lifecycle | harness, game, multiagent | All three |
| Watchers | capability, production | Both |
| Rendering | ui, mapview, tile, resources | All four |
| Testing | systemtest, testutil | Both |
| Parsing | claude, build, dependency | All three |

### GitHub Issues Created (Reference)

| # | Module | Key Findings |
|---|--------|--------------|
| 50 | building | Metrics state machine |
| 51 | build | Adapter fuzzing |
| 52 | claude | JSON parsing, stream interruption |
| 53 | dependency | Import extraction edge cases |
| 54 | belt | Stress tests |
| 55 | advisor | Property tests, pool lifecycle |
| 56 | debug | Snapshot isolation, delve lifecycle |
| 57 | connection | Graph algorithms |
| 58 | game | Scene transitions, loop determinism |
| 59 | production | File watcher accuracy |
| 60 | input | Vim mode state machine |
| 61 | unit | Coverage edge cases |
| 62 | resources | Metric bounds |
| 63 | harness | Lifecycle state machine, backpressure |
| 64 | tile | Fog state machine |
| 65 | multiagent | **CRITICAL: TOCTOU race**, snapshots |
| 66 | ui | Layout math |
| 67 | testutil | Meta-testing |
| 68 | mapview | Coordinate math, zoom/pan |
| 69 | systemtest | Missing interactions |

### Critical Bugs to Prioritize

1. **multiagent TOCTOU race** (Issue #65) - HandoffTask has check-then-act race
2. **multiagent Snapshots() consistency** - Lock released before snapshots taken
3. **harness backpressure** - No test for buffer-full scenario

### Tmux Windows

All in session `CodingGame`:
- review-advisor, review-belt, review-build, review-building
- review-capability, review-claude, review-connection, review-debug
- review-dependency, review-game, review-harness, review-input
- review-mapview, review-multiagent, review-production, review-resources
- review-systemtest, review-testutil, review-tile, review-ui, review-unit

### Next Session Actions

1. Send instructions to all 21 agents via claude-send.sh
2. Monitor for PRs: `gh pr list --state open`
3. Monitor for blockers: `gh issue list --label blocker`
4. Review PRs as they come in
5. Merge PR #70 (refactoring → master) after review
