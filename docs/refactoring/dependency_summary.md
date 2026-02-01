# internal/dependency summary

## Overview
- Extracts dependency relationships from Go source imports into a `connection.Graph`, with optional symbol-level coupling strength.

## Strengths
- Clear separation between fast import-only pass and more expensive symbol pass.
- Tests cover module detection, extraction, and symbol counting.

## Code smells / risks (prioritized)
- **Package-to-file mismatch**: internal imports are mapped to package directories (e.g., `internal/pkga`) but consumers are file paths; this can mix path granularities and confuse map rendering.
- **No exclusion for common large dirs**: walker skips `.git`, `vendor`, `testdata` but not `node_modules`, `bazel-*`, or build output dirs.
- **Logging side effects**: parse errors use the global `log.Printf`, which can spam output and is not injectable or testable.
- **`ExtractGoWithSymbols` unused variables**: `files` and `pkgPath` are collected but not used, indicating partial/incomplete logic.

## Abstraction tightening opportunities
- Decide on a consistent graph node granularity (package vs file) and normalize both ends to match map tiles.
- Inject a logger or return aggregated warnings instead of using `log.Printf` directly.
- Add configurable exclude list (e.g., from config or defaults shared with mapview indexing).

## Testing gaps and suggested tests
- Add tests for mixed granularity (ensure internal imports produce nodes that exist in the map view).
- Add tests for exclusion of `node_modules`/`bazel-*` if implemented.

## Documentation gaps
- Document the current granularity choice and its impact on the visualization.
- Document how external imports are represented (node naming/placement).

## Refactor candidates
- **Small**: Remove unused locals in `ExtractGoWithSymbols` or use them for deterministic ordering.
- **Medium**: Normalize internal import targets to file-level (e.g., first file in package) or shift the graph to package-level consistently.
- **Medium**: Replace global logging with injected logger or error collection.

## Dependencies to review further
- `internal/connection` (graph node expectations)
- `internal/mapview` (tile naming/lookup for dependencies)
