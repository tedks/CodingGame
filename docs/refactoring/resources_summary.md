# internal/resources refactoring notes

## Overview
Resource bar tracker with default metrics (context, API cost, coverage) and rendering helpers. Uses a mutex to guard the resource slice.

## Code smells / risks
- `GetResource` returns a pointer to an internal `Resource` without copy; callers can mutate without locks, causing data races or invariant breaks.
- `Draw` divides by `len(t.resources)` without guarding against zero; if resources list is emptied or not initialized, this will panic.
- `AddResource` does not validate `nil` input or enforce unique names; duplicate names make `UpdateResource` ambiguous.
- `UpdateResource` silently ignores unknown names (no error or return value). Hard to debug missing updates.
- `Update()` has no locking; currently a no-op, but if it later touches shared state it will need a lock.
- Hard-coded default max values (context 200k, cost $100, coverage 100) are not configurable and may drift from reality.

## Abstraction / design
- `Resource` is mutable and exported; consider encapsulating updates via methods on `Tracker` to keep invariants (non-negative, max >= current).
- There’s no unit enforcement; `Unit` is free-form and formatting depends on string matching.

## Tests
- Concurrency test does not check for race conditions; consider running with `--race` in CI to validate.
- No tests for empty resource list or invalid resource inputs.

## Suggested follow-ups
- Return copies from `GetResource`, or expose a read-only view to prevent external mutation.
- Add guard for empty resources in `Draw` and/or ensure defaults always exist.
- Add `SetResource`/`UpsertResource` APIs with validation and result status.
- Make default resource limits configurable via config or runtime capability discovery.
