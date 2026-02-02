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

// Path sanitization edge cases

func TestSanitizeName_EdgeCases(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"empty string", "", ""},
		{"already safe", "TestMyFunction", "TestMyFunction"},
		{"underscores and hyphens", "test_my-function", "test_my-function"},
		{"numbers", "test123", "test123"},
		{"path separators", "internal/testutil/test", "internal_testutil_test"},
		{"spaces", "test with spaces", "test_with_spaces"},
		{"special chars", "test@#$%", "test____"},
		{"dots", "test.file.name", "test_file_name"},
		{"colons", "Test::Subtest/case", "Test__Subtest_case"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := sanitizeName(tc.input)
			if got != tc.want {
				t.Errorf("sanitizeName(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestSanitizeName_LongString(t *testing.T) {
	input := ""
	for i := 0; i < 200; i++ {
		input += "abcde"
	}
	result := sanitizeName(input)
	if len(result) != 1000 {
		t.Errorf("Expected 1000 character result, got %d", len(result))
	}
}

func TestSanitizeName_BytePreservation(t *testing.T) {
	allSafe := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-"
	result := sanitizeName(allSafe)
	if result != allSafe {
		t.Errorf("All safe characters should be preserved, got %q", result)
	}
}

func TestSanitizeName_NonASCIIReplacement(t *testing.T) {
	input := "café"
	result := sanitizeName(input)
	// 'c', 'a', 'f' are safe, 'é' (2 bytes) becomes '__'
	if result != "caf__" {
		t.Errorf("sanitizeName(%q) = %q, expected 'caf__'", input, result)
	}
}
