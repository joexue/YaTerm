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
}

func (t *Terminal) handleOSCMode(mode string, code string) {
	switch mode {
	case "0":
	case "1":
	case "2":
	}
}
