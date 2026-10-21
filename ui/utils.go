package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"image/color"
	"math"
	"strconv"
	"strings"
	yatheme "yaterm/theme"
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
	size := fyne.MeasureText("M", terminalTextSize(), fyne.TextStyle{Monospace: true})
	size.Height = float32(math.Round(float64(size.Height)))
	return size
}

// terminalTextSize is the size of the terminal's text from the YaTerm theme,
// or the normal text size with another theme
func terminalTextSize() float32 {
	if size := theme.Size(yatheme.SizeNameTerminalText); size > 0 {
		return size
	}
	return theme.TextSize()
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

// tmuxQuote quotes text for a tmux command, in double quotes \, " and $ need
// a backslash
func tmuxQuote(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, `$`, `\$`)
	return `"` + r.Replace(s) + `"`
}

func isOctal(c byte) bool {
	return c >= '0' && c <= '7'
}

// decodeTmux undoes the escapes of tmux: "\ooo" (octal) is a byte, "\\" a
// backslash. %output escapes the control characters and the backslash as
// octal, capture-pane -C escapes the backslash as "\\".
func decodeTmux(s string) []byte {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c != '\\' || i+1 >= len(s) {
			out = append(out, c)
			continue
		}
		if s[i+1] == '\\' {
			out = append(out, '\\')
			i++
			continue
		}
		if i+3 < len(s) && isOctal(s[i+1]) && isOctal(s[i+2]) && isOctal(s[i+3]) {
			out = append(out, (s[i+1]-'0')<<6|(s[i+2]-'0')<<3|(s[i+3]-'0'))
			i += 3
			continue
		}
		out = append(out, c)
	}
	return out
}

// tmuxID reads an id like "%3" or "@1" or "$2"
func tmuxID(s string, prefix byte) (int, bool) {
	if len(s) < 2 || s[0] != prefix {
		return 0, false
	}
	n, err := strconv.Atoi(s[1:])
	return n, err == nil
}

// queryThemeColor answers OSC 10, 11 and 12 with the theme's text,
// background and cursor color
func queryThemeColor() [3]color.Color {
	return [3]color.Color{theme.Color(theme.ColorNameForeground), theme.Color(theme.ColorNameBackground), theme.Color(theme.ColorNamePrimary)}
}

// currentTheme is the app's YatermTheme, nil when the app uses another theme
// (e.g. in tests, or inside a container.ThemeOverride)
func currentTheme() *yatheme.YatermTheme {
	th, _ := theme.Current().(*yatheme.YatermTheme)
	return th
}
