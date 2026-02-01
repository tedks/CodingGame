# internal/building summary

## Overview
- Represents build targets as “buildings” with state, location, and aggregated build metrics/history.

## Strengths
- Good encapsulation of metrics and state with concurrency-safe access.
- Trend analysis and cache metrics provide meaningful, real data views.
- Extensive tests for metrics, trends, and concurrency.

## Code smells / risks (prioritized)
- **Integration tests not wired**: `internal/building/integration_test.go` is not included in `internal/building/BUILD.bazel`, so it doesn’t run under Bazel.
- **Unused fields/types**: `buildQueue` and `BuildRequest` are defined but unused (API drift / incomplete feature).
- **MinDuration sentinel**: `Metrics.MinDuration` initializes to max int64 and remains so until at least one build; consumers may see nonsense values.
- **State invariants not enforced**: `RecordBuild` doesn’t verify `StateBuilding` or prevent concurrent builds; `StartBuild` doesn’t check current state.

## Abstraction tightening opportunities
- Introduce a `BuildLifecycle` helper or `BuildingRunner` to manage state transitions and enforce invariants (start → record → idle/success/failed).
- Consider moving queueing logic into a dedicated scheduler/manager so `Building` focuses on state + metrics only.

## Testing gaps and suggested tests
- Wire integration tests into Bazel or split into a separate `go_test` target.
- Add tests for `MinDuration` when no builds (define expected zero behavior).
- Add tests for invalid transitions (e.g., `RecordBuild` without `StartBuild`) once behavior is defined.

## Documentation gaps
- Document expected state transitions and whether `RecordBuild` is safe to call without `StartBuild`.
- Document how/if `buildQueue` is intended to be used, or remove until needed.

## Refactor candidates
- **Small**: Add `integration_test.go` to `BUILD.bazel`.
- **Small**: Initialize metrics with zero/`nil` state (avoid sentinel max duration) and document defaults.
- **Medium**: Enforce state transitions (guard multiple starts or record without start).
- **Large**: Extract queue/scheduling into a separate component and remove unused fields from `Building`.

## Dependencies to review further
- `internal/build` (adapter result semantics and cache metrics)
