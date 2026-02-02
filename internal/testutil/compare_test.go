package testutil

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"
)

func TestCompareImagesIdentical(t *testing.T) {
	// Create two identical images
	img1 := image.NewRGBA(image.Rect(0, 0, 10, 10))
	img2 := image.NewRGBA(image.Rect(0, 0, 10, 10))

	// Fill with same color
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			c := color.RGBA{R: 100, G: 150, B: 200, A: 255}
			img1.Set(x, y, c)
			img2.Set(x, y, c)
		}
	}

	result, err := CompareImages(img1, img2, DefaultCompareOptions())
	if err != nil {
		t.Fatalf("CompareImages() error: %v", err)
	}

	if !result.Match {
		t.Errorf("CompareImages() Match = false, want true for identical images")
	}
	if result.DiffCount != 0 {
		t.Errorf("CompareImages() DiffCount = %d, want 0", result.DiffCount)
	}
	if result.DiffPercent != 0 {
		t.Errorf("CompareImages() DiffPercent = %f, want 0", result.DiffPercent)
	}
}

func TestCompareImagesDifferent(t *testing.T) {
	img1 := image.NewRGBA(image.Rect(0, 0, 10, 10))
	img2 := image.NewRGBA(image.Rect(0, 0, 10, 10))

	// Fill img1 with red
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			img1.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}

	// Fill img2 with blue
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			img2.Set(x, y, color.RGBA{R: 0, G: 0, B: 255, A: 255})
		}
	}

	result, err := CompareImages(img1, img2, DefaultCompareOptions())
	if err != nil {
		t.Fatalf("CompareImages() error: %v", err)
	}

	if result.Match {
		t.Errorf("CompareImages() Match = true, want false for different images")
	}
	if result.DiffCount != 100 {
		t.Errorf("CompareImages() DiffCount = %d, want 100", result.DiffCount)
	}
	if result.DiffPercent != 100.0 {
		t.Errorf("CompareImages() DiffPercent = %f, want 100.0", result.DiffPercent)
	}
}

func TestCompareImagesWithTolerance(t *testing.T) {
	img1 := image.NewRGBA(image.Rect(0, 0, 10, 10))
	img2 := image.NewRGBA(image.Rect(0, 0, 10, 10))

	// Fill with colors that differ by 5 in red channel
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			img1.Set(x, y, color.RGBA{R: 100, G: 100, B: 100, A: 255})
			img2.Set(x, y, color.RGBA{R: 105, G: 100, B: 100, A: 255})
		}
	}

	// With tolerance 0, should not match
	result, err := CompareImages(img1, img2, DefaultCompareOptions())
	if err != nil {
		t.Fatalf("CompareImages() error: %v", err)
	}
	if result.Match {
		t.Error("CompareImages() with tolerance 0 should not match images differing by 5")
	}

	// With tolerance 10, should match
	opts := CompareOptions{Tolerance: 10}
	result, err = CompareImages(img1, img2, opts)
	if err != nil {
		t.Fatalf("CompareImages() error: %v", err)
	}
	if !result.Match {
		t.Error("CompareImages() with tolerance 10 should match images differing by 5")
	}
}

func TestCompareImagesDimensionMismatch(t *testing.T) {
	img1 := image.NewRGBA(image.Rect(0, 0, 10, 10))
	img2 := image.NewRGBA(image.Rect(0, 0, 20, 20))

	_, err := CompareImages(img1, img2, DefaultCompareOptions())
	if err == nil {
		t.Error("CompareImages() should error on dimension mismatch")
	}
}

func TestGenerateDiff(t *testing.T) {
	img1 := image.NewRGBA(image.Rect(0, 0, 10, 10))
	img2 := image.NewRGBA(image.Rect(0, 0, 10, 10))

	// Make half the pixels match, half differ
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			img1.Set(x, y, color.RGBA{R: 100, G: 100, B: 100, A: 255})
			if x < 5 {
				img2.Set(x, y, color.RGBA{R: 100, G: 100, B: 100, A: 255}) // Match
			} else {
				img2.Set(x, y, color.RGBA{R: 200, G: 200, B: 200, A: 255}) // Differ
			}
		}
	}

	diff, err := GenerateDiff(img1, img2)
	if err != nil {
		t.Fatalf("GenerateDiff() error: %v", err)
	}

	// Check that differing pixels are red
	for y := 0; y < 10; y++ {
		for x := 5; x < 10; x++ {
			c := diff.At(x, y).(color.RGBA)
			if c.R != 255 || c.G != 0 || c.B != 0 {
				t.Errorf("GenerateDiff() pixel (%d,%d) = %v, want red", x, y, c)
			}
		}
	}
}

func TestSaveDiff(t *testing.T) {
	img1 := image.NewRGBA(image.Rect(0, 0, 10, 10))
	img2 := image.NewRGBA(image.Rect(0, 0, 10, 10))

	// Fill with different colors
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			img1.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
			img2.Set(x, y, color.RGBA{R: 0, G: 255, B: 0, A: 255})
		}
	}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "diff.png")

	err := SaveDiff(img1, img2, path)
	if err != nil {
		t.Fatalf("SaveDiff() error: %v", err)
	}

	// Check file exists
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Diff file not found: %v", err)
	}
	if info.Size() == 0 {
		t.Error("Diff file is empty")
	}
}

// Edge case tests for image comparison

func TestCompareImages_TransparentPixels(t *testing.T) {
	img1 := image.NewRGBA(image.Rect(0, 0, 10, 10))
	img2 := image.NewRGBA(image.Rect(0, 0, 10, 10))

	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			img1.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 0})
			img2.Set(x, y, color.RGBA{R: 0, G: 255, B: 0, A: 0})
		}
	}

	result, err := CompareImages(img1, img2, DefaultCompareOptions())
	if err != nil {
		t.Fatalf("CompareImages() error: %v", err)
	}

	if result.Match {
		t.Error("Transparent pixels with different RGB should NOT match")
	}
}

func TestCompareImages_NonZeroOriginBounds(t *testing.T) {
	img1 := image.NewRGBA(image.Rect(10, 10, 20, 20))
	img2 := image.NewRGBA(image.Rect(10, 10, 20, 20))

	for y := 10; y < 20; y++ {
		for x := 10; x < 20; x++ {
			c := color.RGBA{R: 128, G: 128, B: 128, A: 255}
			img1.Set(x, y, c)
			img2.Set(x, y, c)
		}
	}

	result, err := CompareImages(img1, img2, DefaultCompareOptions())
	if err != nil {
		t.Fatalf("CompareImages() error: %v", err)
	}

	if !result.Match {
		t.Error("Identical images with non-zero origin should match")
	}
	if result.TotalPixels != 100 {
		t.Errorf("Expected 100 total pixels, got %d", result.TotalPixels)
	}
}

func TestCompareImages_DifferentImageTypes(t *testing.T) {
	rgba := image.NewRGBA(image.Rect(0, 0, 10, 10))
	nrgba := image.NewNRGBA(image.Rect(0, 0, 10, 10))

	c := color.RGBA{R: 100, G: 150, B: 200, A: 255}
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			rgba.Set(x, y, c)
			nrgba.Set(x, y, c)
		}
	}

	result, err := CompareImages(rgba, nrgba, DefaultCompareOptions())
	if err != nil {
		t.Fatalf("CompareImages() error: %v", err)
	}

	if !result.Match {
		t.Error("RGBA and NRGBA images with same color should match")
	}
}

func TestColorDifference_KnownDeltas(t *testing.T) {
	cases := []struct {
		name     string
		c1, c2   color.Color
		expected float64
	}{
		{"black to white", color.RGBA{0, 0, 0, 255}, color.RGBA{255, 255, 255, 255}, 255},
		{"identical", color.RGBA{100, 100, 100, 255}, color.RGBA{100, 100, 100, 255}, 0},
		{"red only", color.RGBA{0, 100, 100, 255}, color.RGBA{50, 100, 100, 255}, 50},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			diff := colorDifference(tc.c1, tc.c2)
			if diff != tc.expected {
				t.Errorf("colorDifference() = %v, want %v", diff, tc.expected)
			}
		})
	}
}

func TestGenerateDiff_DimmingCorrectness(t *testing.T) {
	img1 := image.NewRGBA(image.Rect(0, 0, 10, 10))
	img2 := image.NewRGBA(image.Rect(0, 0, 10, 10))

	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			c := color.RGBA{R: 255, G: 255, B: 255, A: 255}
			img1.Set(x, y, c)
			img2.Set(x, y, c)
		}
	}

	diff, err := GenerateDiff(img1, img2)
	if err != nil {
		t.Fatalf("GenerateDiff() error: %v", err)
	}

	// 255*257 >> 10 = 63
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			c := diff.At(x, y).(color.RGBA)
			if c.R != 63 || c.G != 63 || c.B != 63 {
				t.Errorf("Pixel (%d,%d) = (%d,%d,%d), expected (63,63,63)", x, y, c.R, c.G, c.B)
			}
		}
	}
}

func TestGenerateDiff_DiffPixelsAreRed(t *testing.T) {
	img1 := image.NewRGBA(image.Rect(0, 0, 5, 5))
	img2 := image.NewRGBA(image.Rect(0, 0, 5, 5))

	for y := 0; y < 5; y++ {
		for x := 0; x < 5; x++ {
			img1.Set(x, y, color.RGBA{R: 0, G: 0, B: 0, A: 255})
			img2.Set(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}

	diff, err := GenerateDiff(img1, img2)
	if err != nil {
		t.Fatalf("GenerateDiff() error: %v", err)
	}

	for y := 0; y < 5; y++ {
		for x := 0; x < 5; x++ {
			c := diff.At(x, y).(color.RGBA)
			if c.R != 255 || c.G != 0 || c.B != 0 || c.A != 255 {
				t.Errorf("Diff pixel (%d,%d) = (%d,%d,%d,%d), want red", x, y, c.R, c.G, c.B, c.A)
			}
		}
	}
}
