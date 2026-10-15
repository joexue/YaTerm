package terminal

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

// processDec takes the final character of "ESC ( X" (G0) or "ESC ) X" (G1).
// It is one character, then the terminal goes back to normal output.
func (t *Terminal) processDec(r rune, set int) {
	switch r {
	case charsetDECGraphics, charsetUK:
		t.charsets[set] = r
	default:
		// ASCII, and the national sets that are not supported
		t.charsets[set] = charsetASCII
	}

	t.state = stateGROUND
}
