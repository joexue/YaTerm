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
	stateCSI
	stateOSC
	stateDCS
	stateAPC
	stateDEC
	stateTEST
)

// ref: https://wezterm.org/escape-sequences.html#c1-control-codes
func (t *Terminal) parseEscState(r rune) {
	// Inside a DCS only "ESC \" ends it, any other ESC is part of its text,
	// e.g. tmux passthrough doubles the ESC of the sequences it carries
	if t.state == stateDCS && r != '\\' {
		if len(t.param) < dcsMaxLen {
			t.param = append(t.param, asciiEscape, r)
		}
		return
	}

	// Inside an APC, SOS or PM, an ESC with anything but '\' aborts it and
	// starts a new escape sequence, so a string that is never ended does not
	// swallow all the output after it
	if t.state == stateAPC && r != '\\' {
		t.state = stateGROUND
	}

	switch r {
	// Moves the cursor down one line in the same column. If the cursor is at the bottom margin, the page scrolls up
	case 'D':
		t.wrapPending = false
		t.lineFeed()
		t.screen.MoveCursor(t.cursorY, t.cursorX)

	// Moves the cursor to the left margin on the next line. If the cursor is at the bottom margin, scroll the page up
	case 'E':
		t.wrapPending = false
		t.cursorX = 0
		t.lineFeed()
		t.screen.MoveCursor(t.cursorY, t.cursorX)

	// Sets a horizontal tab stop at the column where the cursor is
	case 'H':

	// Move the cursor up one line. If the cursor is at the top margin, scroll the region down
	case 'M':
		t.wrapPending = false
		t.reverseIndex()
		t.screen.MoveCursor(t.cursorY, t.cursorX)

	case 'P':
		t.state = stateDCS
		t.param = []rune{}

	case '[':
		t.state = stateCSI
		t.param = []rune{}

	// No direct effect; ST is used to delimit the end of stateOSC style escape sequences
	case '\\':
		if t.state == stateOSC {
			t.processOsc(0x9c)
		} else if t.state == stateDCS {
			t.processDcs(0x9c)
		} else {
			t.state = stateGROUND
		}

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

	// "(0" Translate characters j-x to line drawing glyphs
	// "(B" Disables stateDEC Line Drawing character translation
	// "(A" Alternative character
	case '(':
		t.state = stateDEC
		t.decState = decStateCONTROL0

	case ')':
		t.state = stateDEC
		t.decState = decStateCONTROL1

	// Operating System Command Sequences
	case ']':
		t.state = stateOSC
		t.param = []rune{}

	// "#8" Fills the display with E characters for diagnostic/test purposes (for vttest)
	case '#':
		t.state = stateTEST

	// APC, SOS and PM: strings that are read and ignored, see processApc
	case '_', 'X', '^':
		t.state = stateAPC

	// Just reset the sate to ground if we don't recognize it, may not right or never happen?
	default:
		t.state = stateGROUND
	}
}

func (t *Terminal) ProcessOutput(bytes []byte, num int) {
	t.stateMu.Lock()
	defer t.stateMu.Unlock()

	data := bytes[:num]
	if len(t.pending) > 0 {
		data = append(t.pending, data...)
		t.pending = nil
	}

	for len(data) > 0 {
		// A character split over two reads is finished with the next read
		if !utf8.FullRune(data) {
			t.pending = append([]byte(nil), data...)
			break
		}

		r, size := utf8.DecodeRune(data)
		data = data[size:]

		// Not UTF-8, e.g. a raw 8 bit C1 byte, dropped
		if r == utf8.RuneError && size == 1 {
			continue
		}

		t.processRune(r)
	}

	t.screen.Flush(t.cursorY, t.cursorX)
}

// processRune runs one character of the output through the state machine
func (t *Terminal) processRune(r rune) {
	if r == asciiEscape {
		t.escape = true
		return
	}

	if isC1(r) {
		t.processC1(r)
		return
	}

	if t.escape {
		t.parseEscState(r)
		t.escape = false
		return
	}

	switch t.state {
	case stateCSI:
		t.processCsi(r)
	case stateOSC:
		t.processOsc(r)
	case stateAPC:
		t.processApc(r)
	case stateDCS:
		t.processDcs(r)
	case stateDEC:
		t.processDec(r)
	case stateTEST:
		t.processTest(r)
	default:
		t.processGround(r)
	}
}
