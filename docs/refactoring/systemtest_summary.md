# internal/systemtest refactoring notes

## Overview
System test suite built around a single `TestSystemTests` entry point due to GLFW constraints. Tests exercise input handler, menu/prompt UI, and some MapView mouse interactions.

## Code smells / risks
- Several tests have no assertions and effectively do nothing (e.g., navigation tests check `IsActionHeld` but don’t fail when false; views toggle test doesn’t assert). These won’t catch regressions.
- MapView mouse tests mostly don’t assert outcomes due to uncertain tile positions. They compile but provide little signal.
- System tests are skipped when `DISPLAY`/`WAYLAND_DISPLAY` are missing even though many tests don’t require a real display (they just drive `input.Handler` and UI structs). This reduces coverage in CI unless xvfb is configured correctly.
- The project guidance says to use the Scenario DSL for system tests, but these tests manually call `Queue*` and `Update()`; consistency and readability suffer.

## Abstraction / design
- Tests are split into many small helper-style functions, but `main_test.go` is the only runner. There’s no grouping by capability for targeted runs (e.g., build tags or separate suites for headless vs. display).
- Input-action tests use the handler directly rather than a real scene/viewport; end-to-end intent is partial.

## Testing improvements
- Convert “no-op” tests to assert on concrete state changes (action fired, view changed, selection updated).
- For MapView, expose deterministic tile lookup for tests (e.g., helper to find the first tile’s screen rect) and assert selection/callbacks accordingly.
- Split tests into headless-safe and display-required groups; only skip display-required ones.
- Consider using `testutil.Scenario` for consistent event scripting and readability.

## Suggested follow-ups
- Add real assertions in navigation/view toggle tests.
- Update MapView tests to compute tile coordinates from the layout instead of hard-coded guesses.
- Add coverage for handler → scene integration (system test requirement for input wiring).
