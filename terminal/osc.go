package terminal

func (t *Terminal) ProcessOsc(r rune) {
	if r == asciiBell || r == asciiNull || r == '\\' {
		t.handleOSC()
		t.state = stateGROUND
	} else {
		t.controlSequence += string(r)
	}
}

func (t *Terminal) handleOSC() {
	// The buffer is shared with CSI, so leftover title text would corrupt the next sequence
	t.controlSequence = ""
}

func (t *Terminal) handleOSCMode(mode string, code string) {
	switch mode {
	case "0":
	case "1":
	case "2":
	}
}
