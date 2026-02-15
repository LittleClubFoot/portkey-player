package idle

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateScreen(t *testing.T) {
	dir := t.TempDir()
	outputPath := filepath.Join(dir, "idle.png")

	if err := GenerateScreen(outputPath); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the file exists and is a valid PNG.
	f, err := os.Open(outputPath)
	if err != nil {
		t.Fatalf("failed to open generated file: %v", err)
	}
	defer f.Close()

	img, err := png.Decode(f)
	if err != nil {
		t.Fatalf("failed to decode PNG: %v", err)
	}

	bounds := img.Bounds()
	if bounds.Dx() != screenWidth || bounds.Dy() != screenHeight {
		t.Errorf("expected %dx%d, got %dx%d", screenWidth, screenHeight, bounds.Dx(), bounds.Dy())
	}
}

func TestGenerateScreen_InvalidPath(t *testing.T) {
	err := GenerateScreen("/nonexistent/dir/idle.png")
	if err == nil {
		t.Fatal("expected error for invalid path")
	}
}
