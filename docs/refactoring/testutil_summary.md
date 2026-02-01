# internal/testutil refactoring notes

## Overview
Test utilities for input simulation, scenario DSL, screenshot capture, visual diffing, and race verification. Provides `TestInputSource`, `Scenario`, and `RunAndCapture` helpers.

## Code smells / risks
- `ScenarioGame.Update` applies actions after `AdvanceFrame()`, so queued events don’t affect the same frame’s `Update()` and are effectively delayed by one frame. This can cause off-by-one timing in scenario assertions.
- `injectInputSource` claims reflection but only checks for `SetInputSource(interface{})` method; components that expose `SetInputSource(input.InputSource)` won’t get injected. This can silently skip input wiring in scenario tests.
- `TestMain` in `runner_test.go` skips all testutil tests if no display, even though most tests are headless (compare, scenario, input). Coverage is unnecessarily reduced in CI without display.
- `TestInputSource.AdvanceFrame` processes all queued events in one frame with no scheduling; if callers expect per-frame sequencing, they must manually interleave `AdvanceFrame()` calls. This is easy to misuse and should be documented.

## Abstraction / design
- `InputSourceSetter` uses `SetInputSource(source interface{})` rather than the concrete `input.InputSource` interface, which weakens type safety and tooling assistance.
- `Scenario` DSL is not used by system tests despite guidance; consider standardizing on it for readability and deterministic timing.

## Tests
- Good unit coverage for core utilities; however, `runner_test.go` uses `TestMain` to skip tests in headless mode, which may hide regressions.
- No tests for the scenario runner’s frame timing semantics (action apply vs. input consumption). Consider adding explicit timing tests.

## Suggested follow-ups
- Fix scenario timing by applying queued actions before `AdvanceFrame()` or by adding a pre-frame hook.
- Expand input source injection to accept `SetInputSource(input.InputSource)` and/or use reflection to match compatible signatures.
- Split tests into headless-safe vs. display-required; only skip display-required ones in `TestMain`.
- Document `TestInputSource` frame semantics in godoc to prevent misuse.
