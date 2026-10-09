package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"image/color"
	"math"
)

/*
 * ================================
 *  Utility functions
 * ================================
 */
func measureCell() fyne.Size {
	size := fyne.MeasureText("M", theme.TextSize(), fyne.TextStyle{Monospace: true})
	size.Width = float32(math.Round(float64(size.Width)))
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
