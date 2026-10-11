package theme

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"image/color"
	"yaterm/assert"
)

// Names of a scheme's colors that are not one of the 16 regular colors, as
// in a Windows Terminal scheme
const (
	ColorBackground          = "background"
	ColorForeground          = "foreground"
	ColorCursor              = "cursorColor"
	ColorSelectionBackground = "selectionBackground"
)

// Scheme is a terminal color scheme. The 16 regular colors are found by
// number, in the order of the terminal's SGR colors (30..37, then the bright
// 90..97), the same as basicColors in terminal/color.go. The others are found
// by name, see ColorBackground and the other names.
type Scheme struct {
	Name   string
	ANSI   [16]color.Color
	Colors map[string]color.Color
}

// The schemes below are the built in schemes of Windows Terminal, from its
// defaults.json (https://github.com/microsoft/terminal, file
// src/cascadia/TerminalSettingsModel/defaults.json).
// Copyright (c) Microsoft Corporation, MIT License, see THIRD_PARTY_NOTICES.
//
// A scheme without a selectionBackground there gets #FFFFFF, the color
// Windows Terminal uses then.

// Campbell is the "Campbell" scheme of Windows Terminal
var Campbell = Scheme{
	Name: "Campbell",

	ANSI: [16]color.Color{
		color.RGBA{R: 0x0C, G: 0x0C, B: 0x0C, A: 0xff}, // black
		color.RGBA{R: 0xC5, G: 0x0F, B: 0x1F, A: 0xff}, // red
		color.RGBA{R: 0x13, G: 0xA1, B: 0x0E, A: 0xff}, // green
		color.RGBA{R: 0xC1, G: 0x9C, B: 0x00, A: 0xff}, // yellow
		color.RGBA{R: 0x00, G: 0x37, B: 0xDA, A: 0xff}, // blue
		color.RGBA{R: 0x88, G: 0x17, B: 0x98, A: 0xff}, // purple (magenta)
		color.RGBA{R: 0x3A, G: 0x96, B: 0xDD, A: 0xff}, // cyan
		color.RGBA{R: 0xCC, G: 0xCC, B: 0xCC, A: 0xff}, // white
		color.RGBA{R: 0x76, G: 0x76, B: 0x76, A: 0xff}, // bright black
		color.RGBA{R: 0xE7, G: 0x48, B: 0x56, A: 0xff}, // bright red
		color.RGBA{R: 0x16, G: 0xC6, B: 0x0C, A: 0xff}, // bright green
		color.RGBA{R: 0xF9, G: 0xF1, B: 0xA5, A: 0xff}, // bright yellow
		color.RGBA{R: 0x3B, G: 0x78, B: 0xFF, A: 0xff}, // bright blue
		color.RGBA{R: 0xB4, G: 0x00, B: 0x9E, A: 0xff}, // bright purple (magenta)
		color.RGBA{R: 0x61, G: 0xD6, B: 0xD6, A: 0xff}, // bright cyan
		color.RGBA{R: 0xF2, G: 0xF2, B: 0xF2, A: 0xff}, // bright white
	},

	Colors: map[string]color.Color{
		ColorBackground:          color.RGBA{R: 0x0C, G: 0x0C, B: 0x0C, A: 0xff},
		ColorForeground:          color.RGBA{R: 0xCC, G: 0xCC, B: 0xCC, A: 0xff},
		ColorCursor:              color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xff},
		ColorSelectionBackground: color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xff},
	},
}

// CampbellPowershell is the "Campbell Powershell" scheme of Windows Terminal
var CampbellPowershell = Scheme{
	Name: "Campbell Powershell",

	ANSI: [16]color.Color{
		color.RGBA{R: 0x0C, G: 0x0C, B: 0x0C, A: 0xff}, // black
		color.RGBA{R: 0xC5, G: 0x0F, B: 0x1F, A: 0xff}, // red
		color.RGBA{R: 0x13, G: 0xA1, B: 0x0E, A: 0xff}, // green
		color.RGBA{R: 0xC1, G: 0x9C, B: 0x00, A: 0xff}, // yellow
		color.RGBA{R: 0x00, G: 0x37, B: 0xDA, A: 0xff}, // blue
		color.RGBA{R: 0x88, G: 0x17, B: 0x98, A: 0xff}, // purple (magenta)
		color.RGBA{R: 0x3A, G: 0x96, B: 0xDD, A: 0xff}, // cyan
		color.RGBA{R: 0xCC, G: 0xCC, B: 0xCC, A: 0xff}, // white
		color.RGBA{R: 0x76, G: 0x76, B: 0x76, A: 0xff}, // bright black
		color.RGBA{R: 0xE7, G: 0x48, B: 0x56, A: 0xff}, // bright red
		color.RGBA{R: 0x16, G: 0xC6, B: 0x0C, A: 0xff}, // bright green
		color.RGBA{R: 0xF9, G: 0xF1, B: 0xA5, A: 0xff}, // bright yellow
		color.RGBA{R: 0x3B, G: 0x78, B: 0xFF, A: 0xff}, // bright blue
		color.RGBA{R: 0xB4, G: 0x00, B: 0x9E, A: 0xff}, // bright purple (magenta)
		color.RGBA{R: 0x61, G: 0xD6, B: 0xD6, A: 0xff}, // bright cyan
		color.RGBA{R: 0xF2, G: 0xF2, B: 0xF2, A: 0xff}, // bright white
	},

	Colors: map[string]color.Color{
		ColorBackground:          color.RGBA{R: 0x01, G: 0x24, B: 0x56, A: 0xff},
		ColorForeground:          color.RGBA{R: 0xCC, G: 0xCC, B: 0xCC, A: 0xff},
		ColorCursor:              color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xff},
		ColorSelectionBackground: color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xff},
	},
}

// Vintage is the "Vintage" scheme of Windows Terminal
var Vintage = Scheme{
	Name: "Vintage",

	ANSI: [16]color.Color{
		color.RGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xff}, // black
		color.RGBA{R: 0x80, G: 0x00, B: 0x00, A: 0xff}, // red
		color.RGBA{R: 0x00, G: 0x80, B: 0x00, A: 0xff}, // green
		color.RGBA{R: 0x80, G: 0x80, B: 0x00, A: 0xff}, // yellow
		color.RGBA{R: 0x00, G: 0x00, B: 0x80, A: 0xff}, // blue
		color.RGBA{R: 0x80, G: 0x00, B: 0x80, A: 0xff}, // purple (magenta)
		color.RGBA{R: 0x00, G: 0x80, B: 0x80, A: 0xff}, // cyan
		color.RGBA{R: 0xC0, G: 0xC0, B: 0xC0, A: 0xff}, // white
		color.RGBA{R: 0x80, G: 0x80, B: 0x80, A: 0xff}, // bright black
		color.RGBA{R: 0xFF, G: 0x00, B: 0x00, A: 0xff}, // bright red
		color.RGBA{R: 0x00, G: 0xFF, B: 0x00, A: 0xff}, // bright green
		color.RGBA{R: 0xFF, G: 0xFF, B: 0x00, A: 0xff}, // bright yellow
		color.RGBA{R: 0x00, G: 0x00, B: 0xFF, A: 0xff}, // bright blue
		color.RGBA{R: 0xFF, G: 0x00, B: 0xFF, A: 0xff}, // bright purple (magenta)
		color.RGBA{R: 0x00, G: 0xFF, B: 0xFF, A: 0xff}, // bright cyan
		color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xff}, // bright white
	},

	Colors: map[string]color.Color{
		ColorBackground:          color.RGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xff},
		ColorForeground:          color.RGBA{R: 0xC0, G: 0xC0, B: 0xC0, A: 0xff},
		ColorCursor:              color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xff},
		ColorSelectionBackground: color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xff},
	},
}

// OneHalfDark is the "One Half Dark" scheme of Windows Terminal
var OneHalfDark = Scheme{
	Name: "One Half Dark",

	ANSI: [16]color.Color{
		color.RGBA{R: 0x28, G: 0x2C, B: 0x34, A: 0xff}, // black
		color.RGBA{R: 0xE0, G: 0x6C, B: 0x75, A: 0xff}, // red
		color.RGBA{R: 0x98, G: 0xC3, B: 0x79, A: 0xff}, // green
		color.RGBA{R: 0xE5, G: 0xC0, B: 0x7B, A: 0xff}, // yellow
		color.RGBA{R: 0x61, G: 0xAF, B: 0xEF, A: 0xff}, // blue
		color.RGBA{R: 0xC6, G: 0x78, B: 0xDD, A: 0xff}, // purple (magenta)
		color.RGBA{R: 0x56, G: 0xB6, B: 0xC2, A: 0xff}, // cyan
		color.RGBA{R: 0xDC, G: 0xDF, B: 0xE4, A: 0xff}, // white
		color.RGBA{R: 0x5A, G: 0x63, B: 0x74, A: 0xff}, // bright black
		color.RGBA{R: 0xE0, G: 0x6C, B: 0x75, A: 0xff}, // bright red
		color.RGBA{R: 0x98, G: 0xC3, B: 0x79, A: 0xff}, // bright green
		color.RGBA{R: 0xE5, G: 0xC0, B: 0x7B, A: 0xff}, // bright yellow
		color.RGBA{R: 0x61, G: 0xAF, B: 0xEF, A: 0xff}, // bright blue
		color.RGBA{R: 0xC6, G: 0x78, B: 0xDD, A: 0xff}, // bright purple (magenta)
		color.RGBA{R: 0x56, G: 0xB6, B: 0xC2, A: 0xff}, // bright cyan
		color.RGBA{R: 0xDC, G: 0xDF, B: 0xE4, A: 0xff}, // bright white
	},

	Colors: map[string]color.Color{
		ColorBackground:          color.RGBA{R: 0x28, G: 0x2C, B: 0x34, A: 0xff},
		ColorForeground:          color.RGBA{R: 0xDC, G: 0xDF, B: 0xE4, A: 0xff},
		ColorCursor:              color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xff},
		ColorSelectionBackground: color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xff},
	},
}

// OneHalfLight is the "One Half Light" scheme of Windows Terminal
var OneHalfLight = Scheme{
	Name: "One Half Light",

	ANSI: [16]color.Color{
		color.RGBA{R: 0x38, G: 0x3A, B: 0x42, A: 0xff}, // black
		color.RGBA{R: 0xE4, G: 0x56, B: 0x49, A: 0xff}, // red
		color.RGBA{R: 0x50, G: 0xA1, B: 0x4F, A: 0xff}, // green
		color.RGBA{R: 0xC1, G: 0x83, B: 0x01, A: 0xff}, // yellow
		color.RGBA{R: 0x01, G: 0x84, B: 0xBC, A: 0xff}, // blue
		color.RGBA{R: 0xA6, G: 0x26, B: 0xA4, A: 0xff}, // purple (magenta)
		color.RGBA{R: 0x09, G: 0x97, B: 0xB3, A: 0xff}, // cyan
		color.RGBA{R: 0xFA, G: 0xFA, B: 0xFA, A: 0xff}, // white
		color.RGBA{R: 0x4F, G: 0x52, B: 0x5D, A: 0xff}, // bright black
		color.RGBA{R: 0xDF, G: 0x6C, B: 0x75, A: 0xff}, // bright red
		color.RGBA{R: 0x98, G: 0xC3, B: 0x79, A: 0xff}, // bright green
		color.RGBA{R: 0xE4, G: 0xC0, B: 0x7A, A: 0xff}, // bright yellow
		color.RGBA{R: 0x61, G: 0xAF, B: 0xEF, A: 0xff}, // bright blue
		color.RGBA{R: 0xC5, G: 0x77, B: 0xDD, A: 0xff}, // bright purple (magenta)
		color.RGBA{R: 0x56, G: 0xB5, B: 0xC1, A: 0xff}, // bright cyan
		color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xff}, // bright white
	},

	Colors: map[string]color.Color{
		ColorBackground:          color.RGBA{R: 0xFA, G: 0xFA, B: 0xFA, A: 0xff},
		ColorForeground:          color.RGBA{R: 0x38, G: 0x3A, B: 0x42, A: 0xff},
		ColorCursor:              color.RGBA{R: 0x4F, G: 0x52, B: 0x5D, A: 0xff},
		ColorSelectionBackground: color.RGBA{R: 0x38, G: 0x3A, B: 0x42, A: 0xff},
	},
}

// SolarizedDark is the "Solarized Dark" scheme of Windows Terminal
var SolarizedDark = Scheme{
	Name: "Solarized Dark",

	ANSI: [16]color.Color{
		color.RGBA{R: 0x00, G: 0x2B, B: 0x36, A: 0xff}, // black
		color.RGBA{R: 0xDC, G: 0x32, B: 0x2F, A: 0xff}, // red
		color.RGBA{R: 0x85, G: 0x99, B: 0x00, A: 0xff}, // green
		color.RGBA{R: 0xB5, G: 0x89, B: 0x00, A: 0xff}, // yellow
		color.RGBA{R: 0x26, G: 0x8B, B: 0xD2, A: 0xff}, // blue
		color.RGBA{R: 0xD3, G: 0x36, B: 0x82, A: 0xff}, // purple (magenta)
		color.RGBA{R: 0x2A, G: 0xA1, B: 0x98, A: 0xff}, // cyan
		color.RGBA{R: 0xEE, G: 0xE8, B: 0xD5, A: 0xff}, // white
		color.RGBA{R: 0x07, G: 0x36, B: 0x42, A: 0xff}, // bright black
		color.RGBA{R: 0xCB, G: 0x4B, B: 0x16, A: 0xff}, // bright red
		color.RGBA{R: 0x58, G: 0x6E, B: 0x75, A: 0xff}, // bright green
		color.RGBA{R: 0x65, G: 0x7B, B: 0x83, A: 0xff}, // bright yellow
		color.RGBA{R: 0x83, G: 0x94, B: 0x96, A: 0xff}, // bright blue
		color.RGBA{R: 0x6C, G: 0x71, B: 0xC4, A: 0xff}, // bright purple (magenta)
		color.RGBA{R: 0x93, G: 0xA1, B: 0xA1, A: 0xff}, // bright cyan
		color.RGBA{R: 0xFD, G: 0xF6, B: 0xE3, A: 0xff}, // bright white
	},

	Colors: map[string]color.Color{
		ColorBackground:          color.RGBA{R: 0x00, G: 0x2B, B: 0x36, A: 0xff},
		ColorForeground:          color.RGBA{R: 0x83, G: 0x94, B: 0x96, A: 0xff},
		ColorCursor:              color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xff},
		ColorSelectionBackground: color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xff},
	},
}

// SolarizedLight is the "Solarized Light" scheme of Windows Terminal
var SolarizedLight = Scheme{
	Name: "Solarized Light",

	ANSI: [16]color.Color{
		color.RGBA{R: 0x00, G: 0x2B, B: 0x36, A: 0xff}, // black
		color.RGBA{R: 0xDC, G: 0x32, B: 0x2F, A: 0xff}, // red
		color.RGBA{R: 0x85, G: 0x99, B: 0x00, A: 0xff}, // green
		color.RGBA{R: 0xB5, G: 0x89, B: 0x00, A: 0xff}, // yellow
		color.RGBA{R: 0x26, G: 0x8B, B: 0xD2, A: 0xff}, // blue
		color.RGBA{R: 0xD3, G: 0x36, B: 0x82, A: 0xff}, // purple (magenta)
		color.RGBA{R: 0x2A, G: 0xA1, B: 0x98, A: 0xff}, // cyan
		color.RGBA{R: 0xEE, G: 0xE8, B: 0xD5, A: 0xff}, // white
		color.RGBA{R: 0x07, G: 0x36, B: 0x42, A: 0xff}, // bright black
		color.RGBA{R: 0xCB, G: 0x4B, B: 0x16, A: 0xff}, // bright red
		color.RGBA{R: 0x58, G: 0x6E, B: 0x75, A: 0xff}, // bright green
		color.RGBA{R: 0x65, G: 0x7B, B: 0x83, A: 0xff}, // bright yellow
		color.RGBA{R: 0x83, G: 0x94, B: 0x96, A: 0xff}, // bright blue
		color.RGBA{R: 0x6C, G: 0x71, B: 0xC4, A: 0xff}, // bright purple (magenta)
		color.RGBA{R: 0x93, G: 0xA1, B: 0xA1, A: 0xff}, // bright cyan
		color.RGBA{R: 0xFD, G: 0xF6, B: 0xE3, A: 0xff}, // bright white
	},

	Colors: map[string]color.Color{
		ColorBackground:          color.RGBA{R: 0xFD, G: 0xF6, B: 0xE3, A: 0xff},
		ColorForeground:          color.RGBA{R: 0x65, G: 0x7B, B: 0x83, A: 0xff},
		ColorCursor:              color.RGBA{R: 0x00, G: 0x2B, B: 0x36, A: 0xff},
		ColorSelectionBackground: color.RGBA{R: 0x2C, G: 0x4D, B: 0x57, A: 0xff},
	},
}

// TangoDark is the "Tango Dark" scheme of Windows Terminal
var TangoDark = Scheme{
	Name: "Tango Dark",

	ANSI: [16]color.Color{
		color.RGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xff}, // black
		color.RGBA{R: 0xCC, G: 0x00, B: 0x00, A: 0xff}, // red
		color.RGBA{R: 0x4E, G: 0x9A, B: 0x06, A: 0xff}, // green
		color.RGBA{R: 0xC4, G: 0xA0, B: 0x00, A: 0xff}, // yellow
		color.RGBA{R: 0x34, G: 0x65, B: 0xA4, A: 0xff}, // blue
		color.RGBA{R: 0x75, G: 0x50, B: 0x7B, A: 0xff}, // purple (magenta)
		color.RGBA{R: 0x06, G: 0x98, B: 0x9A, A: 0xff}, // cyan
		color.RGBA{R: 0xD3, G: 0xD7, B: 0xCF, A: 0xff}, // white
		color.RGBA{R: 0x55, G: 0x57, B: 0x53, A: 0xff}, // bright black
		color.RGBA{R: 0xEF, G: 0x29, B: 0x29, A: 0xff}, // bright red
		color.RGBA{R: 0x8A, G: 0xE2, B: 0x34, A: 0xff}, // bright green
		color.RGBA{R: 0xFC, G: 0xE9, B: 0x4F, A: 0xff}, // bright yellow
		color.RGBA{R: 0x72, G: 0x9F, B: 0xCF, A: 0xff}, // bright blue
		color.RGBA{R: 0xAD, G: 0x7F, B: 0xA8, A: 0xff}, // bright purple (magenta)
		color.RGBA{R: 0x34, G: 0xE2, B: 0xE2, A: 0xff}, // bright cyan
		color.RGBA{R: 0xEE, G: 0xEE, B: 0xEC, A: 0xff}, // bright white
	},

	Colors: map[string]color.Color{
		ColorBackground:          color.RGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xff},
		ColorForeground:          color.RGBA{R: 0xD3, G: 0xD7, B: 0xCF, A: 0xff},
		ColorCursor:              color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xff},
		ColorSelectionBackground: color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xff},
	},
}

// TangoLight is the "Tango Light" scheme of Windows Terminal
var TangoLight = Scheme{
	Name: "Tango Light",

	ANSI: [16]color.Color{
		color.RGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xff}, // black
		color.RGBA{R: 0xCC, G: 0x00, B: 0x00, A: 0xff}, // red
		color.RGBA{R: 0x4E, G: 0x9A, B: 0x06, A: 0xff}, // green
		color.RGBA{R: 0xC4, G: 0xA0, B: 0x00, A: 0xff}, // yellow
		color.RGBA{R: 0x34, G: 0x65, B: 0xA4, A: 0xff}, // blue
		color.RGBA{R: 0x75, G: 0x50, B: 0x7B, A: 0xff}, // purple (magenta)
		color.RGBA{R: 0x06, G: 0x98, B: 0x9A, A: 0xff}, // cyan
		color.RGBA{R: 0xD3, G: 0xD7, B: 0xCF, A: 0xff}, // white
		color.RGBA{R: 0x55, G: 0x57, B: 0x53, A: 0xff}, // bright black
		color.RGBA{R: 0xEF, G: 0x29, B: 0x29, A: 0xff}, // bright red
		color.RGBA{R: 0x8A, G: 0xE2, B: 0x34, A: 0xff}, // bright green
		color.RGBA{R: 0xFC, G: 0xE9, B: 0x4F, A: 0xff}, // bright yellow
		color.RGBA{R: 0x72, G: 0x9F, B: 0xCF, A: 0xff}, // bright blue
		color.RGBA{R: 0xAD, G: 0x7F, B: 0xA8, A: 0xff}, // bright purple (magenta)
		color.RGBA{R: 0x34, G: 0xE2, B: 0xE2, A: 0xff}, // bright cyan
		color.RGBA{R: 0xEE, G: 0xEE, B: 0xEC, A: 0xff}, // bright white
	},

	Colors: map[string]color.Color{
		ColorBackground:          color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xff},
		ColorForeground:          color.RGBA{R: 0x55, G: 0x57, B: 0x53, A: 0xff},
		ColorCursor:              color.RGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xff},
		ColorSelectionBackground: color.RGBA{R: 0x14, G: 0x14, B: 0x14, A: 0xff},
	},
}

// DarkPlus is the "Dark+" scheme of Windows Terminal
var DarkPlus = Scheme{
	Name: "Dark+",

	ANSI: [16]color.Color{
		color.RGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xff}, // black
		color.RGBA{R: 0xCD, G: 0x31, B: 0x31, A: 0xff}, // red
		color.RGBA{R: 0x0D, G: 0xBC, B: 0x79, A: 0xff}, // green
		color.RGBA{R: 0xE5, G: 0xE5, B: 0x10, A: 0xff}, // yellow
		color.RGBA{R: 0x24, G: 0x72, B: 0xC8, A: 0xff}, // blue
		color.RGBA{R: 0xBC, G: 0x3F, B: 0xBC, A: 0xff}, // purple (magenta)
		color.RGBA{R: 0x11, G: 0xA8, B: 0xCD, A: 0xff}, // cyan
		color.RGBA{R: 0xE5, G: 0xE5, B: 0xE5, A: 0xff}, // white
		color.RGBA{R: 0x66, G: 0x66, B: 0x66, A: 0xff}, // bright black
		color.RGBA{R: 0xF1, G: 0x4C, B: 0x4C, A: 0xff}, // bright red
		color.RGBA{R: 0x23, G: 0xD1, B: 0x8B, A: 0xff}, // bright green
		color.RGBA{R: 0xF5, G: 0xF5, B: 0x43, A: 0xff}, // bright yellow
		color.RGBA{R: 0x3B, G: 0x8E, B: 0xEA, A: 0xff}, // bright blue
		color.RGBA{R: 0xD6, G: 0x70, B: 0xD6, A: 0xff}, // bright purple (magenta)
		color.RGBA{R: 0x29, G: 0xB8, B: 0xDB, A: 0xff}, // bright cyan
		color.RGBA{R: 0xE5, G: 0xE5, B: 0xE5, A: 0xff}, // bright white
	},

	Colors: map[string]color.Color{
		ColorBackground:          color.RGBA{R: 0x1E, G: 0x1E, B: 0x1E, A: 0xff},
		ColorForeground:          color.RGBA{R: 0xCC, G: 0xCC, B: 0xCC, A: 0xff},
		ColorCursor:              color.RGBA{R: 0x80, G: 0x80, B: 0x80, A: 0xff},
		ColorSelectionBackground: color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xff},
	},
}

// VSCodeDarkModern is the "VSCode Dark Modern" scheme of Windows Terminal
var VSCodeDarkModern = Scheme{
	Name: "VSCode Dark Modern",

	ANSI: [16]color.Color{
		color.RGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xff}, // black
		color.RGBA{R: 0xCD, G: 0x31, B: 0x31, A: 0xff}, // red
		color.RGBA{R: 0x0D, G: 0xBC, B: 0x79, A: 0xff}, // green
		color.RGBA{R: 0xE5, G: 0xE5, B: 0x10, A: 0xff}, // yellow
		color.RGBA{R: 0x24, G: 0x72, B: 0xC8, A: 0xff}, // blue
		color.RGBA{R: 0xBC, G: 0x3F, B: 0xBC, A: 0xff}, // purple (magenta)
		color.RGBA{R: 0x11, G: 0xA8, B: 0xCD, A: 0xff}, // cyan
		color.RGBA{R: 0xE5, G: 0xE5, B: 0xE5, A: 0xff}, // white
		color.RGBA{R: 0x66, G: 0x66, B: 0x66, A: 0xff}, // bright black
		color.RGBA{R: 0xF1, G: 0x4C, B: 0x4C, A: 0xff}, // bright red
		color.RGBA{R: 0x23, G: 0xD1, B: 0x8B, A: 0xff}, // bright green
		color.RGBA{R: 0xF5, G: 0xF5, B: 0x43, A: 0xff}, // bright yellow
		color.RGBA{R: 0x3B, G: 0x8E, B: 0xEA, A: 0xff}, // bright blue
		color.RGBA{R: 0xD6, G: 0x70, B: 0xD6, A: 0xff}, // bright purple (magenta)
		color.RGBA{R: 0x29, G: 0xB8, B: 0xDB, A: 0xff}, // bright cyan
		color.RGBA{R: 0xE5, G: 0xE5, B: 0xE5, A: 0xff}, // bright white
	},

	Colors: map[string]color.Color{
		ColorBackground:          color.RGBA{R: 0x1F, G: 0x1F, B: 0x1F, A: 0xff},
		ColorForeground:          color.RGBA{R: 0xCC, G: 0xCC, B: 0xCC, A: 0xff},
		ColorCursor:              color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xff},
		ColorSelectionBackground: color.RGBA{R: 0x26, G: 0x4F, B: 0x78, A: 0xff},
	},
}

// VSCodeLightModern is the "VSCode Light Modern" scheme of Windows Terminal
var VSCodeLightModern = Scheme{
	Name: "VSCode Light Modern",

	ANSI: [16]color.Color{
		color.RGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xff}, // black
		color.RGBA{R: 0xCD, G: 0x31, B: 0x31, A: 0xff}, // red
		color.RGBA{R: 0x00, G: 0xBC, B: 0x00, A: 0xff}, // green
		color.RGBA{R: 0x94, G: 0x98, B: 0x00, A: 0xff}, // yellow
		color.RGBA{R: 0x04, G: 0x51, B: 0xA5, A: 0xff}, // blue
		color.RGBA{R: 0xBC, G: 0x05, B: 0xBC, A: 0xff}, // purple (magenta)
		color.RGBA{R: 0x05, G: 0x98, B: 0xBC, A: 0xff}, // cyan
		color.RGBA{R: 0x55, G: 0x55, B: 0x55, A: 0xff}, // white
		color.RGBA{R: 0x66, G: 0x66, B: 0x66, A: 0xff}, // bright black
		color.RGBA{R: 0xCD, G: 0x31, B: 0x31, A: 0xff}, // bright red
		color.RGBA{R: 0x14, G: 0xCE, B: 0x14, A: 0xff}, // bright green
		color.RGBA{R: 0xB5, G: 0xBA, B: 0x00, A: 0xff}, // bright yellow
		color.RGBA{R: 0x04, G: 0x51, B: 0xA5, A: 0xff}, // bright blue
		color.RGBA{R: 0xBC, G: 0x05, B: 0xBC, A: 0xff}, // bright purple (magenta)
		color.RGBA{R: 0x05, G: 0x98, B: 0xBC, A: 0xff}, // bright cyan
		color.RGBA{R: 0xA5, G: 0xA5, B: 0xA5, A: 0xff}, // bright white
	},

	Colors: map[string]color.Color{
		ColorBackground:          color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xff},
		ColorForeground:          color.RGBA{R: 0x3B, G: 0x3B, B: 0x3B, A: 0xff},
		ColorCursor:              color.RGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xff},
		ColorSelectionBackground: color.RGBA{R: 0xAD, G: 0xD6, B: 0xFF, A: 0xff},
	},
}

// CGA is the "CGA" scheme of Windows Terminal
var CGA = Scheme{
	Name: "CGA",

	ANSI: [16]color.Color{
		color.RGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xff}, // black
		color.RGBA{R: 0xAA, G: 0x00, B: 0x00, A: 0xff}, // red
		color.RGBA{R: 0x00, G: 0xAA, B: 0x00, A: 0xff}, // green
		color.RGBA{R: 0xAA, G: 0x55, B: 0x00, A: 0xff}, // yellow
		color.RGBA{R: 0x00, G: 0x00, B: 0xAA, A: 0xff}, // blue
		color.RGBA{R: 0xAA, G: 0x00, B: 0xAA, A: 0xff}, // purple (magenta)
		color.RGBA{R: 0x00, G: 0xAA, B: 0xAA, A: 0xff}, // cyan
		color.RGBA{R: 0xAA, G: 0xAA, B: 0xAA, A: 0xff}, // white
		color.RGBA{R: 0x55, G: 0x55, B: 0x55, A: 0xff}, // bright black
		color.RGBA{R: 0xFF, G: 0x55, B: 0x55, A: 0xff}, // bright red
		color.RGBA{R: 0x55, G: 0xFF, B: 0x55, A: 0xff}, // bright green
		color.RGBA{R: 0xFF, G: 0xFF, B: 0x55, A: 0xff}, // bright yellow
		color.RGBA{R: 0x55, G: 0x55, B: 0xFF, A: 0xff}, // bright blue
		color.RGBA{R: 0xFF, G: 0x55, B: 0xFF, A: 0xff}, // bright purple (magenta)
		color.RGBA{R: 0x55, G: 0xFF, B: 0xFF, A: 0xff}, // bright cyan
		color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xff}, // bright white
	},

	Colors: map[string]color.Color{
		ColorBackground:          color.RGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xff},
		ColorForeground:          color.RGBA{R: 0xAA, G: 0xAA, B: 0xAA, A: 0xff},
		ColorCursor:              color.RGBA{R: 0x00, G: 0xAA, B: 0x00, A: 0xff},
		ColorSelectionBackground: color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xff},
	},
}

// IBM5153 is the "IBM 5153" scheme of Windows Terminal
var IBM5153 = Scheme{
	Name: "IBM 5153",

	ANSI: [16]color.Color{
		color.RGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xff}, // black
		color.RGBA{R: 0xAA, G: 0x00, B: 0x00, A: 0xff}, // red
		color.RGBA{R: 0x00, G: 0xAA, B: 0x00, A: 0xff}, // green
		color.RGBA{R: 0xC4, G: 0x7E, B: 0x00, A: 0xff}, // yellow
		color.RGBA{R: 0x00, G: 0x00, B: 0xAA, A: 0xff}, // blue
		color.RGBA{R: 0xAA, G: 0x00, B: 0xAA, A: 0xff}, // purple (magenta)
		color.RGBA{R: 0x00, G: 0xAA, B: 0xAA, A: 0xff}, // cyan
		color.RGBA{R: 0xAA, G: 0xAA, B: 0xAA, A: 0xff}, // white
		color.RGBA{R: 0x55, G: 0x55, B: 0x55, A: 0xff}, // bright black
		color.RGBA{R: 0xFF, G: 0x55, B: 0x55, A: 0xff}, // bright red
		color.RGBA{R: 0x55, G: 0xFF, B: 0x55, A: 0xff}, // bright green
		color.RGBA{R: 0xFF, G: 0xFF, B: 0x55, A: 0xff}, // bright yellow
		color.RGBA{R: 0x55, G: 0x55, B: 0xFF, A: 0xff}, // bright blue
		color.RGBA{R: 0xFF, G: 0x55, B: 0xFF, A: 0xff}, // bright purple (magenta)
		color.RGBA{R: 0x55, G: 0xFF, B: 0xFF, A: 0xff}, // bright cyan
		color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xff}, // bright white
	},

	Colors: map[string]color.Color{
		ColorBackground:          color.RGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xff},
		ColorForeground:          color.RGBA{R: 0xAA, G: 0xAA, B: 0xAA, A: 0xff},
		ColorCursor:              color.RGBA{R: 0x00, G: 0xAA, B: 0x00, A: 0xff},
		ColorSelectionBackground: color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xff},
	},
}

// Dimidium is the "Dimidium" scheme of Windows Terminal
var Dimidium = Scheme{
	Name: "Dimidium",

	ANSI: [16]color.Color{
		color.RGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xff}, // black
		color.RGBA{R: 0xCF, G: 0x49, B: 0x4C, A: 0xff}, // red
		color.RGBA{R: 0x60, G: 0xB4, B: 0x42, A: 0xff}, // green
		color.RGBA{R: 0xDB, G: 0x9C, B: 0x11, A: 0xff}, // yellow
		color.RGBA{R: 0x05, G: 0x75, B: 0xD8, A: 0xff}, // blue
		color.RGBA{R: 0xAF, G: 0x5E, B: 0xD2, A: 0xff}, // purple (magenta)
		color.RGBA{R: 0x1D, G: 0xB6, B: 0xBB, A: 0xff}, // cyan
		color.RGBA{R: 0xBA, G: 0xB7, B: 0xB6, A: 0xff}, // white
		color.RGBA{R: 0x81, G: 0x7E, B: 0x7E, A: 0xff}, // bright black
		color.RGBA{R: 0xFF, G: 0x64, B: 0x3B, A: 0xff}, // bright red
		color.RGBA{R: 0x37, G: 0xE5, B: 0x7B, A: 0xff}, // bright green
		color.RGBA{R: 0xFC, G: 0xCD, B: 0x1A, A: 0xff}, // bright yellow
		color.RGBA{R: 0x68, G: 0x8D, B: 0xFD, A: 0xff}, // bright blue
		color.RGBA{R: 0xED, G: 0x6F, B: 0xE9, A: 0xff}, // bright purple (magenta)
		color.RGBA{R: 0x32, G: 0xE0, B: 0xFB, A: 0xff}, // bright cyan
		color.RGBA{R: 0xDE, G: 0xE3, B: 0xE4, A: 0xff}, // bright white
	},

	Colors: map[string]color.Color{
		ColorBackground:          color.RGBA{R: 0x14, G: 0x14, B: 0x14, A: 0xff},
		ColorForeground:          color.RGBA{R: 0xBA, G: 0xB7, B: 0xB6, A: 0xff},
		ColorCursor:              color.RGBA{R: 0x37, G: 0xE5, B: 0x7B, A: 0xff},
		ColorSelectionBackground: color.RGBA{R: 0x8D, G: 0xB8, B: 0xE5, A: 0xff},
	},
}

// Ottosson is the "Ottosson" scheme of Windows Terminal
var Ottosson = Scheme{
	Name: "Ottosson",

	ANSI: [16]color.Color{
		color.RGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xff}, // black
		color.RGBA{R: 0xBE, G: 0x2C, B: 0x21, A: 0xff}, // red
		color.RGBA{R: 0x3F, G: 0xAE, B: 0x3A, A: 0xff}, // green
		color.RGBA{R: 0xBE, G: 0x9A, B: 0x4A, A: 0xff}, // yellow
		color.RGBA{R: 0x20, G: 0x4D, B: 0xBE, A: 0xff}, // blue
		color.RGBA{R: 0xBB, G: 0x54, B: 0xBE, A: 0xff}, // purple (magenta)
		color.RGBA{R: 0x00, G: 0xA7, B: 0xB2, A: 0xff}, // cyan
		color.RGBA{R: 0xBE, G: 0xBE, B: 0xBE, A: 0xff}, // white
		color.RGBA{R: 0x80, G: 0x80, B: 0x80, A: 0xff}, // bright black
		color.RGBA{R: 0xFF, G: 0x3E, B: 0x30, A: 0xff}, // bright red
		color.RGBA{R: 0x58, G: 0xEA, B: 0x51, A: 0xff}, // bright green
		color.RGBA{R: 0xFF, G: 0xC9, B: 0x44, A: 0xff}, // bright yellow
		color.RGBA{R: 0x2F, G: 0x6A, B: 0xFF, A: 0xff}, // bright blue
		color.RGBA{R: 0xFC, G: 0x74, B: 0xFF, A: 0xff}, // bright purple (magenta)
		color.RGBA{R: 0x00, G: 0xE1, B: 0xF0, A: 0xff}, // bright cyan
		color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xff}, // bright white
	},

	Colors: map[string]color.Color{
		ColorBackground:          color.RGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xff},
		ColorForeground:          color.RGBA{R: 0xBE, G: 0xBE, B: 0xBE, A: 0xff},
		ColorCursor:              color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xff},
		ColorSelectionBackground: color.RGBA{R: 0x92, G: 0xA4, B: 0xFD, A: 0xff},
	},
}

// Schemes are all the schemes, by their name
var Schemes = map[string]*Scheme{
	"Campbell":            &Campbell,
	"Campbell Powershell": &CampbellPowershell,
	"Vintage":             &Vintage,
	"One Half Dark":       &OneHalfDark,
	"One Half Light":      &OneHalfLight,
	"Solarized Dark":      &SolarizedDark,
	"Solarized Light":     &SolarizedLight,
	"Tango Dark":          &TangoDark,
	"Tango Light":         &TangoLight,
	"Dark+":               &DarkPlus,
	"VSCode Dark Modern":  &VSCodeDarkModern,
	"VSCode Light Modern": &VSCodeLightModern,
	"CGA":                 &CGA,
	"IBM 5153":            &IBM5153,
	"Dimidium":            &Dimidium,
	"Ottosson":            &Ottosson,
}

type YatermTheme struct {
	fyne.Theme

	variant fyne.ThemeVariant
}

func (f *YatermTheme) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	return f.Theme.Color(name, f.variant)
}

func (f *YatermTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNameSplitThickness:
		return 4
	}

	return f.Theme.Size(name)
}

func (t *YatermTheme) Font(s fyne.TextStyle) fyne.Resource {
	if !s.Monospace {
		return t.Theme.Font(s)
	}

	if s.Bold {
		return assert.ResourceJetBrainsMonoBoldTtf
	} else if s.Italic {
		return assert.ResourceJetBrainsMonoItalicTtf
	}

	return assert.ResourceJetBrainsMonoRegularTtf
}

func NewYatermTheme() *YatermTheme {
	return &YatermTheme{Theme: theme.DarkTheme(), variant: theme.VariantDark}
}
