package terminal

func (t *Terminal) OutputChar(r rune) {
	t.Screen.Project(r, t.cursorY, t.cursorX, t.fg, t.bg)
	t.cursorX += 1 // to fix for eastern characters
	if t.cursorX >= t.col {
		t.cursorX = 0
		t.cursorY += 1
		if t.cursorY == t.row {
			t.Screen.ScrollUp()
			t.cursorY = t.row - 1
		}
	}

	t.Screen.MoveCursor(t.cursorY, t.cursorX)
}

func (t *Terminal) ProcessGround(r rune) {
	switch r {
	// Ignore for now
	case asciiBell:
		return

	case asciiBackspace:
		if t.cursorX == 0 {
			return
		} else {
			t.cursorX -= 1
			t.Screen.MoveCursor(t.cursorY, t.cursorX)
		}
		return

	case asciiTab:
		// Advance to the next tab stop (every 8 columns), stopping at the last column.
		w := 8 - t.cursorX%8
		if rest := t.col - 1 - t.cursorX; w > rest {
			w = rest
		}
		for i := 0; i < w; i++ {
			t.OutputChar(' ')
		}
		return

	case asciiLineFeed, asciiVertTab, asciiFormFeed:
		t.cursorY = t.cursorY + 1
		t.cursorX = 0
		if t.cursorY == t.row {
			t.Screen.ScrollUp()
			t.cursorY = t.row - 1
		}
		t.Screen.MoveCursor(t.cursorY, t.cursorX)
		return

	case asciiReturn:
		t.cursorX = 0
		t.Screen.MoveCursor(t.cursorY, t.cursorX)
		return
	}

	t.OutputChar(r)
}
