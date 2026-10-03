package terminal

import (
	"fmt"
	"unicode/utf8"
)

const (
	asciiBell      = 7
	asciiBackspace = 8
	asciiEscape    = 27

	noEscape = 5000
	tabWidth = 8
)

const (
	GROUND = iota
	ESC
	CSI
	OSC
	APC
	DSC
	VT100
)

func (t *Terminal) parseEscape(r rune) {
	//t.stateCode += string(r)
	if (r < '0' || r > '9') && r != ';' && r != '=' && r != '?' && r != '>' {
		/*
			code := t.state.code
			fyne.Do(func() {
				t.handleEscape(code)
			})
			t.state.code = ""
		*/
		t.state = GROUND
	}
}

func (t *Terminal) parseEscState(r rune) {
	switch r {
	case '[':
		t.state = ESC
	case '\\':
		if t.state == OSC {
			/*
				code := t.state.code
				fyne.Do(func() {
					t.handleOSC(code)
				})
			*/
		}
		//t.state.code = ""
		t.state = GROUND
	case ']':
		t.state = OSC
	case '(', ')':
		t.state = VT100
	case '7':
		t.savedRow = t.cursorY
		t.savedCol = t.cursorX
	case '8':
		t.cursorX = t.savedRow
		t.cursorY = t.savedCol
	case 'D':
		//t.scrollDown()
	case 'M':
		//t.scrollUp()
	case 'P':
		t.state = DSC
	case '_':
		t.state = APC
	case '=', '>':
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

		fmt.Println("xxxxx state:", t.state, r, string(r))

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
		case ESC:
			t.parseEscape(r)
			//t.parseEscState(r)
			continue
		case CSI:
			continue
		case OSC:
			continue
		case APC:
			continue
		case VT100:
			continue
		}

		if f := t.OnProject; f != nil {
			f(r, t.cursorY, t.cursorX, t.fg, t.bg)
			t.cursorX += size
		}
	}
}
