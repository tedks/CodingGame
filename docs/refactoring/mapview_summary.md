# internal/mapview summary

## Overview
- Renders the tile-based map view with directory layout, fog of war, selection, zoom/pan, and optional dataflow belts.

## Strengths
- Uses `InputSource` for mouse input, supporting test injection.
- Tree layout is deterministic with directory-first sorting.
- Tests cover layout, zoom math, and selection basics.

## Code smells / risks (prioritized)
- **Potential divide-by-zero**: `drawDataflowView` uses `tilesPerRow := m.width / int(tileSize)` and then `i % tilesPerRow`; if `tileSize > width`, `tilesPerRow` becomes 0 and will panic.
- **View-mode mismatch in selection**: `TileAtScreenPos` always uses `treeLayout` positions even in `ViewDataflow`, so selection can be incorrect in dataflow mode.
- **Nil input source**: `SetInputSource` does not handle `nil` (unlike other packages), risking nil dereference.
- **Stale comment**: `fileTypeFogColor` comment exists but the function is missing.
- **Hidden directory filtering**: excludes most dotfiles but does not skip large common dirs (`node_modules`, `.cache`, `dist`, etc.), potentially slowing scans.

## Abstraction tightening opportunities
- Add a `layoutStrategy` interface or conditional tile lookup for different view modes.
- Centralize ignore patterns with `dependency`/`resources` to avoid inconsistent scanning behavior.
- Add a safe guard for tile size vs viewport width to avoid crash in small window sizes.

## Testing gaps and suggested tests
- Add tests for dataflow view layout (tilePositions mapping and belt rendering) and for selection behavior when in dataflow view.
- Add tests for tiny viewport widths to prevent divide-by-zero regressions.
- Add tests for `SetInputSource(nil)` default behavior once implemented.

## Documentation gaps
- Document which view modes support selection and how tile positions are derived in each mode.
- Document directory filtering rules and rationale.

## Refactor candidates
- **Small**: Guard `tilesPerRow` to be at least 1 and/or early-return if width too small.
- **Small**: Handle `SetInputSource(nil)` by reverting to `input.DefaultSource`.
- **Medium**: Split selection logic by view mode (tree vs dataflow positions).
- **Medium**: Add configurable ignore patterns and share with dependency extraction.

## Dependencies to review further
- `internal/dependency` (path granularity alignment for dataflow graph)
- `internal/belt` (rendering expectations for connection graph)
