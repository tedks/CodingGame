# internal/capability summary

## Overview
- Discovers, tracks, and renders Claude Code capabilities (tools, MCP servers, commands) and supports polling-based config watching.

## Strengths
- Clear domain/type modeling and registry abstraction.
- MCP discovery handles multiple config locations and gracefully degrades on malformed files.
- Rendering is isolated from discovery/registry logic.

## Code smells / risks (prioritized)
- **Renderer order nondeterminism**: capabilities within each domain are drawn in map iteration order, which is randomized; UI ordering can flicker between frames.
- **Watcher data races**: `fileTimes` is accessed without synchronization; `ForceRefresh()` and `poll()` can run concurrently, causing races.
- **Builtin tool list drift**: builtin tools are hardcoded and may not match actual available tools (no source of truth or sync mechanism).
- **Listener slice shared with goroutines**: registry listeners receive a shared slice; a misbehaving listener could mutate it and affect others.

## Abstraction tightening opportunities
- Sort capabilities within each domain in `Renderer.Draw` (by name/type) to stabilize layout.
- Add a mutex around `fileTimes` or ensure all access happens on the polling goroutine; make `ForceRefresh` signal the goroutine instead of touching state directly.
- Consider a `CapabilityProvider` or config-driven builtin list to avoid divergence between UI and actual tool availability.

## Testing gaps and suggested tests
- Add tests for `Watcher` (poll interval behavior, detecting file add/remove/modify, and concurrency safety).
- Add tests for renderer ordering (given unordered input, output order should be stable).
- Add tests that registry listeners receive a copy to prevent accidental mutation effects.

## Documentation gaps
- Document ordering guarantees for registry/renderer outputs.
- Document watcher concurrency expectations and when `ForceRefresh` is safe to call.

## Refactor candidates
- **Small**: Sort capabilities by name within each domain before rendering.
- **Medium**: Add synchronization around `Watcher.fileTimes` and `ForceRefresh()`.
- **Medium**: Provide a deterministic source for builtin tools (config file or generated list).

## Dependencies to review further
- `internal/ui` (where the capability renderer is invoked and how re-rendering is triggered)
