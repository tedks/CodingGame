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

// TestRaceDetectorMustBeEnabled FAILS if race detection is not enabled.
// This test is used by CI to verify that the race detection infrastructure
// is actually working. If this test passes, we know the race detector is on.
//
// This test should only be run with --@io_bazel_rules_go//go/config:race.
// Running it without -race will cause a test failure, which is intentional.
func TestRaceDetectorMustBeEnabled(t *testing.T) {
	if !raceDetectorEnabled {
		t.Fatal("RACE DETECTION IS NOT ENABLED - CI infrastructure failure!\n" +
			"This test verifies that bazel is actually running with race detection.\n" +
			"If you see this error in CI, the --@io_bazel_rules_go//go/config:race flag is not working.\n" +
			"If you see this locally, run with: bazel test --@io_bazel_rules_go//go/config:race")
	}

	t.Log("Race detector verification passed - race detection is enabled")
}
