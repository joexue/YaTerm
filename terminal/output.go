package terminal

import (
	"unicode/utf8"
)

const (
	asciiNull      = 0
	asciiBell      = 7
	asciiBackspace = 8
	asciiTab       = 9
	asciiLineFeed  = 10
	asciiVertTab   = 11
	asciiFormFeed  = 12
	asciiReturn    = 13
	asciiEscape    = 27
)

const (
	stateGROUND = iota
	stateESC
	stateCSI
	stateOSC
	stateOSCTermination
	stateDCS
	stateDCSTmux
	stateDCSTermination
	stateAPC
	stateAPCTermination
	stateDECCharSet0
	stateDECCharSet1
	stateTEST
)

const (
	StringTerminator = 0x9c
)

// ref: https://wezterm.org/escape-sequences.html#c1-control-codes
// ref: https://invisible-island.net/xterm/ctlseqs/ctlseqs.html
func (t *Terminal) parseEscState(r rune) {
	switch r {
	// ================================
	//  State transition sets
	// ================================
	case '[':
		t.state = stateCSI
		t.param = nil
		return

	// Operating System Command Sequences
	case ']':
		t.state = stateOSC
		t.param = nil
		return

	case 'P':
		t.state = stateDCS
		t.param = nil
		return

	// APC, SOS and PM: strings that are read and ignored, see processApc
	case '_', 'X', '^':
		t.state = stateAPC
		t.param = nil
		return

	// "(0" Translate characters j-x to line drawing glyphs
	// "(B" Disables stateDEC Line Drawing character translation
	// "(A" Alternative character
	case '(':
		t.state = stateDECCharSet0
		t.param = nil
		return

	case ')':
		t.state = stateDECCharSet1
		t.param = nil
		return

	// "#8" Fills the display with E characters for diagnostic/test purposes (for vttest)
	case '#':
		t.state = stateTEST
		t.param = nil
		return

	// ================================
	//  Execute directly sets
	// ================================
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

	t.state = stateGROUND
}

// processRune runs one character of the output through the state machine
func (t *Terminal) processRune(r rune) {
	// For OSC DCS APC, they accept ST(String terminator) which is: ESC '\\'
	if r == asciiEscape {
		switch t.state {
		case stateOSC:
			t.state = stateOSCTermination
		case stateDCS:
			t.state = stateDCSTermination
		case stateAPC:
			t.state = stateAPCTermination
		default:
			t.state = stateESC
		}
		return
	}

	switch t.state {
	case stateESC:
		t.parseEscState(r)

	case stateCSI:
		t.processCsi(r)

	case stateOSC:
		t.processOsc(r)

	// If it is 'ESC ST' pattern then let it go otherwise fallback to escape mode
	case stateOSCTermination:
		if r == '\\' {
			t.processOsc(StringTerminator)
		} else {
			t.state = stateESC
			t.parseEscState(r)
		}

	case stateAPC:
		t.processApc(r)

	case stateAPCTermination:
		if r == '\\' {
			t.processApc(StringTerminator)
		} else {
			t.state = stateESC
			t.parseEscState(r)
		}

	case stateDCS:
		t.processDcs(r)

	case stateDCSTermination:
		if r == '\\' {
			t.processDcs(StringTerminator)
		} else {
			t.state = stateESC
			t.parseEscState(r)
		}

	case stateDECCharSet0:
		t.processDec(0, r)

	case stateDECCharSet1:
		t.processDec(1, r)

	case stateTEST:
		t.processTest(r)

	default:
		t.processGround(r)
	}
}

func (t *Terminal) ProcessOutput(bytes []byte, num int) {
	t.stateMu.Lock()
	defer t.stateMu.Unlock()

	var data []byte
	if len(t.leftOver) > 0 {
		data = append(t.leftOver, bytes...)
		t.leftOver = nil
	} else {
		data = bytes[:num]
	}

	for len(data) > 0 {
		// A character split over two reads is finished with the next read
		if len(data) < 4 && !utf8.FullRune(data) {
			t.leftOver = append(t.leftOver, data...)
			break
		}

		r, size := utf8.DecodeRune(data)
		data = data[size:]

		// Not UTF-8, e.g. a raw 8 bit C1 byte, dropped
		if r == utf8.RuneError && size == 1 {
			continue
		}

		if isC1ControlCharacter(r) {
			t.processRune(asciiEscape)
			t.processRune(c1ToC0ControlCharacter(r))
		} else {
			t.processRune(r)
		}
	}

	t.screen.Flush(t.cursorShown())
}
