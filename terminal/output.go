package terminal

import (
	"unicode/utf8"
)

const (
	stateGROUND = iota
	stateESC
	stateESCIntermediate
	stateCSI
	stateOSC
	stateOSCTermination
	stateDCS
	stateDCSTmux
	stateDCSTermination
	stateAPC
	stateAPCTermination
)

// stringTerminator is the 8 bit ST (C1), it ends an OSC, a DCS or an APC
// string, as "ESC \" does
const stringTerminator = 0x9c

// processRune runs one character of the output through the state machine
func (t *Terminal) processRune(r rune) {
	// For OSC DCS APC, they accept ST(String terminator) which is: ESC '\\'
	if r == asciiEscape {
		switch t.state {
		case stateOSC:
			t.state = stateOSCTermination
		case stateDCS, stateDCSTmux:
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
		t.processEsc(r)

	case stateESCIntermediate:
		t.processEscIntermediate(r)

	case stateCSI:
		t.processCsi(r)

	case stateOSC:
		t.processOsc(r)

	// If it is 'ESC ST' pattern then let it go otherwise fallback to escape mode
	case stateOSCTermination:
		if r == '\\' {
			t.processOsc(stringTerminator)
		} else {
			t.processOsc(stringTerminator)
			t.state = stateESC
			t.processEsc(r)
		}

	case stateAPC:
		t.processApc(r)

	case stateAPCTermination:
		if r == '\\' {
			t.processApc(stringTerminator)
		} else {
			t.processApc(stringTerminator)
			t.state = stateESC
			t.processEsc(r)
		}

	case stateDCS:
		t.processDcs(r)

	/*
		case stateDCSTmux:
			t.processDcsTmux(r)
	*/

	case stateDCSTermination:
		if r == '\\' {
			t.processDcs(stringTerminator)
		} else {
			// If ssh connection is closed by peer, Conpty stucks in DCS state.
			// Need to give it extra instruction to restore it back. Otherwise
			// the cursor will not right (Conpty has its own state machine).
			if t.tmux != nil {
				t.closePtyString()
			}
			t.processDcs(stringTerminator)
			t.state = stateESC
			t.processEsc(r)
		}

	default:
		t.processGround(r)
	}
}

// ProcessOutput runs the first num bytes of the output of the program through
// the parser onto the screen. A character cut at the end waits for the rest.
func (t *Terminal) ProcessOutput(bytes []byte, num int) {
	t.stateMu.Lock()
	defer t.stateMu.Unlock()

	data := bytes[:num]
	if len(t.leftOver) > 0 {
		data = append(t.leftOver, data...)
		t.leftOver = nil
	}

	for len(data) > 0 {
		// Tmux handle the bytes
		if t.state == stateDCSTmux {
			t.processDcsTmux(data[0])
			data = data[1:]
			continue
		}

		r, size := utf8.DecodeRune(data)
		// The start of a character whose rest comes with the next output.
		// Any other byte that is not UTF-8 is shown as "�" right away.
		if r == utf8.RuneError && size == 1 && !utf8.FullRune(data) {
			t.leftOver = append([]byte(nil), data...)
			break
		}

		if isC1ControlCharacter(r) {
			t.processRune(asciiEscape)
			t.processRune(c1ToC0ControlCharacter(r))
		} else {
			t.processRune(r)
		}

		data = data[size:]
	}

	t.screen.Flush(t.cursorShown())
}
