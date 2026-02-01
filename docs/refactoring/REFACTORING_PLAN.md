# Refactoring Plan (Merged & Detailed)

This plan merges the internal scan findings with the effort × priority matrix into a single, comprehensive execution roadmap. It is ordered by impact and effort, with explicit phases, targeted files, and rationale.

## Guiding principles
- Fix correctness and safety before architecture.
- Eliminate silent failures and flaky tests early.
- Enforce input abstraction and lifecycle semantics to keep system tests valid.
- Prefer copy-on-read for shared state and explicit invariants for stateful structs.

---

## Phase 0 — Prep (No code changes)
**Goal:** Align on sequencing, scope, and constraints before edits.

- Confirm branch: `refactoring`.
- Re-check AGENTS instructions and constraints (no direct master pushes, test rules, etc.).
- Inventory target files per phase to avoid overlap and ensure clean PR slices.
- Decide whether to split work into multiple PRs (recommended: per phase or per 1–2 themes).

---

## Phase 1 — Silent failures & crash fixes (High priority / Low effort)
**Goal:** Eliminate silent regressions, crashes, and broken features with minimal code changes.

1) **Wire missing integration tests into Bazel**
   - `internal/build/integration_test.go`
   - `internal/building/integration_test.go`
   - Add to relevant `BUILD.bazel` targets.
   - Rationale: tests exist but never run → silent regressions.

2) **Fix InputSource violation in GameScene**
   - `GameScene.handlePromptPanelDrag` uses direct Ebiten calls.
   - Replace with `InputSource` calls to restore system test injection.
   - Rationale: breaks testability contract; small localized fix.

3) **Add nil/divide-by-zero guards**
   - `mapview`: protect against `tilesPerRow == 0`.
   - `resources.Draw`: guard `len(resources)==0`.
   - `connection.Graph.Add(nil)`: no-op for nil input.
   - Renderers: guard `width-200/300` from going negative (production/multiagent/capability).
   - Rationale: prevent runtime panics and undefined behavior.

4) **Fix glob mismatch in advisor**
   - `filepath.Match` does not support `**` but defaults use it.
   - Replace with `doublestar` (or adjust default patterns).
   - Rationale: feature silently broken for default patterns.

5) **Remove clearly dead/unused code**
   - `belt.frameCount`
   - `connection.analysisValid`
   - `building.buildQueue` / `BuildRequest`
   - `advisor` notify hooks (or wire up if intended)
   - `BuildOptions.Verbose` if never used
   - Rationale: reduces confusion about “real” features.

---

## Phase 2 — Concurrency & lifecycle safety (High priority / Medium effort)
**Goal:** Remove data races, channel panics, goroutine leaks, and undefined lifecycle semantics.

1) **Watchers / shared state race fixes**
   - `capability.Watcher.fileTimes`
   - `production.Watcher.fileTimes`
   - `harness.BaseHarness.version`
   - `debug.EventBus`
   - Options: guard with mutexes or confine mutation to a single goroutine.

2) **Channel safety & close semantics**
   - Add `sync.Once` for channel close in `harness.BaseHarness`, `claude.Interceptor`.
   - Guard send-on-closed in `harness.MockHarness`, `harness.Parser`, `claude.Interceptor`.
   - Document ownership and close responsibility per channel.

3) **Goroutine leak prevention**
   - `multiagent` listener notification uses timeout that abandons goroutines.
   - Replace with per-listener timeout or bounded worker queue.
   - Ensure no orphaned `wg.Wait()` goroutines.

4) **Lifecycle/restart semantics**
   - Decide restartable vs non-restartable harnesses.
   - Enforce in `Game.SetConfig` and harness implementations.
   - Update docs/tests to reflect lifecycle rules.

---

## Phase 3 — Test infrastructure & determinism (Medium priority / Medium effort)
**Goal:** Make tests deterministic, avoid over-skipping, and enforce input abstraction.

1) **Replace `time.Sleep()` synchronization**
   - Files: game harness integration tests, tile highlight tests, production watcher tests, capability watcher tests.
   - Introduce `Clock` interface or testutil hooks; use channels for signaling.
   - Avoid time-based flakiness and align with no-sleep guidance.

2) **Split headless vs display tests**
   - Packages: `internal/input`, `internal/ui`, `internal/testutil`, `internal/systemtest`.
   - Headless-safe tests should run without `DISPLAY`.
   - Display-required tests move into `_display_test.go` or tagged targets.

3) **System tests with real assertions**
   - Add concrete assertions to navigation/mapview tests.
   - Use Scenario DSL where possible for clarity and repeatability.

---

## Phase 4 — Data integrity & API boundaries (Medium priority / Low effort)
**Goal:** Prevent external mutation, clamp invalid values, and normalize “no data” semantics.

1) **Fix MinDuration sentinel values**
   - `internal/advisor.AdvisorMetrics`
   - `internal/building.Metrics`
   - `internal/unit.TestMetrics`
   - Replace `math.MaxInt64` sentinel with zero/optional or `HasData` flag.

2) **Clamp percentages and ratios**
   - `unit` coverage, `multiagent` context usage, any UI bar fill math.
   - Clamp to [0,1] or [0,100] before rendering.

3) **Return copies instead of internal references**
   - `resources.GetResource()`
   - `unit.LastTestResult()`
   - `debug.SetFrames()`
   - Registry listener slices
   - Prevent external mutation without locks.

---

## Phase 5 — Consolidation & architecture (Medium priority / High effort)
**Goal:** Reduce duplication, centralize UI layout, and introduce explicit invariants.

1) **Deduplicate Game vs GameScene logic**
   - Event parsing, advisor triggering, path resolution.
   - Decide primary path and deprecate legacy code.

2) **Centralize layout/theme constants**
   - Create shared theme/palette/layout helpers.
   - Replace magic numbers across renderers.

3) **Extract shared renderer patterns**
   - Consolidate card/header layouts (multiagent/production/capability, etc.).

4) **Improve “struct + mutex + getters” pattern**
   - Add explicit state transitions or state machines where needed.
   - Introduce immutable snapshots for read-only access.
   - Document invariants and ownership (especially for concurrency).

---

## Additional notes / cross-cutting items
- **Glob mismatch**: defaults using `**` are effectively broken with `filepath.Match`.
- **Over-skipping tests**: currently hides regressions in CI.
- **Dead code**: removing unused fields reduces confusion about what is real.
- **Renderer bounds**: any negative width calculation should clamp or early-return.

---

## Suggested execution order (if splitting into PRs)
1) Phase 1 (silent failures + crash guards)
2) Phase 2 (concurrency + lifecycle)
3) Phase 3 (test infra + determinism)
4) Phase 4 (data integrity)
5) Phase 5 (consolidation)

---

## Expected outcomes
- Fewer crashes and silent test omissions.
- Deterministic tests with better CI coverage.
- No send-on-closed panics or watcher data races.
- Cleaner API boundaries and more reliable renderers.
- A stable foundation for larger architectural refactors.
