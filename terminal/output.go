package terminal

import (
	//"fmt"
	"unicode/utf8"
)

const (
	asciiNull      = 0
	asciiBell      = 7
	asciiBackspace = 8
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
	switch r {
	// Moves the cursor down one line in the same column. If the cursor is at the bottom margin, the page scrolls up
	case 'D':

	// Moves the cursor to the left margin on the next line. If the cursor is at the bottom margin, scroll the page up
	case 'E':

	// Sets a horizontal tab stop at the column where the cursor is
	case 'H':

	// Move the cursor up one line. If the cursor is at the top margin, scroll the region down
	case 'M':

	case 'P':
		t.state = stateDCS

	case '[':
		t.state = stateCSI

	// No direct effect; ST is used to delimit the end of stateOSC style escape sequences
	case '\\':
		if t.state == stateOSC {
			t.ProcessOsc(r)
		} else {
			t.state = stateGROUND
		}

	// Resets tab stops, margins, modes, graphic rendition, palette, activates primary screen, erases the display and moves cursor to home position
	case 'c':

	// Records cursor position
	case '7':
		t.savedRow = t.cursorY
		t.savedCol = t.cursorX

	// Moves cursor to location it had when stateDECSC was used
	case '8':
		t.cursorX = t.savedRow
		t.cursorY = t.savedCol

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

	// "#8" Fills the display with E characters for diagnostic/test purposes (for vttest)
	case '#':
		t.state = stateTEST

	case '_':
		t.state = stateAPC

	// Just reset the sate to ground if we don't recognize it, may not right or never happen?
	default:
		t.state = stateGROUND
	}
}

func (t *Terminal) ProcessOutput(bytes []byte, num int) {
	var (
		size int
		r    rune
	)

	for i := 0; i < num; i = i + size {
		bytes = bytes[size:]
		r, size = utf8.DecodeRune(bytes)
		if size == 0 {
			break
		}

		if size == 1 && r == utf8.RuneError {
			if len(bytes) < 4 {
				// may be part of unicode
				break
			} else {
				// really wrong code
				continue
			}
		}

		//fmt.Println("xxxxx state:", t.state, r, string(r))

		if r == asciiEscape {
			t.escape = true
			continue
		}

		if t.escape {
			t.parseEscState(r)
			t.escape = false
			continue
		}

		switch t.state {
		case stateCSI:
			t.ProcessCsi(r)
			continue

		case stateOSC:
			t.ProcessOsc(r)
			continue

		case stateAPC:
			t.ProcessApc(r)
			continue

		case stateDCS:
			t.ProcessDcs(r)
			continue

		case stateDEC:
			t.ProcessDec(r)
			continue

		case stateTEST:
			t.ProcessTest(r)
			continue
		}

		t.ProcessGround(r)
	}
}
