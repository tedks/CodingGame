# internal/unit refactoring notes

## Overview
Models tests as “units” with pass/fail state, run history, and aggregated metrics (duration, pass rate, flakiness, coverage). Thread-safe via RWMutex.

## Code smells / risks
- `TestMetrics.MinDuration` is initialized to max int64 and never reset when there are no runs. Calling `Metrics()` on a new unit yields an absurd min duration.
- `LastTestResult()` returns a pointer to internal state (`lastRun`) without copy; callers can mutate without locks.
- `SetCoverage` doesn’t clamp to [0,100] and accepts negative values.
- Run history is unbounded; long-lived sessions can grow memory without limit.

## Abstraction / design
- `RecordTest` accepts a `TestResult` pointer and stores it directly; consider copying to avoid external mutation or data races.
- Flakiness uses only pass/fail transitions, ignoring timing/variance; that may be fine but should be documented as heuristic, not metric.

## Tests
- Good coverage for metrics and flakiness. There’s no test for “zero runs” metrics returning sensible defaults (min duration, max duration).

## Suggested follow-ups
- Normalize metrics for zero-run units (min/max/avg should be 0).
- Return copies for `LastTestResult()` and consider copying on `RecordTest`.
- Add optional history cap (e.g., last N runs) with configuration.
- Clamp coverage percentage values to 0–100.
