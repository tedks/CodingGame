# internal/input summary

## Overview
- Provides input abstraction with modes, focus management, and keybindings (vim/emacs styles).

## Strengths
- Clear mode/focus model and binding separation by style.
- InputSource interface enables test injection and supports mouse input.

## Code smells / risks (prioritized)
- **Test skip on no display**: `mode_test.go` skips the entire package if DISPLAY is unset, even though most tests do not require a display.
- **Binding iteration without lock**: `Handler.IsActionHeld` accesses `bindings.bindings` directly without any lock on the `Bindings` struct (minor risk if bindings are modified concurrently).

## Abstraction tightening opportunities
- Provide a thread-safe wrapper or copy of bindings for read-only access during `Update`/`IsActionHeld`.
- Remove display dependency from non-Ebiten tests, or move display-required tests into a separate file/target with build tags.

## Testing gaps and suggested tests
- Add tests for `InputSource` integration (e.g., a test input source driving `Handler.Update`).
- Add tests for custom bindings with modifiers and mode restrictions.

## Documentation gaps
- Document expected concurrency model (single-threaded update loop) to justify lack of locking on `Bindings`.

## Refactor candidates
- **Small**: Remove `TestMain` display check from `mode_test.go` or gate only tests that need Ebiten.
- **Medium**: Make `Bindings` read-only snapshots in `Handler` to avoid concurrent mutation issues.

## Dependencies to review further
- `internal/ui` and `internal/game` (should use `InputSource` for all input, including mouse)
