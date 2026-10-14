package terminal

func (t *Terminal) handleApc() {
}

// processApc reads an APC ("ESC _ ... ESC \"), and the SOS ("ESC X") and PM
// ("ESC ^") strings, which are handled the same: their content is ignored.
// APC carries the kitty graphics protocol ("ESC _ G ... ESC \"), images are
// not supported, and a program asking for them gets no reply, which is how
// a terminal says it has no support.
//
// It ends with ST, the 7 bit "ESC \" comes through the '\\' case of
// parseEscState, an ESC with anything else aborts it (see parseEscState).
// CAN and SUB cancel it.
func (t *Terminal) processApc(r rune) {
	switch r {
	case StringTerminator:
		t.handleApc()
		t.param = nil
		t.state = stateGROUND

	case asciiCancel, asciiSubstitute: //CAN, SUB
		t.param = nil
		t.state = stateGROUND

	default:
		t.param = append(t.param, r)
	}
}
