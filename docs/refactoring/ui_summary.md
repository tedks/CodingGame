# internal/ui refactoring notes

## Overview
UI primitives: `PromptPanel` (conversation/prompt UI), `Menu`, `StartScreen`, and `SceneManager`. Many drawing routines use `ebitenutil.DebugPrint` and manual layout math.

## Code smells / risks
- `PromptPanel.drawConversation` clips using `startY+height-InputAreaHeight` even though `height` already excludes the input area, effectively shrinking the view by `InputAreaHeight` twice.
- `PromptPanel` uses rough character width estimates (`/7`) for layout and wrapping, which can produce bad wraps on different fonts or at small widths.
- `PromptPanel.Submit` doesn’t clear `Text`. Clearing is left to callers (some tests do it manually). This can cause double-submits or stale text if callers forget.
- `StartScreen.handleProjectInput` uses byte slicing for backspace; multi-byte UTF-8 characters will be truncated incorrectly.
- `Menu` doesn’t ensure `SelectedIndex` points to an enabled item on initialization; if the first item is disabled, Enter does nothing and navigation may feel stuck.
- UI tests and scene tests are skipped entirely when no display is present (`TestMain`), even though most tests are headless.

## Abstraction / design
- Layout constants and color schemes are hard-coded in multiple components. Consider centralizing theme/palette and spacing constants for consistency.
- `PromptPanel` mixes rendering and state logic heavily; consider a view-model style separation for easier testing.

## Tests
- PromptPanel tests cover core behavior but don’t assert clipping/wrapping correctness or cursor blink behavior.
- StartScreen tests depend on harness registry contents; tests assume models are present with static names (opus/sonnet/haiku), which may drift.

## Suggested follow-ups
- Fix conversation clipping and add tests that verify visible lines and scroll limits.
- Decide whether `PromptPanel.Submit` should clear text; make it consistent and update tests/callers.
- Make `StartScreen` input editing UTF-8 safe using `[]rune`.
- Add initialization to select the first enabled menu item, or prevent menus with all disabled items.
- Split display-required tests from headless-safe ones to avoid over-skipping.
