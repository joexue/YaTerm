package terminal

// Which character set "ESC ( X" or "ESC ) X" designates: G0 or G1
const (
	decStateCONTROL0 = iota
	decStateCONTROL1
)

const (
	asciiShiftOut = 0x0e // SO: use G1
	asciiShiftIn  = 0x0f // SI: use G0
)

// Character sets, the final character of the designation
const (
	charsetASCII       = 'B'
	charsetUK          = 'A' // '#' is '£'
	charsetDECGraphics = '0' // line drawing
)

// decGraphics is the DEC Special Graphics set for 0x5f..0x7e, used by
// ncurses programs (htop, mc, dialog) to draw boxes: "lqk" is "┌─┐"
var decGraphics = [...]rune{
	' ', '◆', '▒', '␉', '␌', '␍', '␊', '°', // _ ` a b c d e f
	'±', '␤', '␋', '┘', '┐', '┌', '└', '┼', // g h i j k l m n
	'⎺', '⎻', '─', '⎼', '⎽', '├', '┤', '┴', // o p q r s t u v
	'┬', '│', '≤', '≥', 'π', '≠', '£', '·', // w x y z { | } ~
}

// processDec takes the final character of "ESC ( X" (G0) or "ESC ) X" (G1).
// It is one character, then the terminal goes back to normal output.
func (t *Terminal) processDec(r rune) {
	t.state = stateGROUND

	g := 0
	if t.decState == decStateCONTROL1 {
		g = 1
	}

	switch r {
	case charsetDECGraphics, charsetUK:
		t.charsets[g] = r
	default:
		// ASCII, and the national sets that are not supported
		t.charsets[g] = charsetASCII
	}
}

// translateChar maps a character through the character set in use, G0 or
// G1 as chosen by SI and SO
func (t *Terminal) translateChar(r rune) rune {
	switch t.charsets[t.shifted] {
	case charsetDECGraphics:
		if r >= 0x5f && r <= 0x7e {
			return decGraphics[r-0x5f]
		}
	case charsetUK:
		if r == '#' {
			return '£'
		}
	}
	return r
}
