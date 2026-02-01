# internal/game summary

## Overview
- Core game loop (`Game`) and scene wrapper (`GameScene`) that integrate map view, resources, advisors, harness events, and UI input/prompt handling.

## Strengths
- Clear separation between legacy `Game` and scene-based `GameScene` (UI integration).
- Harness integration tests verify prompt wiring end-to-end.
- Scene supports multiple views (map, tech, production, multi-agent) with renderers.

## Code smells / risks (prioritized)
- **Input abstraction violation**: `GameScene.handlePromptPanelDrag` calls `ebiten.CursorPosition`, `ebiten.Wheel`, and `ebiten.IsMouseButtonPressed` directly, bypassing `InputSource` and breaking system test requirements.
- **Duplicated logic**: `Game` and `GameScene` both implement `handleClaudeEvent`, `parseInsight`, and advisor triggering logic.
- **Harness lifecycle leaks**: `SetConfig` can start a new harness without stopping an existing one; no guard against repeated calls.
- **Path handling inconsistency**: `Game.handleClaudeEvent` uses `filepath.Join(projectPath, path)` without checking for absolute paths (unlike `GameScene.resolveFilePath`).
- **Test synchronization**: `harness_integration_test.go` uses `time.Sleep` for event processing; project guidelines discourage `time.Sleep` for synchronization.

## Abstraction tightening opportunities
- Extract shared event parsing and advisor handling into a reusable helper (used by both `Game` and `GameScene`).
- Move prompt panel mouse handling to `InputSource` to comply with system test rules.
- Provide a `HarnessManager` to own lifecycle, context, and event processing across config changes.

## Testing gaps and suggested tests
- Add tests that assert prompt panel input uses `InputSource` (or system tests validating prompt interactions without direct Ebiten calls).
- Add integration tests for harness reconfiguration (switch harness/model).
- Replace `time.Sleep` in harness tests with channel-based synchronization (e.g., wait for mock harness event receipt).

## Documentation gaps
- Document expected lifecycle of `SetConfig` and whether multiple calls are supported.
- Clarify the difference between legacy interceptor events and harness events.

## Refactor candidates
- **Small**: Normalize file path resolution in `Game.handleClaudeEvent` via `resolveFilePath` helper.
- **Medium**: Deduplicate advisor/insight handling between `Game` and `GameScene`.
- **Medium**: Replace direct Ebiten mouse input in `GameScene` with `InputSource`.
- **Large**: Introduce a unified event pipeline that supports both legacy and harness sources.

## Dependencies to review further
- `internal/input` (InputSource integration for mouse)
- `internal/harness` (event stream lifecycle and close semantics)
