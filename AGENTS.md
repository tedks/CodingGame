# Agent Instructions

This document provides comprehensive guidance for Claude Code agents working on the CodingGame project, including workflow management, issue tracking, and session completion requirements.

## Project Overview

CodingGame is a strategy game interface for Claude Code, inspired by Civilization/Call to Power II/Factorio. It transforms software development into an intuitive visual experience where the codebase becomes a strategic map.

**Core Philosophy**: This is a game interface to coding, not a game about coding. Everything shown is real - no fake bonuses, stats, or artificial progression.

**Implementation**: Go + Ebitengine GUI application. See [ARCHITECTURE.md](ARCHITECTURE.md) for technical details and [DESIGN.md](DESIGN.md) for complete specifications.

## Implementation Status

Run `bd list --all` for current issue state. Phase completion:

| Phase | Name | Status | Epic |
|-------|------|--------|------|
| 0 | Foundation | Complete | CodingGame-w93 |
| 1 | Core Framework | Complete | CodingGame-66a |
| 2 | Buildings & Units | Complete | CodingGame-bac |
| 3 | Advisors | Complete | CodingGame-3p5 |
| 4 | Belts & Debugging | Complete | CodingGame-mrj |
| 5 | Capability Inventory | In Progress | CodingGame-gqo |
| 6 | Production Realm | Not Started | CodingGame-2lu |
| 7 | Multi-Agent | Not Started | CodingGame-0j6 |
| 8 | Polish | Not Started | CodingGame-4kb |

## Beads Issue Tracking

This project uses **bd** (beads) for issue tracking. Run `bd ready` to get started.

**Important**: Beads syncs to the `beads-metadata` branch, NOT your working branch. This keeps issue tracking metadata separate from code changes. The setup script configures this automatically.

## Git Workflow

**CRITICAL: NEVER push directly to master. Always use branches and pull requests.**

### Branch Requirements

All code changes must go through a branch and pull request:

```bash
# Create a descriptive branch name
git checkout -b feature/add-wayland-support
git checkout -b fix/egl-display-error
git checkout -b docs/update-agents-md

# Make your changes, commit, and push the branch
git add .
git commit -m "Description of changes"
git push -u origin feature/add-wayland-support

# Create a pull request
gh pr create --title "Add Wayland support" --body "Description..."
```

### Branch Naming Conventions

Use prefixes to categorize branches:
- `feature/` - New features or capabilities
- `fix/` - Bug fixes
- `docs/` - Documentation updates
- `refactor/` - Code refactoring without behavior changes
- `test/` - Test additions or improvements
- `chore/` - Maintenance tasks (dependencies, CI, etc.)

### Why Branches?

- **Code review**: PRs allow review before merging
- **CI validation**: Tests run on PRs before merge
- **Rollback**: Easy to revert a PR if issues arise
- **History**: Clear record of what changed and why
- **Collaboration**: Multiple machines/sessions can work on the same branch

### Exceptions

The only acceptable direct pushes to master are:
- Emergency hotfixes (document in commit message why PR was skipped)
- Automated commits from CI/bots (e.g., beads sync)

Even for emergencies, prefer a fast-tracked PR if possible.

## First-Time Setup

Run the setup script to install all dependencies:

```bash
./scripts/setup.sh              # Full setup (Nix + Bazel + beads)
./scripts/setup.sh --beads-only # Just beads (if Nix/Bazel already installed)
./scripts/setup.sh --verify     # Check existing setup
```

The script will:
1. Install Nix package manager with flakes support
2. Install direnv for automatic environment loading (optional)
3. Verify Bazel and Go are available
4. Install and configure beads (syncs to `beads-metadata` branch)

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

5. **PUSH BRANCH AND CREATE PR** - This is MANDATORY:
   ```bash
   # Ensure you're on a feature branch, NOT master
   git branch  # Should show feature/*, fix/*, docs/*, etc.

   # Push branch to remote
   git push -u origin $(git branch --show-current)

   # Create pull request
   gh pr create --title "Description" --body "Details..."

   # Sync beads if installed
   bd sync
   ```

6. **Clean up** - Clear stashes, prune remote branches

7. **Verify** - All changes committed AND pushed, PR created

8. **Hand off** - Provide context for next session (include PR link)

**CRITICAL RULES:**
- NEVER push directly to master - always use branches and PRs
- Work is NOT complete until branch is pushed and PR is created
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds
- ALWAYS verify Nix/Bazel environment before running quality gates
- Include the PR URL in your handoff notes

## Merging Pull Requests

When merging PRs, follow these practices:

```bash
# Use normal merge (not squash or rebase) to preserve commit history
gh pr merge <PR-number> --merge

# Do NOT delete the branch immediately after merge
# Branches should be kept for reference and potential follow-up work
```

**Merge guidelines:**
- Use `--merge` (normal merge commit) to preserve full commit history
- Do NOT use `--squash` or `--rebase` unless explicitly requested
- Do NOT delete branches immediately after merge (`--delete-branch`)
- Wait for CI to pass before merging
- Sync beads after merge: `bd sync`

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

**Generate .bazelrc.user (first time or after Nix update):**

The `.bazelrc.user` file contains machine-specific Nix store paths needed for CGO compilation. Generate it with:

```bash
./scripts/gen-bazelrc-user.sh
```

This is required the first time you set up the project, and should be re-run if you update Nix packages.

### Build Commands

```bash
# Build all targets
bazel build //...

# Build main binary
bazel build //:codinggame

# Build specific package
bazel build //internal/tile
```

### Running the Game

```bash
# Build and run (from Nix environment)
bazel run //:codinggame -- /path/to/project

# Or build then run directly
bazel build //:codinggame
./bazel-bin/codinggame_/codinggame /path/to/project
```

**Keyboard controls:**
- `h/j/k/l` or arrows: Pan map
- `+/-`: Zoom in/out
- `Tab`: Toggle directory/dataflow view
- `Esc`: Exit

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

**Headless environments (no display):** When running without a display (SSH, CI, etc.), use xvfb-run:

```bash
# Run all tests in headless environment
xvfb-run -a -s "-screen 0 1024x768x24 -ac" bazel test //...
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
- Some tests need `xvfb-run` for headless display
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

### Integration Testing Requirements

**CRITICAL**: When connecting two components with a callback, event channel, or any other integration point, you MUST write a test that verifies the integration works end-to-end.

**The Anti-Pattern That Fails:**
1. Build Component A (e.g., harness) and test it in isolation
2. Build Component B (e.g., GameScene callbacks) and test it in isolation
3. Connect A to B with a callback or wire-up code
4. **Skip testing that the connection actually works**
5. Ship broken code that passes all unit tests

**The Required Pattern:**
1. Create a **MockHarness** or equivalent test double that records interactions
2. Write tests that:
   - Inject the mock into the real component
   - Trigger the user-facing action (e.g., submit a prompt)
   - **Verify the mock received the expected call**

**Example - Testing Prompt Submission:**
```go
func TestGameScene_PromptSentToHarness(t *testing.T) {
    // Create mock that records prompts
    mock := harness.NewMockHarness()

    // Inject mock into real component
    scene := NewGameScene(...)
    registry := harness.NewRegistry()
    registry.Register("mock", func() harness.Harness { return mock })
    scene.SetHarnessRegistry(registry)
    scene.SetConfig(ui.GameConfig{Harness: "mock"})

    // Trigger the user action
    scene.onPromptSubmit("test prompt")

    // CRITICAL: Verify the integration worked
    if mock.LastPrompt() != "test prompt" {
        t.Fatal("Prompt never reached harness - integration is broken")
    }
}
```

**When to Apply This Rule:**
- Callback wiring: `component.OnFoo = func() { other.Bar() }`
- Event channels: goroutine reads from channel and calls handler
- Dependency injection: component receives interface, calls methods on it
- Any "glue code" that connects two systems

**The Mock Should Live in the Package:**
- `internal/harness/mock.go` - `MockHarness` for testing harness consumers
- Not just in test files - make it available to other packages' tests

### Writing Concurrent Code

**Before writing any concurrent code**, you MUST document the concurrency design. This prevents the pattern of retrofitting synchronization after bugs are found.

**Required documentation (in code comments or PR description):**

1. **Goroutine inventory**: List all goroutines, their purpose, and lifecycle
   ```go
   // Goroutines:
   // - readOutput: reads stdout, owned by Start(), exits when stdout closes
   // - readErrors: reads stderr, owned by Start(), exits when stderr closes
   // - monitorProcess: watches for crash, owned by Start(), exits when process exits
   ```

2. **Channel ownership**: For each channel, document who creates, writes, reads, and closes
   ```go
   // Channel: events (chan Event, buffered 100)
   // - Created by: NewHarness()
   // - Writers: readOutput, readErrors (via EventsWritable())
   // - Readers: external consumer (via Events())
   // - Closed by: monitorProcess or Stop(), protected by sync.Once
   ```

3. **Mutex coverage**: Document which fields each mutex protects
   ```go
   // mu protects: running, cmd, stdin, stdout, stderr, cancelFunc
   // (events channel has its own closeOnce protection)
   ```

4. **Shutdown ordering**: Document the sequence for graceful shutdown
   ```go
   // Shutdown sequence:
   // 1. Signal done channel (unblocks goroutines waiting on select)
   // 2. Cancel context (signals subprocess to exit)
   // 3. Close stdin (sends EOF to subprocess)
   // 4. Close stdout/stderr (unblocks scanner goroutines)
   // 5. Wait for monitorProcess (which calls cmd.Wait())
   // 6. Wait for reader goroutines (wg.Wait())
   // 7. Close events channel (sync.Once prevents double-close)
   ```

**Concurrency patterns to follow:**

- **Single owner for cmd.Wait()**: Only one goroutine should call `cmd.Wait()` on a process
- **sync.Once for channel close**: Always use `sync.Once` when multiple code paths might close a channel
- **done channel for shutdown signaling**: Use a `done` channel with `select` to allow goroutines to exit gracefully
- **WaitGroup for goroutine tracking**: Track all spawned goroutines so `Stop()` can wait for them
- **Mutex for shared state**: Protect all shared mutable state with a mutex; document what it covers

**PROHIBITION: Avoid `time.Sleep()` for synchronization**

`time.Sleep()` is almost never the right synchronization primitive. It creates race conditions, flaky tests, and wastes time. Before using sleep, you MUST demonstrate that all alternatives have been eliminated:

1. **Channel signaling**: Use a channel to signal completion
   ```go
   // BAD: Sleep and hope
   go doWork()
   time.Sleep(100 * time.Millisecond)
   checkResult()

   // GOOD: Wait for signal
   done := make(chan struct{})
   go func() { doWork(); close(done) }()
   <-done
   checkResult()
   ```

2. **WaitGroup**: Use `sync.WaitGroup` for multiple goroutines
   ```go
   // BAD
   for i := 0; i < 10; i++ { go work(i) }
   time.Sleep(time.Second)

   // GOOD
   var wg sync.WaitGroup
   for i := 0; i < 10; i++ {
       wg.Add(1)
       go func(n int) { defer wg.Done(); work(n) }(i)
   }
   wg.Wait()
   ```

3. **Condition variables**: Use `sync.Cond` for complex waiting conditions

4. **Context with timeout**: Use `context.WithTimeout` for deadline-based waiting

**Acceptable uses of sleep** (must document why alternatives don't work):
- Testing time-based features (e.g., "highlight expires after 100ms")
- Polling external systems with no callback mechanism (e.g., file system watchers)
- Rate limiting or backoff (but prefer `time.Ticker` or `time.After` in select)
- Intentional chaos injection in stress tests (use `time.Microsecond`)

When sleep is genuinely required, use `select` with `time.After` so the wait can be cancelled:
```go
select {
case <-done:
    return
case <-time.After(100 * time.Millisecond):
    // timeout handling
}
```

**When fixing a concurrency bug:**

Before committing the fix, search for the same pattern elsewhere:
```bash
grep -r "go func" --include="*.go" internal/
grep -r "chan " --include="*.go" internal/
grep -r "\.Lock()" --include="*.go" internal/
```

Ask: "Does this bug class exist in other files?"

### Package Structure

| Package | Purpose |
|---------|---------|
| `internal/game/` | Main game loop, scene management |
| `internal/mapview/` | Tile grid, fog of war, pan/zoom |
| `internal/tile/` | File representation, fog states |
| `internal/claude/` | Tool interception, JSON event parsing |
| `internal/advisor/` | Subagent pool, insights system |
| `internal/belt/` | Factorio-style dependency visualization |
| `internal/building/` | Build target visualization |
| `internal/unit/` | Test visualization |
| `internal/input/` | Keyboard handling, vim-style modes |
| `internal/ui/` | Scenes, menus, prompts |
| `internal/build/` | Bazel/npm/cargo adapters |
| `internal/dependency/` | Import extraction (Go) |
| `internal/connection/` | Dependency graph, circular detection |
| `internal/debug/` | Debugger integration (delve) |
| `internal/resources/` | Resource bar, metrics tracking |
| `internal/testutil/` | Screenshot capture, image comparison, input simulation |
| `internal/systemtest/` | Exhaustive system tests for all interactions |

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
- No fake/mock metrics in tests - use real examples
- Tests must pass in Nix environment before committing

### Visual Testing

The `internal/testutil` package provides screenshot capture and image comparison:

```go
import "github.com/tedks/CodingGame/internal/testutil"

// Capture a screenshot during a test
screenshot, err := testutil.RunAndCapture(game, 5, 1280, 720)

// Save to file for manual inspection
screenshot.SaveToFile("/tmp/test_screenshot.png")

// Or save to the standard screenshot directory
dir, _ := testutil.ScreenshotDir()  // /tmp/codinggame-screenshots/
path, _ := screenshot.SaveWithTimestamp(dir)

// Compare against a reference image
result, err := testutil.CompareScreenshotToFile(screenshot, "golden/expected.png",
    testutil.CompareOptions{Tolerance: 5, PercentThreshold: 0.1})
if !result.Match {
    testutil.SaveDiff(screenshot.Image(), referenceImg, "diff.png")
}
```

**Display requirements**: Tests using Ebitengine check for `DISPLAY` or `WAYLAND_DISPLAY`. In CI, xvfb provides the virtual display.

**Manual visual testing**: Run the game and verify appearance:
```bash
bazel run //:codinggame -- /path/to/project
```

### System Test Requirements

The `internal/systemtest` package provides exhaustive system tests that drive virtual keyboard and mouse input to verify all interactions work end-to-end.

**Mandate**: When adding or modifying interactive features, you MUST:
1. Use the `InputSource` interface for all input reading (not direct Ebitengine calls)
2. Add system tests in `internal/systemtest/` covering the new interaction
3. Use the `Scenario` DSL for declarative test definitions

**Running system tests:**

```bash
# Headless (CI/SSH)
xvfb-run -a -s "-screen 0 1024x768x24 -ac" bazel test //internal/systemtest:systemtest_test

# With display
bazel test //internal/systemtest:systemtest_test
```

**GLFW Constraint**: All system tests run as subtests of a single `TestSystemTests` entry point because GLFW can only be initialized once per process. To add a new test:
1. Create a test function: `func testMyFeature(t *testing.T) { ... }`
2. Register it in `TestSystemTests` in `main_test.go`

**Example test using scenario DSL:**

```go
func testMyFeature(t *testing.T) {
    source := testutil.NewTestInputSource()
    h := testHandler(source)

    // Queue input events
    source.QueueKeyPress(ebiten.KeyI)
    source.AdvanceFrame()
    h.Update()

    // Assert expected state
    assertMode(t, h, input.ModeInsert)
}
```

**Input Abstraction Rule**: All components reading input must:
- Accept `InputSource` as constructor parameter or via `SetInputSource()` setter
- Default to `input.DefaultSource` (real Ebitengine)
- Never call `inpututil.*` or `ebiten.IsKeyPressed()` directly

**Key packages:**
- `internal/input/source.go` - InputSource interface and EbitenInputSource
- `internal/testutil/input.go` - TestInputSource for tests
- `internal/testutil/scenario.go` - Scenario DSL for declarative tests
- `internal/systemtest/` - Exhaustive system tests (40+ interactions)

## Continuous Integration

GitHub Actions runs CI automatically on PRs and pushes to master.

### CI Workflow (`.github/workflows/ci.yml`)

The CI pipeline:
1. Installs Nix with flakes support
2. Caches Nix store for faster builds
3. Generates `.bazelrc.user` with X11 paths
4. Runs `bazel build //...`
5. Runs `bazel test //...` with xvfb (headless X11)
6. Checks code formatting with `go fmt`

### Claude Code Workflow (`.github/workflows/claude.yml`)

When `@claude` is mentioned in issues/PRs, the Claude Code action runs with:
- Nix environment pre-configured
- xvfb installed for X11 tests
- `.bazelrc.user` generated

**Running tests in Claude workflow:**
```bash
xvfb-run -a -s "-screen 0 1024x768x24 -ac" nix develop --command bazel test //...
```

### Checking CI Status

```bash
# List recent workflow runs
gh run list --limit 5

# View specific run
gh run view <run-id>

# Watch a running workflow
gh run watch
```

## PR Review Guidelines

When reviewing pull requests, check for:

- **No fake gamification**: Ensure all displayed metrics are real
- **Keyboard accessibility**: Can everything be done without a mouse?
- **Performance**: Will this run on older hardware?
- **Error handling**: Does it fail gracefully?
- **Real data**: Are we showing actual metrics, not placeholders?
- **Go idioms**: Does the Go code follow standard conventions?
- **Documentation**: Are new features explained?

### Concurrency Review Checklist

For PRs that introduce goroutines, channels, or mutexes, verify:

**Design documentation exists:**
- [ ] Goroutine inventory (what goroutines, who owns them, when do they exit)
- [ ] Channel ownership (who creates, writes, reads, closes each channel)
- [ ] Mutex coverage (which fields does each mutex protect)
- [ ] Shutdown ordering (sequence for graceful termination)

**Common bugs to check for:**
- [ ] **Double Wait()**: Is `cmd.Wait()` called from only one location?
- [ ] **Channel double-close**: Are all channel closes protected by `sync.Once`?
- [ ] **Goroutine leaks**: Does `Stop()` wait for all goroutines to exit?
- [ ] **Send on closed channel**: Can any goroutine send after channel is closed?
- [ ] **Missing mutex**: Is all shared mutable state protected?
- [ ] **Deadlock potential**: Can goroutines block waiting for each other?
- [ ] **Sleep for synchronization**: Is `time.Sleep()` used instead of proper synchronization? (channels, WaitGroup, etc.)

**Testing requirements:**
- [ ] Race detector passes: `bazel test --@io_bazel_rules_go//go/config:race //...`
- [ ] Lifecycle tests exist (start → use → stop)
- [ ] Concurrent access tests exist (multiple goroutines hitting the API)

**When one concurrency bug is found:**
- Request an audit of similar patterns across the codebase
- Check if the same bug class exists in other files

## Related Documentation

- [DESIGN.md](DESIGN.md) - Complete design specification
- [PHILOSOPHY.md](PHILOSOPHY.md) - Core design principles  
- [ARCHITECTURE.md](ARCHITECTURE.md) - Technical architecture
- [.beads/README.md](.beads/README.md) - Beads issue tracking guide

## Troubleshooting

### "command not found: bazel"
You're not in the Nix environment. Run `nix develop` first, or use `direnv allow`.

### Tests skip with "no display available"
Run with xvfb: `xvfb-run -a -s "-screen 0 1024x768x24 -ac" bazel test //...`

### CGO linking errors
Regenerate `.bazelrc.user`: `./scripts/gen-bazelrc-user.sh`

### Beads database locked
In web Claude Code, use no-daemon mode: `BEADS_NO_DAEMON=1 bd ready`

### GLFW initialization errors in tests
Ebitengine can only initialize GLFW once per process. Run integration tests in isolation: `bazel test //internal/testutil:testutil_test`

### Build cache issues
Clear Bazel cache: `bazel clean --expunge`

## Questions or Clarifications?

When uncertain:
1. Check if it violates "everything must be real"
2. Ask: "Does this describe what exists or prescribe a progression?"
3. Verify it matches the stated metaphor
4. Look for similar patterns in existing design docs

Remember: This is a visualization tool for real development activities, not a traditional game with artificial progression systems.
