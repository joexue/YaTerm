package terminal

func (t *Terminal) handleEscAction(r rune) {
	switch r {
	// Moves the cursor down one line in the same column. If the cursor is at the bottom margin, the page scrolls up
	case 'D':
		t.wrapPending = false
		t.lineFeed()

	// Moves the cursor to the left margin on the next line. If the cursor is at the bottom margin, scroll the page up
	case 'E':
		t.wrapPending = false
		t.cursorX = 0
		t.lineFeed()

	// Sets a horizontal tab stop at the column where the cursor is
	case 'H', 'f':

	// Move the cursor up one line. If the cursor is at the top margin, scroll the region down
	case 'M':
		t.wrapPending = false
		t.reverseIndex()

	// Resets tab stops, margins, modes, graphic rendition, palette, activates primary screen, erases the display and moves cursor to home position
	case 'c':

	// Records cursor position
	case '7':
		t.saveCursor()

	// Moves cursor to location it had when stateDECSC was used
	case '8':
		t.restoreCursor()

	// Enable Application Keypad Mode
	case '=':

	// Set Normal Keypad Mode
	case '>':

	default:
	}
}

// ref: https://wezterm.org/escape-sequences.html#c1-control-codes
// ref: https://invisible-island.net/xterm/ctlseqs/ctlseqs.html
func (t *Terminal) processEsc(r rune) {
	switch r {
	case '[':
		t.state = stateCSI

	// Operating System Command Sequences
	case ']':
		t.state = stateOSC

	case 'P':
		t.state = stateDCS

	// APC, SOS and PM: strings that are read and ignored, see processApc
	case '_', 'X', '^':
		t.state = stateAPC

	// "(0" Translate characters j-x to line drawing glyphs
	// "(B" Disables stateDEC Line Drawing character translation
	// "(A" Alternative character
	case '(':
		t.state = stateDECCharSet0

	// ")0" ")B" ")A" same as abover for character set1
	case ')':
		t.state = stateDECCharSet1

	// "#8" Fills the display with E characters for diagnostic/test purposes (for vttest)
	case '#':
		t.state = stateTEST

	default:
		t.handleEscAction(r)
		t.state = stateGROUND
	}

	t.param = nil
}
