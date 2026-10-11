package terminal

// C1 controls (0x80..0x9f) are the one character form of ESC followed by the
// code minus 0x40, e.g. CSI 0x9b is "ESC [". The output is UTF-8, so they come
// as the characters U+0080..U+009F, like xterm takes them in UTF-8 mode. Raw
// 8 bit bytes 0x80..0x9f are not UTF-8 and are dropped.
const (
	c1IND = 0x84 // ESC D, index
	c1NEL = 0x85 // ESC E, next line
	c1HTS = 0x88 // ESC H, set tab stop
	c1RI  = 0x8d // ESC M, reverse index
	c1SS2 = 0x8e // ESC N, single shift 2
	c1SS3 = 0x8f // ESC O, single shift 3
	c1DCS = 0x90 // ESC P
	c1SOS = 0x98 // ESC X
	c1CSI = 0x9b // ESC [
	c1ST  = 0x9c // ESC \
	c1OSC = 0x9d // ESC ]
	c1PM  = 0x9e // ESC ^
	c1APC = 0x9f // ESC _
)

func isC1(r rune) bool {
	return r >= 0x80 && r <= 0x9f
}

// processC1 runs a C1 control like its ESC form. Inside an OSC, DCS or APC
// only ST means something, it ends the string, other C1 controls are dropped
// there. The C1 controls that have no meaning here are dropped too, they are
// never printed.
func (t *Terminal) processC1(r rune) {
	// A C1 control cancels an ESC just before it
	t.escape = false

	switch t.state {
	case stateOSC:
		if r == c1ST {
			t.processOsc(r)
		}
		return
	case stateDCS:
		if r == c1ST {
			t.processDcs(r)
		}
		return
	case stateAPC:
		if r == c1ST {
			t.processApc(r)
		}
		return
	}

	switch r {
	case c1IND, c1NEL, c1HTS, c1RI, c1SS2, c1SS3, c1DCS, c1SOS, c1CSI, c1ST, c1OSC, c1PM, c1APC:
		t.parseEscState(r - 0x40)
	}
}
