# internal/tile refactoring notes

## Overview
Tile model for files/directories with fog-of-war state, reveal count, metadata, and highlight timing. Uses a RWMutex for thread safety.

## Code smells / risks
- `Highlight`/`IsHighlighted` are time-based and use `time.Now()` directly, which makes tests flaky; test uses `time.Sleep` for expiration.
- `ResetFog` does not reset `lastRevealed` or `revealCount`; the tile can be “fully fogged” but still shows stale reveal metadata (might be intended, but ambiguous).
- `MarkStale` only transitions from `FogRevealed` to `FogStale`; calling it while `FogFull` is a no-op, which may hide state errors.

## Abstraction / design
- Tile exposes only getters/setters but no explicit invariants (e.g., `revealCount` should correlate with `lastRevealed`). Consider a single `SetFogState` with explicit transitions.
- No way to clear `lastError`/metadata or to derive “needs refresh” from `lastModified` vs. `lastRevealed`.

## Tests
- `TestHighlight` uses `time.Sleep(150ms)`, which can be flaky and violates the no-sleep synchronization guidance. Use a controllable clock or inject time.
- Concurrency test only checks for deadlocks/races; no data invariants checked.

## Suggested follow-ups
- Inject a clock or add a `HighlightUntil` setter to make highlight tests deterministic.
- Clarify fog reset semantics (should `lastRevealed`/`revealCount` be cleared?).
- Add explicit transitions or validation for fog state changes.
