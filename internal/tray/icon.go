package tray

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
)

// generateCircleIcon creates a simple filled-circle PNG of the given colour.
func generateCircleIcon(c color.RGBA, size int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	// transparent background
	draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{0, 0, 0, 0}}, image.Point{}, draw.Src)

	cx, cy := size/2, size/2
	r := size/2 - 2
	r2 := r * r

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx, dy := x-cx, y-cy
			if dx*dx+dy*dy <= r2 {
				img.Set(x, y, c)
			}
		}
	}

	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}
