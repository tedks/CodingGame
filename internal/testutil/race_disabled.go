//go:build !race

package testutil

// raceDetectorEnabled is false when built without -race flag.
const raceDetectorEnabled = false
