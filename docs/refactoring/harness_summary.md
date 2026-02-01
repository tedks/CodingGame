# internal/harness summary

## Overview
- Defines the harness abstraction for agent CLIs (config, events, registry) plus a Claude Code implementation and a mock harness for tests.

## Strengths
- Clear interface boundaries and config validation.
- Strong event modeling with SafeFilePath support.
- Claude harness handles subprocess lifecycle with careful cleanup and monitoring.
- Extensive tests for parser and event handling.

## Code smells / risks (prioritized)
- **Restart safety**: `BaseHarness` is documented as non-restartable, but `MockHarness.Start()` allows restarting after `Stop()` even though the events channel is already closed.
- **Unsafe sends after stop**: `MockHarness.SimulateEvent` and `Parser` send directly on channels without guarding against closure, risking panics when stopped.
- **BaseHarness data races**: `SetVersion` and `Version` are not protected by a mutex; concurrent access can race.
- **ClaudeHarness restart risk**: `Start` does not prevent reuse after `Stop`, yet the underlying `BaseHarness` channel may be closed.

## Abstraction tightening opportunities
- Add a `closed` flag or `sync.Once`-guarded Close on `BaseHarness`, and prevent restart after stop (or reinitialize channels explicitly).
- Provide a non-blocking event emission helper that returns errors when the channel is closed.
- Align mock harness behavior with documented lifecycle (single-use) or redesign BaseHarness to support restart.

## Testing gaps and suggested tests
- Add tests for restarting a harness (either forbidden or reinitialized) to lock in the intended behavior.
- Add tests asserting `SimulateEvent` does not panic after `Stop()`.
- Add concurrency tests around `SetVersion` once it is locked.

## Documentation gaps
- Clarify whether harness instances are single-use or restartable, and how callers should manage lifecycle.

## Refactor candidates
- **Small**: Protect `BaseHarness.version` access with the mutex.
- **Medium**: Introduce `EmitEvent` helper with closed-channel guard; update MockHarness/Parser to use it.
- **Medium**: Enforce single-use lifecycle across all harness implementations (return error on restart).

## Dependencies to review further
- `internal/game` (harness lifecycle usage and restart behavior)
- `internal/harness/claude` (event channel closing and parser lifecycle)
