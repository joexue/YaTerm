package terminal

import "github.com/mattn/go-runewidth"

// Cells that hold no character of their own:
// WideTail is the second cell of a wide character (e.g. Chinese), which takes
// two columns and is drawn over both cells. WidePad is the last column left
// blank when a wide character did not fit there and went to the next line,
// it is not text, joining the lines again on a resize drops it.
const (
	WideTail rune = -1
	WidePad  rune = -2
)

// widthCond measures characters like the C library's wcwidth that shells and
// vim use. Ambiguous ones like box drawing are 1 wide, even in a CJK locale,
// where go-runewidth would make them 2 by default.
var widthCond = func() *runewidth.Condition {
	c := runewidth.NewCondition()
	c.EastAsianWidth = false
	return c
}()

func (t *Terminal) outputChar(r rune) {
	r = t.translateChar(r)
	w := widthCond.RuneWidth(r)
	if w == 0 {
		// Combining marks and other zero width characters are not supported
		return
	}
	if w > 1 && t.col < 2 {
		w = 1
	}

	// A wide character that does not fit in the last column goes to the next
	// line, leaving that column blank
	if w == 2 && !t.wrapPending && t.cursorX >= t.col-1 {
		t.screen.Project(WidePad, t.cursorY, t.cursorX, nil, t.bg, Style{})
		t.wrapPending = true
	}

	// The cursor stays on the last column after writing there, and only wraps
	// when the next character comes. Otherwise a "\r\n" right after a full
	// line (which ConPTY sends) would leave an extra blank line.
	if t.wrapPending {
		t.wrapPending = false
		// The line goes on in the next one, so a resize can join them again
		t.screen.SetWrapped(t.cursorY, true)
		t.cursorX = 0
		t.lineFeed()
	}

	t.screen.Project(r, t.cursorY, t.cursorX, t.fg, t.bg, t.style)
	if w == 2 {
		t.screen.Project(WideTail, t.cursorY, t.cursorX+1, t.fg, t.bg, t.style)
	}

	if t.cursorX+w >= t.col {
		t.cursorX = t.col - 1
		t.wrapPending = true
	} else {
		t.cursorX += w
	}

	t.screen.MoveCursor(t.cursorY, t.cursorX)
}

// regionBottom is the last row of the scroll region
func (t *Terminal) regionBottom() int {
	if t.scrollBottom <= 0 || t.scrollBottom >= t.row {
		return t.row - 1
	}
	return t.scrollBottom
}

// scrollRegion scrolls the scroll region up by n lines, down when n < 0.
// Only a full screen scroll up keeps the lines in the history.
func (t *Terminal) scrollRegion(n int) {
	bottom := t.regionBottom()
	if t.scrollTop == 0 && bottom == t.row-1 && n > 0 {
		for i := 0; i < min(n, t.row); i++ {
			t.screen.ScrollUp()
		}
		return
	}
	t.screen.ScrollRegion(t.scrollTop, bottom, n)
}

// lineFeed moves the cursor down one line. At the bottom of the scroll region
// the region scrolls up, below the region it stops at the last row.
func (t *Terminal) lineFeed() {
	switch {
	case t.cursorY == t.regionBottom():
		t.scrollRegion(1)
	case t.cursorY < t.row-1:
		t.cursorY += 1
	}
}

// reverseIndex moves the cursor up one line, at the top of the scroll region
// the region scrolls down
func (t *Terminal) reverseIndex() {
	switch {
	case t.cursorY == t.scrollTop:
		t.scrollRegion(-1)
	case t.cursorY > 0:
		t.cursorY -= 1
	}
}

// setScrollRegion handles "CSI top;bottom r", the cursor goes home
func (t *Terminal) setScrollRegion(top, bottom int) {
	top, bottom = top-1, bottom-1
	if bottom <= 0 || bottom >= t.row {
		bottom = t.row - 1
	}
	if top < 0 {
		top = 0
	}
	if top >= bottom {
		return
	}

	t.scrollTop, t.scrollBottom = top, bottom
	t.moveCursor(0, 0)
}

// insertLines and deleteLines work from the cursor row to the bottom of the
// scroll region, nothing happens when the cursor is outside it
func (t *Terminal) insertLines(n int) {
	if t.cursorY < t.scrollTop || t.cursorY > t.regionBottom() {
		return
	}
	t.screen.ScrollRegion(t.cursorY, t.regionBottom(), -n)
	t.moveCursor(t.cursorY, 0)
}

func (t *Terminal) deleteLines(n int) {
	if t.cursorY < t.scrollTop || t.cursorY > t.regionBottom() {
		return
	}
	t.screen.ScrollRegion(t.cursorY, t.regionBottom(), n)
	t.moveCursor(t.cursorY, 0)
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
	t.savedCharsets, t.savedShifted = t.charsets, t.shifted
}

func (t *Terminal) restoreCursor() {
	t.moveCursor(t.savedRow, t.savedCol)
	t.charsets, t.shifted = t.savedCharsets, t.savedShifted
}

func (t *Terminal) processGround(r rune) {
	switch r {
	// Switch between the G0 and G1 character sets
	case asciiShiftIn:
		t.shifted = 0
		return
	case asciiShiftOut:
		t.shifted = 1
		return

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
