# internal/production refactoring notes

## Overview
Production realm models services discovered from `.production.json`, with registry + watcher, and a renderer for service “cities”. Includes unit tests for discovery/registry/watcher/service/renderer.

## Code smells / risks
- `Watcher` accesses `fileTimes` without synchronization. `poll()` runs concurrently with `ForceRefresh()` or `Start()`; both call `updateFileTimes()`/`checkForChanges()` without locking, so data races are possible.
- Watcher tests rely on `time.Sleep()` for synchronization. This is explicitly discouraged; should use channel signaling or a configurable clock/ticker for deterministic tests.
- Registry listener notifications are fire-and-forget goroutines with no timeout or backpressure; a slow listener can pile up goroutines.
- Registry listeners receive a shared `[]*Service` slice (same slice for all listeners). If any listener modifies it, it can affect others. The doc warns “read-only” but no enforcement.
- `Service.UpdateHealth` doesn’t clear `LastError` when a subsequent health update is healthy with empty error; can leave stale error text.
- Renderer layout constants are hard-coded; summary positions (`width-300`) can go negative for narrow widths.
- Renderer city row details can overlap: traffic and dependency text both draw at `y+72`.

## Abstraction / design
- `ConfigDiscoverer.Discover()` ignores parse errors silently by dropping services from malformed files; registry error reporting is never triggered for config errors. Consider exposing errors via the discoverer to surface problems in UI.
- `deriveWeather` uses static thresholds; consider configuring per-service or per-environment to keep “everything is real” aligned with actual SLOs.
- Registry recomputes sorted lists on every `GetAll`/`GetByType`/`GetByHealth`. If called every frame, consider caching or maintaining a stable order on mutation.

## Tests
- Tests cover behavior well but rely on sleeps for file change detection; can be flaky on slow filesystems.
- No tests for watcher concurrency (Start/Stop/ForceRefresh overlap) or for registry listener panic handling.

## Documentation
- `Watcher` already documents polling design. Add concurrency ownership note for `fileTimes` and whether `ForceRefresh` is safe while running.
- Clarify expectations for `.production.json` errors (silent ignore vs. surfaced).

## Suggested follow-ups
- Guard `fileTimes` with a mutex or confine all access to the polling goroutine.
- Add deterministic tests for watcher change detection using explicit hooks or fake time.
- Clamp renderer layout or add responsive behavior for small viewports; avoid overlapping text.
- Clear `Service.LastError` on healthy updates.
