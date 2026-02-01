# internal/belt summary

## Overview
- Renders dependency connections as animated “belts” between tiles using Ebiten/vector.

## Strengths
- Simple API surface with clear parameters and configurable colors/widths/dash patterns.
- Tests cover color/width helpers and basic safety (nil graph, self-references).

## Code smells / risks (prioritized)
- **Unused state**: `frameCount` is declared but never used.
- **Nil screen assumptions**: `Draw` assumes `screen` is non-nil; tests pass `nil` but only succeed because early exits prevent draw calls. A real call with nil `screen` would panic.
- **Animation timing coupled to wall clock**: uses `time.Since(startTime)` directly, making deterministic rendering/tests difficult.

## Abstraction tightening opportunities
- Accept a time source or `delta` in `Draw` to decouple rendering from wall time and allow deterministic tests.
- Introduce a small `RendererConfig` struct to group configurable fields (colors, widths, dash pattern) and allow validation.

## Testing gaps and suggested tests
- Add a golden/snapshot test for belt rendering in `internal/testutil` to validate dashed/arrow rendering across types.
- Add a test verifying that `Draw` safely no-ops or returns error when `screen` is nil (if you choose to guard explicitly).

## Documentation gaps
- Document expected units (pixels per second, widths) and coordinate space assumptions in the package comment.

## Refactor candidates
- **Small**: Remove `frameCount` or use it to drive animation deterministically.
- **Medium**: Add an optional `SetClock` or `DrawWithTime(t time.Time)` to control animation.
- **Medium**: Guard against nil `screen` or change signature to return error when invalid inputs are supplied.

## Dependencies to review further
- `internal/connection` (connection types/strength semantics)
- `internal/mapview` (tile coordinate system and offsets)
