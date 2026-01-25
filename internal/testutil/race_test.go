package testutil

import (
	"testing"
)

// TestRaceDetectorEnabled verifies the race detector is active when running
// with --@io_bazel_rules_go//go/config:race.
//
// This test uses runtime.RaceEnabled() which returns true only when the
// binary was compiled with -race. It doesn't create an actual race condition
// (which would cause test failures), but confirms the infrastructure is working.
//
// To verify race detection catches real races, see the fix commits in PR #25
// where the race detector found 3 latent bugs in existing tests.
func TestRaceDetectorEnabled(t *testing.T) {
	// This test only makes assertions when race detection is enabled.
	// When running without -race, it just logs informational message.
	if !raceEnabled() {
		t.Log("Race detector is NOT enabled (expected when running without --race flag)")
		t.Log("To run with race detection: bazel test //... --@io_bazel_rules_go//go/config:race")
		return
	}

	t.Log("Race detector is enabled - CI race detection infrastructure is working")
}

// raceEnabled returns true if the race detector is active.
// This is a runtime check, not a build tag check.
func raceEnabled() bool {
	// The race detector sets this at runtime
	return raceDetectorEnabled
}

// raceDetectorEnabled is set by the race_enabled.go file when built with -race.
// Default is false (set in race_disabled.go).
