package terminal

func (t *Terminal) OutputChar(r rune) {
	if f := t.OnProject; f != nil {
		f(r, t.cursorY, t.cursorX, t.fg, t.bg)
		t.cursorX += 1 // to fix for eastern characters
		if t.cursorX > t.col {
			t.cursorX = 0
			t.cursorY += 1
		}

		if f := t.OnCursorMove; f != nil {
			f(t.cursorY, t.cursorX)
		}
	}
}

func (t *Terminal) ProcessGround(r rune) {

	moveCursor := t.OnCursorMove
	if moveCursor == nil {
		return
	}

	switch r {
	// Ignore for now
	case asciiBell:
		return

	case asciiBackspace:
		if t.cursorX == 0 {
			return
		} else {
			t.cursorX -= 1
			moveCursor(t.cursorY, t.cursorX)
		}
		return

	case asciiTab:
		w := t.row - t.cursorX
		if w > 8 {
			for i := 0; i < 8; i++ {
				t.OutputChar(' ')
			}
		} else {
			for i := 0; i < w; i++ {
				t.OutputChar(' ')
			}
		}
		return

	case asciiLineFeed, asciiVertTab, asciiFormFeed:
		t.cursorY = t.cursorY + 1
		t.cursorX = 0
		moveCursor(t.cursorY, 0)
		return

	case asciiReturn:
		moveCursor(t.cursorY, 0)
		return
	}

	t.OutputChar(r)
}
