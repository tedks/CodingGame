// Package testutil provides testing utilities for CodingGame, including
// screenshot capture and image comparison for visual regression testing.
package testutil

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// Screenshot captures the current state of an Ebitengine image and provides
// methods to save it to disk or compare with reference images.
type Screenshot struct {
	img       *image.RGBA
	timestamp time.Time
	name      string
}

// CaptureScreen captures the current contents of an ebiten.Image.
// This is typically called from a Draw method during testing.
func CaptureScreen(screen *ebiten.Image) *Screenshot {
	bounds := screen.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Create RGBA image to hold pixel data
	rgba := image.NewRGBA(image.Rect(0, 0, width, height))

	// ReadPixels reads the pixels from the image
	// The pixels are in RGBA format, 4 bytes per pixel
	screen.ReadPixels(rgba.Pix)

	return &Screenshot{
		img:       rgba,
		timestamp: time.Now(),
		name:      "screenshot",
	}
}

// WithName sets a descriptive name for the screenshot.
func (s *Screenshot) WithName(name string) *Screenshot {
	s.name = name
	return s
}

// SaveToFile saves the screenshot to a PNG file at the given path.
// If the directory doesn't exist, it will be created.
func (s *Screenshot) SaveToFile(path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Create file
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", path, err)
	}
	defer f.Close()

	// Encode as PNG
	if err := png.Encode(f, s.img); err != nil {
		return fmt.Errorf("failed to encode PNG: %w", err)
	}

	return nil
}

// SaveWithTimestamp saves the screenshot with a timestamped filename.
// Returns the full path where the file was saved.
func (s *Screenshot) SaveWithTimestamp(dir string) (string, error) {
	filename := fmt.Sprintf("%s_%s.png",
		s.name,
		s.timestamp.Format("20060102_150405"),
	)
	path := filepath.Join(dir, filename)
	return path, s.SaveToFile(path)
}

// Image returns the underlying image.RGBA for custom processing.
func (s *Screenshot) Image() *image.RGBA {
	return s.img
}

// Bounds returns the dimensions of the screenshot.
func (s *Screenshot) Bounds() image.Rectangle {
	return s.img.Bounds()
}

// Width returns the width of the screenshot in pixels.
func (s *Screenshot) Width() int {
	return s.img.Bounds().Dx()
}

// Height returns the height of the screenshot in pixels.
func (s *Screenshot) Height() int {
	return s.img.Bounds().Dy()
}

// ScreenshotDir returns the default directory for saving test screenshots.
// Creates the directory if it doesn't exist.
func ScreenshotDir() (string, error) {
	// Use a well-known location that agents can find
	dir := filepath.Join(os.TempDir(), "codinggame-screenshots")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create screenshot directory: %w", err)
	}
	return dir, nil
}

// DefaultScreenshotPath returns a path for saving a screenshot with the given name.
// The path is in the default screenshot directory with a timestamped filename.
func DefaultScreenshotPath(name string) (string, error) {
	dir, err := ScreenshotDir()
	if err != nil {
		return "", err
	}
	filename := fmt.Sprintf("%s_%s.png",
		name,
		time.Now().Format("20060102_150405"),
	)
	return filepath.Join(dir, filename), nil
}
