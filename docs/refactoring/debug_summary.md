# internal/debug summary

## Overview
- Defines language-agnostic debugging model (sessions, frames, variables, breakpoints, events, data flow) and a Go/delve adapter.

## Strengths
- Clear separation between core debug model and adapter implementations.
- Good coverage of core data structures and helper methods in tests.
- Data flow modeling aligns with belt visualization concept.

## Code smells / risks (prioritized)
- **Delve RPC concurrency hazards**: `callRPC` uses a shared `net.Conn` with no per-session lock; concurrent calls can interleave requests/responses and corrupt state.
- **No RPC timeouts**: `callRPC` does not set read/write deadlines; RPC calls can hang indefinitely.
- **Session mutability leaks**: `SetFrames` stores the provided slice directly; external callers can mutate it after the call.
- **EventBus not thread-safe**: no locking for subscribe/publish, which is risky if used concurrently.
- **Breakpoint index cleanup**: `RemoveBreakpoint` doesn’t delete empty `byLocation` entries (minor leak).

## Abstraction tightening opportunities
- Add a per-session mutex for RPC calls and a shared `json.Decoder`/`Encoder` to serialize requests.
- Provide a `Session` copy-on-write or defensive copy for frames and handler lists to avoid external mutation.
- Consider a common event bus interface with concurrency guarantees.

## Testing gaps and suggested tests
- Add tests for concurrent adapter calls to ensure no data races or interleaved RPC responses (mock the conn).
- Add tests for `Session.SetFrames` mutation safety once fixed.
- Add tests around breakpoint removal cleanup in `byLocation`.

## Documentation gaps
- Document concurrency expectations for adapters and session methods.
- Document that `DataFlows` are unbounded and may need pruning for long sessions.

## Refactor candidates
- **Small**: Ensure `SetFrames` copies input; clean up `byLocation` entries when empty.
- **Medium**: Add per-session RPC lock + deadlines in delve adapter.
- **Large**: Add event handling loop for breakpoint hits/output, and structured session lifecycle management.

## Dependencies to review further
- `internal/belt` (data flow visualization integration)
- `internal/systemtest` (if debugger interactions will be simulated)
