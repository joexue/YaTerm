package terminal

func (t *Terminal) ProcessOsc(r rune) {
	if r == asciiBell || r == asciiNull {
		t.handleOSC()
		t.state = GROUND
	} else {
		t.oscCode += string(r)
	}
}

func (t *Terminal) handleOSC() {
	/*
		runes := []rune(t.oscCode)
		t.oscCode = ""
	*/
}

func (t *Terminal) handleOSCMode(mode string, code string) {
	switch mode {
	case "0":
	case "1":
	case "2":
	}
}
