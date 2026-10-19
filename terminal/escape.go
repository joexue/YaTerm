package terminal

func (t *Terminal) handleEscAction(r rune) {
	if isC0ControlCharacter(r) {
		t.handleAction(r)
		return
	}

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
	case 'H':

	// Move the cursor up one line. If the cursor is at the top margin, scroll the region down
	case 'M':
		t.wrapPending = false
		t.reverseIndex()

	// Single Shift Select of G2 Character Set (SS2  is 0x8e), VT220.
	// This affects next character only.
	case 'N':

	// Single Shift Select of G3 Character Set (SS3  is 0x8f), VT220.
	// This affects next character only.
	case 'O':

	// Start of Guarded Area (SPA  is 0x96).
	case 'V':

	// End of Guarded Area (EPA  is 0x97).
	case 'W':

	// Start of String (SOS  is 0x98).
	case 'X':

	// Return Terminal ID (DECID is 0x9a).  Obsolete form of CSI c  (DA).
	case 'Z':

	// Privacy Message (PM  is 0x9e).
	case '^':

	// Back Index (DECBI), VT420 and up.
	case '6':

	// Records cursor position
	case '7':
		t.saveCursor()

	// Moves cursor to location it had when stateDECSC was used
	case '8':
		t.restoreCursor()

	// Forward Index (DECFI), VT420 and up.
	case '9':

	// Enable Application Keypad Mode
	case '=':

	// Set Normal Keypad Mode
	case '>':

	// Cursor to lower left corner of screen.  This is enabled by the
	// hpLowerleftBugCompat resource.
	case 'F':

	// Resets tab stops, margins, modes, graphic rendition, palette, activates primary screen, erases the display and moves cursor to home position
	case 'c':
		t.setAlternate(false)

	// Memory Lock (per HP terminals).  Locks memory above the cursor.
	case 'l':

	// Memory Unlock (per HP terminals).
	case 'm':

	// Invoke the G2 Character Set as GL (LS2).
	case 'n':

	// Invoke the G3 Character Set as GL (LS3).
	case 'o':

	// Invoke the G3 Character Set as GR (LS3R).
	case '|':

	// Invoke the G2 Character Set as GR (LS2R).
	case '}':

	// Invoke the G1 Character Set as GR (LS1R), VT100.
	case '~':

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
}

func (t *Terminal) handleTest(r rune) {
	switch r {
	case '8':
		t.testAlign()
	}
}

func (t *Terminal) processEscIntermediate(r rune) {
	if isC0ControlCharacter(r) {
		t.handleAction(r)
		return
	}

	if r == asciiDelete {
		return
	}

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

// ref: https://invisible-island.net/xterm/ctlseqs/ctlseqs.html
// ref: https://wezterm.org/escape-sequences.html#c1-control-codes
// ref: https://github.com/microsoft/terminal
func (t *Terminal) processEsc(r rune) {
	switch r {
	// Control Sequence Introducer
	case '[':
		t.state = stateCSI
		t.param = nil

	// Operating System Command Sequences
	case ']':
		t.state = stateOSC
		t.param = nil

	// Device Control String
	case 'P':
		t.state = stateDCS
		t.param = nil

	// Application Program Command
	case '_':
		t.state = stateAPC
		t.param = nil

	// ignore it
	case asciiDelete:

	default:
		if isIntermediate(r) {
			t.param = []rune{r}
			t.state = stateESCIntermediate
		} else {
			t.handleEscAction(r)
			t.state = stateGROUND
			t.param = nil
		}
	}
}
