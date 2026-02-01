# internal/build summary

## Overview
- Provides a common build adapter interface with concrete implementations for Bazel, Cargo, and npm, exposing real build metrics.

## Strengths
- Clean adapter abstraction and registry with good unit coverage.
- Adapters use timeouts and capture stdout/stderr for diagnostics.
- Tests include many parsing scenarios and edge cases.

## Code smells / risks (prioritized)
- **Integration tests not wired**: `internal/build/integration_test.go` exists but is not listed in `internal/build/BUILD.bazel`, so it never runs under Bazel.
- **Cargo metadata parsing is fragile**: `parseMetadataOutput` uses regex on raw JSON and can pair unrelated `name` and `kind` lines (e.g., package name, dependency name) leading to incorrect targets.
- **`BuildOptions.Verbose` unused** across all adapters (API drift).
- **`CargoAdapter.Build` ignores `targetID`** entirely and always builds the default target/workspace.
- **`check*Installed` uses no timeout** (`bazel version`, `cargo --version`, `npm --version`) and may hang in misconfigured environments.

## Abstraction tightening opportunities
- Introduce a shared helper for running external commands with timeouts and capturing output (reduce duplication and unify error handling).
- Make `BuildOptions` a struct with documented support per adapter (or split into per-adapter options) and validate unsupported options (e.g., npm `Jobs`).
- Use structured JSON parsing for `cargo metadata` to extract targets deterministically.

## Testing gaps and suggested tests
- Add Bazel test target entry for `integration_test.go` or split integration tests into their own `go_test` target.
- Add tests for Cargo JSON parsing using real `cargo metadata` output fixtures (with multiple packages/targets).
- Add tests asserting `BuildOptions.Verbose` behavior once implemented (e.g., capture/log). 

## Documentation gaps
- Document which `BuildOptions` fields are respected per adapter.
- Document that `CargoAdapter.Build` currently ignores `targetID` (or fix it).

## Refactor candidates
- **Small**: Wire `integration_test.go` into `BUILD.bazel`.
- **Medium**: Implement structured JSON parsing for Cargo metadata and align `targetID` handling.
- **Medium**: Add command execution helper with timeouts for version checks and builds.
- **Large**: Add dependency extraction for Bazel targets (e.g., `bazel query --output=graph`) to populate `Target.Dependencies`.

## Dependencies to review further
- External command usage in adapters (potential for centralized runner or mocked execution)
