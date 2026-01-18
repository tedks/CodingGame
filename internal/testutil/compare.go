package testutil

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

// CompareResult holds the result of comparing two images.
type CompareResult struct {
	// Match is true if the images are considered equal within tolerance.
	Match bool
	// DiffCount is the number of pixels that differ.
	DiffCount int
	// TotalPixels is the total number of pixels compared.
	TotalPixels int
	// DiffPercent is the percentage of pixels that differ.
	DiffPercent float64
	// MaxDiff is the maximum color difference found.
	MaxDiff float64
	// AvgDiff is the average color difference of differing pixels.
	AvgDiff float64
}

// CompareOptions configures how images are compared.
type CompareOptions struct {
	// Tolerance is the maximum allowed color difference per channel (0-255).
	// Default is 0 (exact match required).
	Tolerance uint8
	// PercentThreshold is the maximum percentage of differing pixels allowed.
	// Default is 0 (no differing pixels allowed).
	PercentThreshold float64
}

// DefaultCompareOptions returns options for exact matching.
func DefaultCompareOptions() CompareOptions {
	return CompareOptions{
		Tolerance:        0,
		PercentThreshold: 0,
	}
}

// CompareImages compares two images and returns the result.
// Returns an error if the images have different dimensions.
func CompareImages(img1, img2 image.Image, opts CompareOptions) (*CompareResult, error) {
	bounds1 := img1.Bounds()
	bounds2 := img2.Bounds()

	if bounds1.Dx() != bounds2.Dx() || bounds1.Dy() != bounds2.Dy() {
		return nil, fmt.Errorf("image dimensions differ: %dx%d vs %dx%d",
			bounds1.Dx(), bounds1.Dy(), bounds2.Dx(), bounds2.Dy())
	}

	totalPixels := bounds1.Dx() * bounds1.Dy()
	diffCount := 0
	var totalDiff, maxDiff float64

	for y := bounds1.Min.Y; y < bounds1.Max.Y; y++ {
		for x := bounds1.Min.X; x < bounds1.Max.X; x++ {
			c1 := img1.At(x, y)
			c2 := img2.At(x, y)

			diff := colorDifference(c1, c2)
			if diff > float64(opts.Tolerance) {
				diffCount++
				totalDiff += diff
				if diff > maxDiff {
					maxDiff = diff
				}
			}
		}
	}

	diffPercent := float64(diffCount) / float64(totalPixels) * 100
	var avgDiff float64
	if diffCount > 0 {
		avgDiff = totalDiff / float64(diffCount)
	}

	match := diffPercent <= opts.PercentThreshold

	return &CompareResult{
		Match:       match,
		DiffCount:   diffCount,
		TotalPixels: totalPixels,
		DiffPercent: diffPercent,
		MaxDiff:     maxDiff,
		AvgDiff:     avgDiff,
	}, nil
}

// CompareScreenshotToFile compares a screenshot to a reference image file.
func CompareScreenshotToFile(screenshot *Screenshot, referencePath string, opts CompareOptions) (*CompareResult, error) {
	// Load reference image
	f, err := os.Open(referencePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open reference image: %w", err)
	}
	defer f.Close()

	refImg, err := png.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("failed to decode reference image: %w", err)
	}

	return CompareImages(screenshot.Image(), refImg, opts)
}

// GenerateDiff creates a visual diff image highlighting differences between two images.
// Matching pixels are shown dimmed, differing pixels are highlighted in red.
func GenerateDiff(img1, img2 image.Image) (*image.RGBA, error) {
	bounds1 := img1.Bounds()
	bounds2 := img2.Bounds()

	if bounds1.Dx() != bounds2.Dx() || bounds1.Dy() != bounds2.Dy() {
		return nil, fmt.Errorf("image dimensions differ: %dx%d vs %dx%d",
			bounds1.Dx(), bounds1.Dy(), bounds2.Dx(), bounds2.Dy())
	}

	diff := image.NewRGBA(bounds1)

	for y := bounds1.Min.Y; y < bounds1.Max.Y; y++ {
		for x := bounds1.Min.X; x < bounds1.Max.X; x++ {
			c1 := img1.At(x, y)
			c2 := img2.At(x, y)

			if colorsEqual(c1, c2) {
				// Matching pixel - show dimmed version of original
				r, g, b, a := c1.RGBA()
				diff.Set(x, y, color.RGBA{
					R: uint8(r >> 10), // Dim to 25%
					G: uint8(g >> 10),
					B: uint8(b >> 10),
					A: uint8(a >> 8),
				})
			} else {
				// Differing pixel - highlight in red
				diff.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
			}
		}
	}

	return diff, nil
}

// SaveDiff generates and saves a visual diff image to the given path.
func SaveDiff(img1, img2 image.Image, path string) error {
	diff, err := GenerateDiff(img1, img2)
	if err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create diff file: %w", err)
	}
	defer f.Close()

	return png.Encode(f, diff)
}

// colorDifference calculates the maximum channel difference between two colors.
func colorDifference(c1, c2 color.Color) float64 {
	r1, g1, b1, a1 := c1.RGBA()
	r2, g2, b2, a2 := c2.RGBA()

	// Convert from 16-bit to 8-bit range
	dr := math.Abs(float64(r1>>8) - float64(r2>>8))
	dg := math.Abs(float64(g1>>8) - float64(g2>>8))
	db := math.Abs(float64(b1>>8) - float64(b2>>8))
	da := math.Abs(float64(a1>>8) - float64(a2>>8))

	// Return maximum difference
	return math.Max(math.Max(dr, dg), math.Max(db, da))
}

// colorsEqual checks if two colors are exactly equal.
func colorsEqual(c1, c2 color.Color) bool {
	r1, g1, b1, a1 := c1.RGBA()
	r2, g2, b2, a2 := c2.RGBA()
	return r1 == r2 && g1 == g2 && b1 == b2 && a1 == a2
}
