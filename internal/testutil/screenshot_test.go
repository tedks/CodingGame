package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

// Note: Tests that require the Ebitengine game loop (CaptureScreen, etc.)
// are in runner_test.go and use the RunAndCapture helper.

func TestScreenshotDir(t *testing.T) {
	dir, err := ScreenshotDir()
	if err != nil {
		t.Fatalf("ScreenshotDir() error: %v", err)
	}

	// Check directory exists
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("Screenshot directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Errorf("ScreenshotDir() returned non-directory: %s", dir)
	}
}

func TestDefaultScreenshotPath(t *testing.T) {
	path, err := DefaultScreenshotPath("test")
	if err != nil {
		t.Fatalf("DefaultScreenshotPath() error: %v", err)
	}

	// Check path is in the screenshot directory
	dir, err := ScreenshotDir()
	if err != nil {
		t.Fatalf("ScreenshotDir() error: %v", err)
	}

	if filepath.Dir(path) != dir {
		t.Errorf("DefaultScreenshotPath() = %s, want directory %s", path, dir)
	}

	// Check filename contains the name
	filename := filepath.Base(path)
	if len(filename) < 4 || filename[:4] != "test" {
		t.Errorf("DefaultScreenshotPath() filename %s doesn't start with 'test'", filename)
	}

	// Check it ends with .png
	if filepath.Ext(path) != ".png" {
		t.Errorf("DefaultScreenshotPath() = %s, want .png extension", path)
	}
}
