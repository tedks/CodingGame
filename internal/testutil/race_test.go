package testutil

import (
	"testing"
)

// TestRaceDetectorEnabled is an informational test that logs whether
// race detection is enabled. It always passes.
//
// To verify race detection catches real races, see the fix commits in PR #25
// where the race detector found 3 latent bugs in existing tests.
func TestRaceDetectorEnabled(t *testing.T) {
	if !raceDetectorEnabled {
		t.Log("Race detector is NOT enabled (expected when running without --race flag)")
		t.Log("To run with race detection: bazel test //... --@io_bazel_rules_go//go/config:race")
		return
	}

	t.Log("Race detector is enabled - CI race detection infrastructure is working")
}
