package terminal

import (
	"strconv"
)

const (
	asciiNull      = 0x00 // Null
	asciiSoh       = 0x01 // Start of Heading
	asciiStx       = 0x02 // Start of Text
	asciiExt       = 0x03 // End of Text
	asciiEot       = 0x04 // End of Transmission
	asciiEnq       = 0x05 // Enquiry
	asciiAck       = 0x06 // Acknowledge
	asciiBell      = 0x07 // Bell
	asciiBackspace = 0x08 // Backspace
	asciiTab       = 0x09 // Horizontal Tab
	asciiLineFeed  = 0x0A // Line Feed (new line)
	asciiVertTab   = 0x0B // Vertical Tab
	asciiFormFeed  = 0x0C // Form Feed (new page)
	asciiReturn    = 0x0D // Carriage Return
	asciiShiftOut  = 0x0E // Shift Out
	asciiShiftIn   = 0x0F // Shift In
	asciiDataLink  = 0x10 // Data Link Escape
	asciiDevCtl1   = 0x11 // Device Control 1
	asciiDevCtl2   = 0x12 // Device Control 2
	asciiDevCtl3   = 0x13 // Device Control 3
	asciiDevCtl4   = 0x14 // Device Control 4
	asciiNack      = 0x15 // Negative Acknowledge
	asciiSyn       = 0x16 // Synchronous Idle
	asciiEndTB     = 0x17 // End of Transmission Block
	asciiCancel    = 0x18 // Cancel
	asciiEndMedia  = 0x19 // End of Medium
	asciiSub       = 0x1A // Substitute
	asciiEscape    = 0x1B // Escape
	asciiFileSep   = 0x1C // File Separator
	asciiGroupSep  = 0x1D // Group Separator
	asciiRecordSep = 0x1E // Record Separator
	asciiUnitSep   = 0x1F // Unit Separator
	asciiSpace     = 0x20 // Space, first printable character
	asciiDelete    = 0x7F // Delete
)

func isC1ControlCharacter(r rune) bool {
	return r >= 0x80 && r <= 0x9F
}

func c1ToC0ControlCharacter(r rune) rune {
	return r - 0x40
}

// isIntermediate tells if r is a CSI intermediate byte
func isIntermediate(r rune) bool {
	return r >= 0x20 && r <= 0x2F
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

// param returns the i-th parameter, or def when it is missing or zero for counts
func param(params []int, i, def int) int {
	if i >= len(params) || params[i] < 0 {
		return def
	}

	if def > 0 && params[i] == 0 {
		return def
	}

	return params[i]
}

func isC0ControlCharacter(r rune) bool {
	return (r >= asciiNull && r <= asciiEndTB) ||
		r == asciiEndMedia ||
		(r >= asciiFileSep && r <= asciiUnitSep)
}
