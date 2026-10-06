package terminal

import (
	"image/color"
	"strconv"
	"strings"
)

// The 16 basic colors, xterm defaults
var basicColors = [16]color.Color{
	color.RGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xff}, // black
	color.RGBA{R: 0xcd, G: 0x00, B: 0x00, A: 0xff}, // red
	color.RGBA{R: 0x00, G: 0xcd, B: 0x00, A: 0xff}, // green
	color.RGBA{R: 0xcd, G: 0xcd, B: 0x00, A: 0xff}, // yellow
	color.RGBA{R: 0x00, G: 0x00, B: 0xee, A: 0xff}, // blue
	color.RGBA{R: 0xcd, G: 0x00, B: 0xcd, A: 0xff}, // magenta
	color.RGBA{R: 0x00, G: 0xcd, B: 0xcd, A: 0xff}, // cyan
	color.RGBA{R: 0xe5, G: 0xe5, B: 0xe5, A: 0xff}, // white
	color.RGBA{R: 0x7f, G: 0x7f, B: 0x7f, A: 0xff}, // bright black
	color.RGBA{R: 0xff, G: 0x00, B: 0x00, A: 0xff}, // bright red
	color.RGBA{R: 0x00, G: 0xff, B: 0x00, A: 0xff}, // bright green
	color.RGBA{R: 0xff, G: 0xff, B: 0x00, A: 0xff}, // bright yellow
	color.RGBA{R: 0x5c, G: 0x5c, B: 0xff, A: 0xff}, // bright blue
	color.RGBA{R: 0xff, G: 0x00, B: 0xff, A: 0xff}, // bright magenta
	color.RGBA{R: 0x00, G: 0xff, B: 0xff, A: 0xff}, // bright cyan
	color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}, // bright white
}

// indexedColor returns one of the 256 xterm colors
func indexedColor(i int) color.Color {
	switch {
	case i < 0 || i > 255:
		return nil

	case i < 16:
		return basicColors[i]

	// 6x6x6 color cube
	case i < 232:
		i -= 16
		level := func(v int) uint8 {
			if v == 0 {
				return 0
			}
			return uint8(55 + v*40)
		}
		return color.RGBA{R: level(i / 36), G: level(i / 6 % 6), B: level(i % 6), A: 0xff}

	// 24 grayscale steps
	default:
		g := uint8(8 + (i-232)*10)
		return color.RGBA{R: g, G: g, B: g, A: 0xff}
	}
}

func (t *Terminal) resetGraphics() {
	t.fg = nil
	t.bg = nil
	t.style = Style{}
}

// handleColorMode handles SGR "CSI ... m", ref: https://wezterm.org/escape-sequences.html#graphic-rendition-sgr
func (t *Terminal) handleColorMode(message string) {
	if message != "" && (message[0] == '>' || message[0] == '?') {
		return
	}

	// "CSI m" is the same as "CSI 0 m"
	if message == "" {
		t.resetGraphics()
		return
	}

	modes := strings.Split(message, ";")
	for i := 0; i < len(modes); i++ {
		// "38:2::r:g:b" style, the extended color is inside one field
		if strings.Contains(modes[i], ":") {
			sub := strings.Split(modes[i], ":")
			t.handleExtendedColor(atoi(sub[0]), sub[1:], true)
			continue
		}

		mode := atoi(modes[i])
		switch {
		case mode == 0:
			t.resetGraphics()
		case mode == 1:
			t.style.Bold = true
		case mode == 2:
			t.style.Faint = true
		case mode == 3:
			t.style.Italic = true
		case mode == 4:
			t.style.Underline = true
		case mode == 5 || mode == 6:
			t.style.Blinking = true
		case mode == 7:
			t.style.Inverse = true
		case mode == 8:
			t.style.Hidden = true
		case mode == 9:
			t.style.Strikethrough = true
		case mode == 21:
			t.style.Underline = true // doubly underlined, draw as single
		case mode == 22:
			t.style.Bold = false
			t.style.Faint = false
		case mode == 23:
			t.style.Italic = false
		case mode == 24:
			t.style.Underline = false
		case mode == 25:
			t.style.Blinking = false
		case mode == 27:
			t.style.Inverse = false
		case mode == 28:
			t.style.Hidden = false
		case mode == 29:
			t.style.Strikethrough = false

		case mode >= 30 && mode <= 37:
			t.fg = basicColors[mode-30]
		case mode == 39:
			t.fg = nil
		case mode >= 40 && mode <= 47:
			t.bg = basicColors[mode-40]
		case mode == 49:
			t.bg = nil
		case mode >= 90 && mode <= 97:
			t.fg = basicColors[mode-90+8]
		case mode >= 100 && mode <= 107:
			t.bg = basicColors[mode-100+8]

		// "38;5;n" or "38;2;r;g;b", the arguments follow as separate fields
		case mode == 38 || mode == 48:
			i += t.handleExtendedColor(mode, modes[i+1:], false)
		}
	}
}

// handleExtendedColor sets the 256 or RGB color for 38 (fg) / 48 (bg) and
// returns how many of args it used.
func (t *Terminal) handleExtendedColor(mode int, args []string, colon bool) int {
	if len(args) == 0 {
		return 0
	}

	var (
		c    color.Color
		used int
	)

	switch atoi(args[0]) {
	case 5:
		if len(args) < 2 {
			return len(args)
		}
		c = indexedColor(atoi(args[1]))
		used = 2

	case 2:
		rgb := args[1:]
		// The colon form may have a color space id before r:g:b ("38:2::r:g:b")
		if colon && len(rgb) >= 4 {
			rgb = rgb[1:]
		}
		if len(rgb) < 3 {
			return len(args)
		}
		c = color.RGBA{R: clampByte(atoi(rgb[0])), G: clampByte(atoi(rgb[1])), B: clampByte(atoi(rgb[2])), A: 0xff}
		used = 4

	default:
		return 1
	}

	if mode == 38 {
		t.fg = c
	} else if mode == 48 {
		t.bg = c
	}

	return used
}

// atoi treats an empty or invalid field as 0, as terminals do
func atoi(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}

func clampByte(n int) uint8 {
	if n < 0 {
		return 0
	}
	if n > 255 {
		return 255
	}
	return uint8(n)
}
