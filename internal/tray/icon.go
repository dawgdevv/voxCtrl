package tray

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
)

// ── Anti-aliased drawing primitives ──────────────────────────────────────────
//
// All primitives use sub-pixel anti-aliasing via distance-to-edge blending.
// This keeps edges smooth at 64×64 without needing a full rasterization lib.

// blend alpha-composites colour c at coverage alpha onto the existing pixel.
func blend(img *image.RGBA, x, y int, c color.RGBA, alpha float64) {
	if x < 0 || y < 0 || x >= img.Bounds().Max.X || y >= img.Bounds().Max.Y {
		return
	}
	if alpha <= 0 {
		return
	}
	if alpha > 1 {
		alpha = 1
	}
	src := img.RGBAAt(x, y)
	srcA := float64(c.A) / 255.0 * alpha
	dstA := 1.0 - srcA
	img.SetRGBA(x, y, color.RGBA{
		R: uint8(float64(c.R)*srcA + float64(src.R)*dstA),
		G: uint8(float64(c.G)*srcA + float64(src.G)*dstA),
		B: uint8(float64(c.B)*srcA + float64(src.B)*dstA),
		A: uint8(math.Min(float64(src.A)+float64(c.A)*srcA, 255)),
	})
}

// fillRect draws an anti-aliased filled rounded rectangle.
// x0,y0 = top-left corner; x1,y1 = bottom-right corner; r = corner radius.
func fillRect(img *image.RGBA, x0, y0, x1, y1, r float64, c color.RGBA) {
	ix0, iy0 := int(x0-1), int(y0-1)
	ix1, iy1 := int(x1+1), int(y1+1)
	for py := iy0; py <= iy1; py++ {
		for px := ix0; px <= ix1; px++ {
			fx, fy := float64(px), float64(py)
			// nearest point inside the rounded rect's "spine"
			nx := clamp(fx, x0+r, x1-r)
			ny := clamp(fy, y0+r, y1-r)
			dx, dy := fx-nx, fy-ny
			dist := math.Sqrt(dx*dx+dy*dy) - r
			if dist < 0 {
				alpha := clamp(-dist, 0, 1)
				blend(img, px, py, c, alpha)
			}
		}
	}
}

// strokeLine draws an anti-aliased line with given half-thickness.
func strokeLine(img *image.RGBA, x0, y0, x1, y1, hw float64, c color.RGBA) {
	dx, dy := x1-x0, y1-y0
	length := math.Sqrt(dx*dx + dy*dy)
	if length < 1e-9 {
		return
	}
	ix0 := int(math.Min(x0, x1) - hw - 1)
	ix1 := int(math.Max(x0, x1) + hw + 1)
	iy0 := int(math.Min(y0, y1) - hw - 1)
	iy1 := int(math.Max(y0, y1) + hw + 1)
	for py := iy0; py <= iy1; py++ {
		for px := ix0; px <= ix1; px++ {
			fx, fy := float64(px), float64(py)
			t := clamp(((fx-x0)*dx+(fy-y0)*dy)/(length*length), 0, 1)
			nearX := x0 + t*dx
			nearY := y0 + t*dy
			ddx, ddy := fx-nearX, fy-nearY
			dist := math.Sqrt(ddx*ddx+ddy*ddy) - hw
			if dist < 0 {
				blend(img, px, py, c, clamp(-dist, 0, 1))
			}
		}
	}
}

// strokeArc draws an anti-aliased arc. Angles in radians, hw = half-thickness.
func strokeArc(img *image.RGBA, cx, cy, r, hw, a0, a1 float64, c color.RGBA) {
	// walk the arc in small angular steps, draw short line segments
	arcLen := math.Abs(a1-a0) * r
	steps := int(arcLen*2) + 8
	for i := 0; i < steps; i++ {
		t0 := a0 + (a1-a0)*float64(i)/float64(steps)
		t1 := a0 + (a1-a0)*float64(i+1)/float64(steps)
		strokeLine(img,
			cx+r*math.Cos(t0), cy+r*math.Sin(t0),
			cx+r*math.Cos(t1), cy+r*math.Sin(t1),
			hw, c)
	}
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// ── Microphone renderer ───────────────────────────────────────────────────────
//
// The icon is a minimal microphone:
//
//   ┌───┐   ← capsule body (rounded rect)
//   │   │
//   │   │
//   └───┘
//  (     )  ← pickup arc (open semicircle)
//     │     ← stand (vertical line)
//   ─────   ← base (horizontal line)
//
// Four colour variants: idle (grey), active (white), success (green), error (red).

func renderMicIcon(size int, bg, fg, accent color.RGBA) []byte {
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	// transparent base
	draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{0, 0, 0, 0}}, image.Point{}, draw.Src)

	s := float64(size)

	// ── background: dark pill ────────────────────────────────────────────────
	pad := s * 0.07
	fillRect(img, pad, pad, s-pad, s-pad, s*0.20, bg)

	// ── mic body (capsule) ───────────────────────────────────────────────────
	// sits in the upper 55% of the icon, centred horizontally
	cx := s * 0.50
	bw := s * 0.185 // half-width  (full width = 37% of size)
	bt := s * 0.175 // top of capsule
	bb := s * 0.555 // bottom of capsule
	br := bw        // radius = half-width → perfect pill

	fillRect(img, cx-bw, bt, cx+bw, bb, br, fg)

	// ── subtle inner shine on capsule ────────────────────────────────────────
	// a small lighter lozenge in the top-left quadrant of the capsule
	shine := color.RGBA{255, 255, 255, 38}
	fillRect(img, cx-bw*0.55, bt+s*0.03, cx-bw*0.05, bt+s*0.22, bw*0.45, shine)

	// ── pickup arc ───────────────────────────────────────────────────────────
	// semicircle curving downward from the bottom of the capsule
	arcCY := bb + s*0.005 // arc pivot: just below capsule bottom
	arcR := bw * 1.62     // arc radius slightly wider than capsule
	arcHW := s * 0.058    // stroke half-thickness
	strokeArc(img, cx, arcCY, arcR, arcHW,
		math.Pi*0.07, math.Pi*0.93, // slightly open arc (not full 180°)
		accent)

	// ── stand: vertical line from arc bottom down ────────────────────────────
	standTop := arcCY + arcR + arcHW*0.5
	standBot := s * 0.845
	standHW := s * 0.055
	strokeLine(img, cx, standTop, cx, standBot, standHW, accent)

	// ── base: horizontal foot ────────────────────────────────────────────────
	baseHalfW := s * 0.275
	strokeLine(img, cx-baseHalfW, standBot, cx+baseHalfW, standBot, standHW, accent)

	// ── round caps on base ends (small filled circles) ───────────────────────
	capR := standHW + 0.5
	fillRect(img, cx-baseHalfW-capR, standBot-capR, cx-baseHalfW+capR, standBot+capR, capR, accent)
	fillRect(img, cx+baseHalfW-capR, standBot-capR, cx+baseHalfW+capR, standBot+capR, capR, accent)

	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

// ── Four icon variants ────────────────────────────────────────────────────────
// Each returns a PNG-encoded []byte ready to pass to systray.SetIcon().
// All render at `size` pixels square; 64 is the recommended size.

// generateIdleIcon — charcoal background, muted slate mic.
// Default state: daemon is running, waiting for hotkey.
func generateIdleIcon(size int) []byte {
	return renderMicIcon(size,
		color.RGBA{24, 24, 32, 245},    // dark charcoal bg
		color.RGBA{155, 158, 175, 255}, // muted slate mic body
		color.RGBA{110, 113, 130, 255}, // darker slate arc/stand/base
	)
}

// generateActiveIcon — near-black bg, bright white mic.
// State: hotkey held, recording in progress.
func generateActiveIcon(size int) []byte {
	return renderMicIcon(size,
		color.RGBA{14, 14, 20, 245},    // deep bg
		color.RGBA{235, 238, 255, 255}, // bright white mic body
		color.RGBA{190, 195, 220, 255}, // cool-white arc/stand
	)
}

// generateSuccessIcon — dark forest bg, vivid green mic.
// State: command matched and executed successfully.
func generateSuccessIcon(size int) []byte {
	return renderMicIcon(size,
		color.RGBA{10, 26, 14, 245},   // dark forest bg
		color.RGBA{72, 199, 100, 255}, // vivid green mic body
		color.RGBA{48, 160, 74, 255},  // deeper green arc/stand
	)
}

// generateErrorIcon — dark crimson bg, red mic.
// State: transcription failed, no command matched, or execution error.
func generateErrorIcon(size int) []byte {
	return renderMicIcon(size,
		color.RGBA{28, 10, 12, 245},  // dark crimson bg
		color.RGBA{220, 68, 72, 255}, // vivid red mic body
		color.RGBA{178, 44, 50, 255}, // deeper red arc/stand
	)
}
