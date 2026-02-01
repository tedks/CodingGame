# internal/multiagent refactoring notes

## Overview
Multi-agent orchestration model with agents, orchestrator, and renderer. Thread-safe-ish with per-agent locks and an orchestrator RWMutex. Asynchronous listener notifications with timeout.

## Code smells / risks
- Listener notification timeout can leak goroutines: on timeout, the goroutine waiting on `wg.Wait()` never exits if a listener is stuck, plus the stuck listener goroutine itself remains. Consider per-listener timeout/cancellation and avoid a shared `wg` wait goroutine that can block forever.
- Timeout accounting is coarse: `abandonedListenerCount` increments per notification batch, not per listener. If multiple listeners are stuck, metrics undercount and are ambiguous.
- `StartTask` accepts an empty task string; `Orchestrator.AssignTask` checks, but direct `Agent.StartTask("")` can leave blank tasks. Consider validating at agent level or documenting that only orchestrator should call `StartTask`.
- `Agent.UpdateTokenUsage` does not clamp `contextUsage` to `[0,1]` when tokens exceed limit; renderer uses this value for bar width, which can overflow. Clamp to bounds and handle negative values defensively.
- `DefaultTokenLimit` comment says “update when model context sizes change” but no runtime config; consider pulling from config to avoid staleness.
- `Renderer` hard-coded layout values; doesn’t handle narrow widths (header token summary uses `width-200` which can go negative). Add layout bounds checks or responsive behavior.
- `Renderer` status/usage colors are globals; consider moving to theme or UI palette to stay consistent with the rest of the UI.

## Abstraction / design
- Orchestrator owns a map but allows external mutation of agents without notifying listeners. If UI relies on listeners for updates, consider wrapper methods (e.g., `UpdateAgentStatus`), or document that callers must notify.
- `GetAll` sorts by agent name on every call; if used frequently in render loop, consider returning stable order maintained at mutation time, or cache with invalidation.
- `GetSharedFiles` relies on `FilesRead()` copying maps; could be heavy for large contexts. Consider incremental tracking or limiting to changed agents.

## Tests
- Tests cover agent lifecycle, orchestrator behavior, concurrent access, and listener panic recovery.
- No tests cover renderer output/layout bounds. Consider snapshot/image tests or validate no panic under narrow `width/height`.
- Concurrency tests don’t run with race detector by default; consider adding a Bazel target or CI job for `--race` when multi-agent work ramps up.

## Documentation
- Document listener timeout behavior and goroutine leak tradeoffs (abandoned vs. awaited) in package docs or in `notifyListeners` comment.
- Clarify whether orchestrator is the only intended mutator of agent status/task.

## Suggested follow-ups
- Add a bounded listener notification strategy (per-listener timeout and cancellation, or queue-based fan-out) and tests verifying no goroutine leaks.
- Clamp context usage in `Agent.UpdateTokenUsage`, and add tests for over/under limits.
- Add layout safety checks in renderer when width/height are small.
