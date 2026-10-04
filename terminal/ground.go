package terminal

func (t *Terminal) ProcessGround(r rune) {
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
