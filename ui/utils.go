package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"image/color"
	"math"
	"strings"
)

/*
 * ================================
 *  Utility functions
 * ================================
 */
// measureCell keeps the font's real advance width: a run of text is drawn as
// one canvas.Text, so its characters sit at the font's spacing, and a rounded
// width would let the cursor drift from the text (8.406 vs 8 is a whole cell
// off by column 25). Only the height is rounded so rows stack without seams.
func measureCell() fyne.Size {
	size := fyne.MeasureText("M", theme.TextSize(), fyne.TextStyle{Monospace: true})
	size.Height = float32(math.Round(float64(size.Height)))
	return size
}

// blend mixes a and b half and half, used to draw faint text
func blend(a, b color.Color) color.Color {
	ar, ag, ab, aa := a.RGBA()
	br, bg, bb, ba := b.RGBA()

	return color.RGBA64{
		R: uint16((ar + br) / 2),
		G: uint16((ag + bg) / 2),
		B: uint16((ab + bb) / 2),
		A: uint16((aa + ba) / 2),
	}
}

// abs returns the absolute value of an int, math.Abs is only for float64
func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func isWordRune(r rune) bool {
	return r != 0 && !strings.ContainsRune(wordDelimiters, r)
}

// over paints c on top of an opaque bg and returns the opaque result
func over(c, bg color.Color) color.Color {
	cr, cg, cb, ca := c.RGBA()
	br, bgg, bb, _ := bg.RGBA()
	mix := func(fg, back uint32) uint8 {
		// RGBA() is premultiplied, so fg already has the alpha applied
		return uint8((fg + back*(0xffff-ca)/0xffff) >> 8)
	}
	return color.RGBA{R: mix(cr, br), G: mix(cg, bgg), B: mix(cb, bb), A: 0xff}
}
