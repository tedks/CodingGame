package testutil

import (
	"testing"
)

// TestRaceDetectorMustBeEnabled FAILS if race detection is not enabled.
// This test is used by CI to verify that the race detection infrastructure
// is actually working. If this test passes, we know the race detector is on.
//
// This test is in a separate file so it can be in a separate Bazel test target
// with tags that exclude it from regular `bazel test //...` runs. It should
// only be run explicitly with --@io_bazel_rules_go//go/config:race.
//
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
