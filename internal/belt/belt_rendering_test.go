package belt

import (
	"image/color"
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/tedks/CodingGame/internal/connection"
	"github.com/tedks/CodingGame/internal/testutil"
)

// requireDisplay skips the test if no display is available.
func requireDisplay(t *testing.T) {
	t.Helper()
	if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		t.Skip("Skipping: no display available (set DISPLAY or WAYLAND_DISPLAY)")
	}
}

// =============================================================================
// Rendering Tests: Actual ebiten.Image pixel verification
// These tests require a display (xvfb in CI) and verify actual rendering.
// =============================================================================

// TestBeltRendering_ActualPixels verifies that belts actually render pixels.
// This is a single entry point test because GLFW can only init once per process.
func TestBeltRendering_ActualPixels(t *testing.T) {
	requireDisplay(t)

	if testing.Short() {
		t.Skip("Skipping rendering tests in short mode")
	}

	t.Run("SingleConnection", testRenderingSingleConnection)
	t.Run("AllConnectionTypes", testRenderingAllConnectionTypes)
	t.Run("CircularOverridesColor", testRenderingCircularOverridesColor)
	t.Run("StrengthAffectsWidth", testRenderingStrengthAffectsWidth)
}

func testRenderingSingleConnection(t *testing.T) {
	r := NewRenderer()
	graph := connection.NewGraph()
	graph.AddNew("a.go", "b.go", connection.TypeImport)

	positions := map[string]TilePosition{
		"a.go": {X: 100, Y: 300, Width: 50, Height: 50},
		"b.go": {X: 600, Y: 300, Width: 50, Height: 50},
	}

	game := testutil.NewSimpleTestGame(800, 600, func(screen *ebiten.Image) {
		// Fill with black background
		screen.Fill(color.Black)
		// Draw the belt
		r.Draw(screen, graph, positions, 0, 0)
	})

	screenshot, err := testutil.RunAndCapture(game, 3, 800, 600)
	if err != nil {
		t.Fatalf("Failed to capture screenshot: %v", err)
	}

	// Verify that some non-black pixels exist between the two tile centers
	// a.go center: (125, 325), b.go center: (625, 325)
	img := screenshot.Image()
	nonBlackCount := 0
	for x := 130; x < 620; x++ {
		for y := 320; y < 330; y++ {
			r, g, b, _ := img.At(x, y).RGBA()
			if r > 0 || g > 0 || b > 0 {
				nonBlackCount++
			}
		}
	}

	if nonBlackCount == 0 {
		t.Error("No pixels were rendered for the belt connection")
	}
}

func testRenderingAllConnectionTypes(t *testing.T) {
	r := NewRenderer()

	// Create connections of all types
	graph := connection.NewGraph()
	graph.AddNew("a.go", "b.go", connection.TypeImport)
	graph.AddNew("c.go", "d.go", connection.TypeInheritance)
	graph.AddNew("e.go", "f.go", connection.TypeComposition)
	graph.AddNew("g.go", "h.go", connection.TypeCall)
	graph.AddNew("i.go", "j.go", connection.TypeUnknown)

	positions := map[string]TilePosition{
		"a.go": {X: 50, Y: 50, Width: 50, Height: 50},
		"b.go": {X: 200, Y: 50, Width: 50, Height: 50},
		"c.go": {X: 50, Y: 150, Width: 50, Height: 50},
		"d.go": {X: 200, Y: 150, Width: 50, Height: 50},
		"e.go": {X: 50, Y: 250, Width: 50, Height: 50},
		"f.go": {X: 200, Y: 250, Width: 50, Height: 50},
		"g.go": {X: 50, Y: 350, Width: 50, Height: 50},
		"h.go": {X: 200, Y: 350, Width: 50, Height: 50},
		"i.go": {X: 50, Y: 450, Width: 50, Height: 50},
		"j.go": {X: 200, Y: 450, Width: 50, Height: 50},
	}

	game := testutil.NewSimpleTestGame(400, 600, func(screen *ebiten.Image) {
		screen.Fill(color.Black)
		r.Draw(screen, graph, positions, 0, 0)
	})

	screenshot, err := testutil.RunAndCapture(game, 3, 400, 600)
	if err != nil {
		t.Fatalf("Failed to capture screenshot: %v", err)
	}

	// Verify pixels exist for each row (each connection type)
	img := screenshot.Image()
	rows := []int{75, 175, 275, 375, 475} // Y positions for each connection

	for i, rowY := range rows {
		hasPixels := false
		for x := 100; x < 200; x++ {
			for y := rowY - 5; y < rowY+5; y++ {
				r, g, b, _ := img.At(x, y).RGBA()
				if r > 0 || g > 0 || b > 0 {
					hasPixels = true
					break
				}
			}
			if hasPixels {
				break
			}
		}
		if !hasPixels {
			t.Errorf("Connection type %d at row %d did not render any pixels", i, rowY)
		}
	}
}

func testRenderingCircularOverridesColor(t *testing.T) {
	r := NewRenderer()

	// Create a circular connection
	graph := connection.NewGraph()
	conn := graph.AddNew("a.go", "b.go", connection.TypeImport)
	conn.SetCircular(true)

	positions := map[string]TilePosition{
		"a.go": {X: 100, Y: 100, Width: 50, Height: 50},
		"b.go": {X: 300, Y: 100, Width: 50, Height: 50},
	}

	game := testutil.NewSimpleTestGame(500, 300, func(screen *ebiten.Image) {
		screen.Fill(color.Black)
		r.Draw(screen, graph, positions, 0, 0)
	})

	screenshot, err := testutil.RunAndCapture(game, 3, 500, 300)
	if err != nil {
		t.Fatalf("Failed to capture screenshot: %v", err)
	}

	// Circular connections should be red (circularColor = RGBA{255, 80, 80, 255})
	// Look for red-ish pixels in the belt area
	img := screenshot.Image()
	redPixelCount := 0
	for x := 150; x < 300; x++ {
		for y := 120; y < 135; y++ {
			red, green, _, _ := img.At(x, y).RGBA()
			// Red channel should be significantly higher than green for red color
			// RGBA values are pre-multiplied and scaled to 0-65535
			if red > 20000 && red > green*2 {
				redPixelCount++
			}
		}
	}

	if redPixelCount == 0 {
		t.Error("Circular connection did not render with red color")
	}
}

func testRenderingStrengthAffectsWidth(t *testing.T) {
	// Create two renderers with different strength connections
	r := NewRenderer()

	// Weak connection (strength 1)
	graphWeak := connection.NewGraph()
	connWeak := graphWeak.AddNew("a.go", "b.go", connection.TypeImport)
	connWeak.SetStrength(1)

	// Strong connection (strength 10)
	graphStrong := connection.NewGraph()
	connStrong := graphStrong.AddNew("a.go", "b.go", connection.TypeImport)
	connStrong.SetStrength(10)

	positions := map[string]TilePosition{
		"a.go": {X: 100, Y: 150, Width: 50, Height: 50},
		"b.go": {X: 400, Y: 150, Width: 50, Height: 50},
	}

	// Capture weak connection
	gameWeak := testutil.NewSimpleTestGame(600, 400, func(screen *ebiten.Image) {
		screen.Fill(color.Black)
		r.Draw(screen, graphWeak, positions, 0, 0)
	})

	screenshotWeak, err := testutil.RunAndCapture(gameWeak, 3, 600, 400)
	if err != nil {
		t.Fatalf("Failed to capture weak screenshot: %v", err)
	}

	// Count non-black pixels for weak connection
	imgWeak := screenshotWeak.Image()
	weakPixelCount := 0
	for x := 150; x < 400; x++ {
		for y := 150; y < 200; y++ {
			r, g, b, _ := imgWeak.At(x, y).RGBA()
			if r > 0 || g > 0 || b > 0 {
				weakPixelCount++
			}
		}
	}

	// Capture strong connection
	gameStrong := testutil.NewSimpleTestGame(600, 400, func(screen *ebiten.Image) {
		screen.Fill(color.Black)
		r.Draw(screen, graphStrong, positions, 0, 0)
	})

	screenshotStrong, err := testutil.RunAndCapture(gameStrong, 3, 600, 400)
	if err != nil {
		t.Fatalf("Failed to capture strong screenshot: %v", err)
	}

	// Count non-black pixels for strong connection
	imgStrong := screenshotStrong.Image()
	strongPixelCount := 0
	for x := 150; x < 400; x++ {
		for y := 150; y < 200; y++ {
			r, g, b, _ := imgStrong.At(x, y).RGBA()
			if r > 0 || g > 0 || b > 0 {
				strongPixelCount++
			}
		}
	}

	// Strong connection should have more pixels due to wider belt
	if strongPixelCount <= weakPixelCount {
		t.Errorf("Strong connection (%d pixels) should be wider than weak connection (%d pixels)",
			strongPixelCount, weakPixelCount)
	}
}
