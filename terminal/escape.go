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

func (t *Terminal) testAlign() {
	for i := 0; i < t.row; i++ {
		for j := 0; j < t.col; j++ {
			t.screen.Project('E', i, j, t.fg, t.bg, t.style)
		}
	}

	t.cursorX = 0
	t.cursorY = 0
	t.screen.Flush(t.cursorY, t.cursorX)
}

// processDec takes the final character of "ESC ( X" (G0) or "ESC ) X" (G1).
// It is one character, then the terminal goes back to normal output.
func (t *Terminal) handleCharacterSet(r rune, set int) {
	switch r {
	case charsetDECGraphics, charsetUK:
		t.charsets[set] = r
	default:
		// ASCII, and the national sets that are not supported
		t.charsets[set] = charsetASCII
	}

	t.state = stateGROUND
}

func (t *Terminal) handleTest(r rune) {
	switch r {
	case '8':
		t.testAlign()
	}
}

func (t *Terminal) processEscIntermediate(r rune) {
	interMediate := t.param[0]

	switch interMediate {
	case '#':
		t.handleTest(r)
	case '(':
		t.handleCharacterSet(r, 0)
	case ')':
		t.handleCharacterSet(r, 1)
	}

	t.param = nil
	t.state = stateGROUND
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

	default:
		if isIntermediate(r) {
			t.param = append(t.param, r)
			t.state = stateESCIntermediate
			return
		} else {
			t.handleEscAction(r)
			t.state = stateGROUND
		}
	}

	t.param = nil
}
