package terminal

func (t *Terminal) ProcessCsi(r rune) {
	t.controlSequence += string(r)
	if (r < '0' || r > '9') && r != ';' && r != '=' && r != '?' && r != '>' {
		t.handleCsi()
		t.state = stateGROUND
	}
}

func (t *Terminal) handleColorMode(message string) {
	if message == "" || message == "0" {
		t.bg = nil
		t.fg = nil
		t.bold = false
		t.italic = false
		t.underline = false
		t.strikethrough = false
		t.blinking = false
		return
	}

	if message[0] == '>' || message[0] == '?' {
		return
	}
	/*
	   modes := strings.Split(message, ";")

	   	for i := 0; i < len(modes); i++ {
	   		mode := modes[i]
	   		if mode == "" {
	   			continue
	   		}

	   		if (mode == "38" || mode == "48") && i+1 < len(modes) {
	   			nextMode := modes[i+1]
	   			if nextMode == "5" && i+2 < len(modes) {
	   				t.handleColorModeMap(mode, modes[i+2])
	   				i += 2
	   			} else if nextMode == "2" && i+4 < len(modes) {
	   				t.handleColorModeRGB(mode, modes[i+2], modes[i+3], modes[i+4])
	   				i += 4
	   			}
	   		} else {
	   			t.handleColorMode(mode)
	   		}
	   	}
	*/
}

func (t *Terminal) handleCsi() {
	runes := []rune(t.controlSequence)
	t.controlSequence = ""

	if len(runes) < 2 {
		return
	}

	switch runes[len(runes)-1] {
	case 'm':
		t.handleColorMode(string(runes[0 : len(runes)-1]))
	case '@':
		//escapeInsertChars,
	case 'A':
	//escapeMoveCursorUp,
	case 'B':
	//escapeMoveCursorDown,
	case 'C':
	//escapeMoveCursorRight,
	case 'D':
	//escapeMoveCursorLeft,
	case 'd':
	//escapeMoveCursorRow,
	case 'H':
	//escapeMoveCursor,
	case 'f':
	//escapeMoveCursor,
	case 'G':
	//escapeMoveCursorCol,
	case 'h':
	//escapePrivateModeOn,
	case 'L':
	//escapeInsertLines,
	case 'l':
	//escapePrivateModeOff,
	case 'J':
	//escapeEraseInScreen,
	case 'K':
	//escapeEraseInLine,
	case 'P':
	//escapeDeleteChars,
	case 'r':
	//escapeSetScrollArea,
	case 's':
	//escapeSaveCursor,
	case 'S':
	//escapeScrollUp,
	case 'u':
	//escapeRestoreCursor,
	case 'i':
	//escapePrinterMode,
	case 'c':
	//escapeDeviceAttribute,
	case 'X':
	//escapeEraseChars,
	case 'M':
	//escapeDeleteLines,
	case 'T':
	//escapeScrollDown,
	case 'b':
	//escapeRepeatChar,
	case 't':
		//escapeWindowOps,
	}
}
