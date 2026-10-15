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

const (
	StringTerminator = 0x9c
)

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
			t.processOsc(StringTerminator)
		} else {
			t.state = stateESC
			t.processEsc(r)
		}

	case stateAPC:
		t.processApc(r)

	case stateAPCTermination:
		if r == '\\' {
			t.processApc(StringTerminator)
		} else {
			t.state = stateESC
			t.processEsc(r)
		}

	case stateDCS:
		t.processDcs(r)

	case stateDCSTermination:
		if r == '\\' {
			t.processDcs(StringTerminator)
		} else {
			t.state = stateESC
			t.processEsc(r)
		}

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
		if len(data) < utf8.UTFMax && !utf8.FullRune(data) {
			t.leftOver = append(t.leftOver, data...)
			break
		}

		r, size := utf8.DecodeRune(data)
		data = data[size:]

		// Not UTF-8, still show it as "�" let user know
		if r == utf8.RuneError && size == 1 {
			t.processRune(utf8.RuneError)
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
