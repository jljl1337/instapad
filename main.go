package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"math"
	"os"
	"path/filepath"
	"strings"
)

// Acceptable aspect ratio range (width / height):
//
//	3:4    → 0.75  (portrait limit)
//	1.91:1 → 1.91  (landscape limit)
const (
	minRatio = 3.0 / 4.0  // 0.75
	maxRatio = 1.91 / 1.0 // 1.91
)

func main() {
	entries, err := os.ReadDir(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading directory: %v\n", err)
		os.Exit(1)
	}

	processed, skipped, padded := 0, 0, 0

	outDir := "padded"
	if err := os.MkdirAll(outDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "error creating output directory: %v\n", err)
		os.Exit(1)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if ext != ".jpg" && ext != ".jpeg" {
			continue
		}

		base := strings.TrimSuffix(name, filepath.Ext(name))
		if strings.HasSuffix(base, "_padded") {
			continue // skip already-padded outputs
		}

		processed++
		fmt.Printf("Processing: %s\n", name)

		img, err := loadJPEG(name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ✗ load error: %v\n", err)
			continue
		}

		bounds := img.Bounds()
		w := bounds.Dx()
		h := bounds.Dy()
		ratio := float64(w) / float64(h)

		fmt.Printf("  size: %dx%d  ratio: %.4f\n", w, h, ratio)

		if ratio >= minRatio && ratio <= maxRatio {
			fmt.Printf("  ✓ ratio in range [%.4f, %.4f] — skipping\n", minRatio, maxRatio)
			skipped++
			continue
		}

		newW, newH := targetDimensions(w, h, ratio)
		fmt.Printf("  → padding to %dx%d (ratio %.4f)\n", newW, newH, float64(newW)/float64(newH))

		paddedImg := addWhitePadding(img, newW, newH)

		outName := filepath.Join(outDir, name)
		if err := saveJPEG(outName, paddedImg); err != nil {
			fmt.Fprintf(os.Stderr, "  ✗ save error: %v\n", err)
			continue
		}

		fmt.Printf("  ✓ saved: %s\n", outName)
		padded++
	}

	fmt.Printf("\ndone — checked: %d | padded: %d | already valid: %d\n",
		processed, padded, skipped)
}

// targetDimensions returns a canvas size that:
//   - contains the original w×h image without cropping
//   - has an aspect ratio within [minRatio, maxRatio]
func targetDimensions(w, h int, ratio float64) (int, int) {
	if ratio > maxRatio {
		// Too wide → add vertical padding so ratio drops to maxRatio.
		// newH = w / maxRatio
		newH := int(math.Ceil(float64(w) / maxRatio))
		return w, newH
	}
	// ratio < minRatio → too tall → add horizontal padding so ratio rises to minRatio.
	// newW = h * minRatio
	newW := int(math.Ceil(float64(h) * minRatio))
	return newW, h
}

// addWhitePadding centers img on a white newW×newH canvas.
func addWhitePadding(img image.Image, newW, newH int) image.Image {
	canvas := image.NewRGBA(image.Rect(0, 0, newW, newH))

	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	draw.Draw(canvas, canvas.Bounds(), &image.Uniform{C: white}, image.Point{}, draw.Src)

	orig := img.Bounds()
	offsetX := (newW - orig.Dx()) / 2
	offsetY := (newH - orig.Dy()) / 2
	dest := image.Rect(offsetX, offsetY, offsetX+orig.Dx(), offsetY+orig.Dy())
	draw.Draw(canvas, dest, img, orig.Min, draw.Src)

	return canvas
}

func loadJPEG(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return jpeg.Decode(f)
}

func saveJPEG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return jpeg.Encode(f, img, &jpeg.Options{Quality: 95})
}
