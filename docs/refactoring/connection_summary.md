# internal/connection summary

## Overview
- Models dependency connections and provides a graph with indexing, circular dependency detection, and basic graph analytics.

## Strengths
- Thread-safe access across connection and graph types.
- Good indexing for from/to queries and cycle detection with Tarjan SCC.
- Tests cover most public APIs, including concurrency.

## Code smells / risks (prioritized)
- **Unused analysis cache**: `analysisValid` is set but never checked; cached circular results are never reused.
- **Comment mismatch**: `SetStrength` comment says “0 strength means minimal coupling (1 symbol)” but it sets `strength = 0`.
- **Nil safety**: `Graph.Add` does not guard against `nil` connections; a nil input will panic.

## Abstraction tightening opportunities
- Either remove `analysisValid` or make `DetectCircular`/`CircularPaths` respect it to avoid unnecessary recomputation.
- Clarify and normalize the meaning of strength (0 vs 1) to keep renderers/metrics consistent.

## Testing gaps and suggested tests
- Add tests for `Graph.Add(nil)` behavior once defined.
- Add tests verifying caching behavior if `analysisValid` is used.

## Documentation gaps
- Document whether `DetectCircular` must be called before `CircularPaths`/`HasCircularDependencies` and whether results are cached.
- Clarify semantics of `strength` in the package-level docs.

## Refactor candidates
- **Small**: Align `SetStrength` comments with actual behavior (or clamp to min 1 if that’s the intended invariant).
- **Medium**: Implement or remove `analysisValid` caching.

## Dependencies to review further
- `internal/belt` (uses `Connection.Strength` and `IsCircular` for rendering)
