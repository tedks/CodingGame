# internal/advisor summary

## Overview
- Manages advisor configurations, advisor state/metrics, insight lifecycle, and advisor pool execution (including harness integration).

## Strengths
- Clear separation between config parsing, advisor state, insight model, and pool management.
- Extensive unit and integration tests for most public behaviors.
- Metrics tracked explicitly (runs, tokens, durations) and exposed via helper methods.

## Code smells / risks (prioritized)
- **Glob matching mismatch**: `Config.MatchesFile` uses `filepath.Match`, which does **not** support `**` globs, but defaults include `**/auth/**`, `**/*.go`, etc. This likely never matches as intended.
- **Unused notification hooks**: `Pool.notifyInsight` and `Pool.notifyStateChange` exist but are never called; listener events only cover add/remove, leaving insight/state changes unobserved.
- **StartAnalysis result ignored**: `Pool.RunAdvisor` calls `advisor.StartAnalysis()` and ignores its boolean result; double-runs could overlap and corrupt metrics/state.
- **MinDuration sentinel**: `AdvisorMetrics.MinDuration` initialized to max int64 and exposed through `Metrics()` even when no runs have occurred; consumers may see a nonsensical huge duration.
- **Background loop no effect**: `runBackgroundAdvisor` ticks but does not actually run analysis or emit state changes, which may be misleading in UI/metrics.

## Abstraction tightening opportunities
- Introduce a dedicated glob matcher (e.g., doublestar) and centralize pattern validation to align config, defaults, and runtime behavior.
- Move advisor execution and state transitions behind a small interface (e.g., `Runner`) to make `Pool` orchestration testable without harness and to enforce start/complete invariants.
- Replace direct access to `Advisor.Config()` in `Pool.RunAdvisor` with explicit getters for harness config to keep `Config` immutable and reduce accidental copying.

## Testing gaps and suggested tests
- Add tests for `**` pattern matching once matcher is upgraded (regression for current default configs).
- Add integration test that verifies pool listener callbacks for insight generation and state changes once those notifications are wired.
- Add test for `RunAdvisor` handling when `StartAnalysis()` returns false (should refuse to run or cancel).

## Documentation gaps
- Document expected glob semantics in `Config.FocusPatterns` (currently implied but incorrect for `**`).
- Document concurrency model for `Pool` (goroutine ownership, channel closure, stop behavior), especially around `RunAdvisorAsync`.

## Refactor candidates
- **Small**: Wire `notifyInsight` / `notifyStateChange` into `Advisor.AddInsight` and state transitions (via pool hooks or explicit calls in `RunAdvisor`).
- **Medium**: Replace `filepath.Match` with a glob library that supports `**` and validate patterns early (with error reporting).
- **Medium**: Add guardrails to prevent overlapping advisor runs (enforce `StartAnalysis` success in `RunAdvisor`).
- **Large**: Rework background scheduling to actually run advisors or clearly mark as stub to avoid misleading “background” trigger.

## Dependencies to review further
- `internal/harness` (advisor execution and event parsing)
- `internal/ui` (if advisor state/insights are rendered and rely on pool notifications)
