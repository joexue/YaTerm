package terminal

func (t *Terminal) outputChar(r rune) {
	// The cursor stays on the last column after writing there, and only wraps
	// when the next character comes. Otherwise a "\r\n" right after a full
	// line (which ConPTY sends) would leave an extra blank line.
	if t.wrapPending {
		t.wrapPending = false
		t.cursorX = 0
		t.lineFeed()
	}

	t.screen.Project(r, t.cursorY, t.cursorX, t.fg, t.bg, t.style)
	if t.cursorX >= t.col-1 {
		t.wrapPending = true
	} else {
		t.cursorX += 1 // to fix for eastern characters
	}

	t.screen.MoveCursor(t.cursorY, t.cursorX)
}

// lineFeed moves the cursor down one line, scrolling at the bottom
func (t *Terminal) lineFeed() {
	t.cursorY += 1
	if t.row > 0 && t.cursorY >= t.row {
		t.screen.ScrollUp()
		t.cursorY = t.row - 1
	}
}

// moveCursor moves the cursor to row, col clamped to the screen
func (t *Terminal) moveCursor(row, col int) {
	if row >= t.row {
		row = t.row - 1
	}
	if row < 0 {
		row = 0
	}
	if col >= t.col {
		col = t.col - 1
	}
	if col < 0 {
		col = 0
	}

	t.cursorY, t.cursorX = row, col
	t.wrapPending = false
	t.screen.MoveCursor(t.cursorY, t.cursorX)
}

func (t *Terminal) saveCursor() {
	t.savedRow = t.cursorY
	t.savedCol = t.cursorX
}

func (t *Terminal) restoreCursor() {
	t.moveCursor(t.savedRow, t.savedCol)
}

func (t *Terminal) processGround(r rune) {
	switch r {
	// Ignore for now
	case asciiBell:
		return

	case asciiBackspace:
		t.moveCursor(t.cursorY, t.cursorX-1)
		return

	case asciiTab:
		// Move to the next tab stop (every 8 columns), stopping at the last column.
		t.moveCursor(t.cursorY, (t.cursorX/8+1)*8)
		return

	case asciiLineFeed, asciiVertTab, asciiFormFeed:
		// Line feed keeps the column, the tty or ConPTY sends "\r\n" for a new line
		t.wrapPending = false
		t.lineFeed()
		t.screen.MoveCursor(t.cursorY, t.cursorX)
		return

	case asciiReturn:
		t.moveCursor(t.cursorY, 0)
		return
	}

	t.outputChar(r)
}
