# internal/claude summary

## Overview
- Intercepts Claude tool output (JSON lines), converts them to typed events, and dispatches events to handlers; currently mostly stubbed.

## Strengths
- Clear event typing and helper simulation functions for tests.
- Tests cover handler wiring and event type inference.

## Code smells / risks (prioritized)
- **Unsafe channel usage**: `SimulateFile*` and `parseOutput` send on `i.events` without checking `running`; if `Stop()` closes the channel, these can panic.
- **Documentation mismatch**: package comment claims handlers run in separate goroutines, but `dispatchEvents` calls handlers synchronously.
- **No lifecycle coordination**: `Start()` spawns `dispatchEvents` without a stop signal beyond closing `events`; `parseOutput` has no way to stop or to avoid blocking sends if channel fills.
- **Unused event types**: `EventToolResult` defined but never produced.

## Abstraction tightening opportunities
- Introduce a lifecycle/state machine (`Start`/`Stop`) with `sync.Once` for channel close and a `done` channel to stop goroutines cleanly.
- Add a non-blocking event enqueue API that returns an error if the interceptor is stopped.
- Decide whether handlers should be invoked synchronously (ordered) or async (concurrent) and align docs and tests accordingly.

## Testing gaps and suggested tests
- Add tests for `Stop()` + `SimulateFileRead()` to ensure no panic and predictable behavior.
- Add tests for backpressure (events channel full) to define expected behavior.
- Add tests verifying `EventToolResult` handling once supported.

## Documentation gaps
- Document handler execution semantics (sync vs async) and ordering guarantees.
- Document how to stop `parseOutput` cleanly when the subprocess ends.

## Refactor candidates
- **Small**: Update package comment or dispatch semantics to match.
- **Medium**: Add `done` channel + `sync.Once` around close; guard sends when stopped.
- **Large**: Implement actual Claude subprocess integration with context-driven shutdown and backpressure handling.

## Dependencies to review further
- `internal/harness` (if/when integrating real Claude subprocess events)
