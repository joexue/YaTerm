package ui

import (
	"image/color"
	"slices"
	"sync"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"yaterm/terminal"
)

var (
	_ fyne.Widget     = (*Screen)(nil)
	_ fyne.Scrollable = (*Screen)(nil)
	_ terminal.Screen = (*Screen)(nil)
)

// One mouse wheel notch is 25 units in Fyne (10 on macOS), scroll 3 lines a notch
const (
	scrollNotchUnits = 25
	scrollNotchLines = 3
)

// cell is one character on the screen, colors are already resolved for
// inverse/faint/hidden, a nil fg or bg means the theme default.
type cell struct {
	r      rune
	fg, bg color.Color
	style  fyne.TextStyle
}

// blank tells if the cell shows nothing, so it can be dropped from a line end
func (c cell) blank() bool {
	return (c.r == 0 || c.r == ' ') && c.bg == nil && !c.style.Underline && !c.style.Strikethrough
}

// line is one row of cells. wrapped means the text goes on in the next line
// because it reached the right edge (auto wrap), not because of a new line,
// so a resize can join them and wrap them again at the new width.
// Screen lines are always the screen width, history lines may be shorter as
// their blank end is dropped, but a wrapped line is always full width.
type line struct {
	cells   []cell
	wrapped bool
}

func newLine(col int) line {
	return line{cells: make([]cell, col)}
}

// sameText tells if two cells can be drawn by the same text object, the
// background is merged separately so it does not split a text run.
func (c cell) sameText(o cell) bool {
	return c.fg == o.fg && c.style == o.style
}

// screenLine draws one row. Neighbor cells with the same background share one
// rectangle, and those with the same text color and style share one text object.
// The objects are pooled and reused between redraws, and only the ones whose
// content changed are refreshed so the others keep their rendered texture.
type screenLine struct {
	box   *fyne.Container
	rects []*canvas.Rectangle
	texts []*canvas.Text

	// Reused between redraws to avoid allocating
	next   []fyne.CanvasObject
	buf    []byte
	selBuf []cell
}

func newScreenLine() *screenLine {
	return &screenLine{box: container.NewWithoutLayout()}
}

func (l *screenLine) rect(i int) *canvas.Rectangle {
	if i == len(l.rects) {
		l.rects = append(l.rects, canvas.NewRectangle(color.Transparent))
	}
	return l.rects[i]
}

func (l *screenLine) text(i int) *canvas.Text {
	if i == len(l.texts) {
		l.texts = append(l.texts, canvas.NewText("", color.Transparent))
	}
	return l.texts[i]
}

type Screen struct {
	widget.BaseWidget

	cursor *canvas.Rectangle
	// The character under the cursor, drawn on the block in the background
	// color so the block does not hide it
	cursorText *canvas.Text

	// mu guards the cell buffer, written directly by the terminal goroutine
	// and read on the UI thread when a flush renders it.
	mu       sync.Mutex
	lines    []line
	dirty    []bool
	row, col int
	scrolled int // ScrollUp calls not yet applied to rows
	flushing bool
	curRow   int
	curCol   int

	// Lines scrolled off the top, offset is how many lines the view is
	// scrolled back into them, 0 shows the live screen.
	hist      *history
	offset    int
	viewDirty bool

	// histTotal counts every line ever pushed into the history, so a line has
	// a fixed number (its absolute line) while the history and view scroll.
	// The selection uses it to stay on the same text.
	histTotal int
	sel       selection

	// UI thread only
	rows      []*screenLine
	objects   []fyne.CanvasObject
	cellSize  fyne.Size
	scrollAcc float32

	size fyne.Size
}

func NewScreen() *Screen {
	s := &Screen{
		cursor:     canvas.NewRectangle(theme.Color(theme.ColorNamePrimary)),
		cursorText: canvas.NewText("", color.Transparent),
		cellSize:   measureCell(),
		hist:       newHistory(HistoryLines),
	}

	s.cursorText.Hide()
	s.objects = []fyne.CanvasObject{s.cursor, s.cursorText}
	s.ExtendBaseWidget(s)

	return s
}

func (s *Screen) Project(r rune, row, col int, fg, bg color.Color, style terminal.Style) {
	textStyle := fyne.TextStyle{
		Monospace: true,

		Bold:          style.Bold,
		Italic:        style.Italic,
		Underline:     style.Underline,
		Strikethrough: style.Strikethrough,
	}

	// Inverse, faint and hidden need real colors, so fill in the theme defaults
	if style.Inverse || style.Faint || style.Hidden {
		if fg == nil {
			fg = theme.Color(theme.ColorNameForeground)
		}
		if bg == nil {
			bg = theme.Color(theme.ColorNameBackground)
		}
		if style.Inverse {
			fg, bg = bg, fg
		}
		if style.Faint {
			fg = blend(fg, bg)
		}
		if style.Hidden {
			fg = bg
		}
	}

	c := cell{r: r, fg: fg, bg: bg, style: textStyle}

	s.mu.Lock()
	defer s.mu.Unlock()

	// The terminal may still use the old size right after a resize
	if row < 0 || row >= s.row || col < 0 || col >= s.col {
		return
	}

	s.lines[row].cells[col] = c
	s.dirty[row] = true
}

// SetWrapped marks a row as going on in the next one, called by the terminal
// when it wraps at the right edge, and cleared when the line end is erased.
func (s *Screen) SetWrapped(row int, wrapped bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if row >= 0 && row < len(s.lines) {
		s.lines[row].wrapped = wrapped
	}
}

// Resize only sizes the widget, the rows and columns come from the terminal
// through Reflow, which Pane calls before laying out the screen.
func (s *Screen) Resize(size fyne.Size) {
	if size.Width <= 0 || size.Height <= 0 || size == s.size {
		return
	}

	s.size = size
	s.cellSize = measureCell()
	s.cursor.Resize(s.cellSize)
	s.BaseWidget.Resize(size)

	s.mu.Lock()
	s.render()
	s.updateCursor()
	s.mu.Unlock()
}

// Reflow fits the history and the screen to row x col. Lines that were auto
// wrapped are joined again and wrapped at the new width, so a narrower screen
// moves the line end to the next line instead of cutting it, and a wider one
// merges it back. The screen keeps the cursor line (or the last line with
// content) at the bottom, lines above go to the history. Returns the new
// cursor position.
func (s *Screen) Reflow(row, col, curRow, curCol int) (int, int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	defer func() {
		// Everything is redrawn, so pending scrolls no longer matter
		s.dirty = make([]bool, s.row)
		for i := range s.dirty {
			s.dirty[i] = true
		}
		s.scrolled = 0
		s.viewDirty = true
		s.offset = min(s.offset, s.hist.len())
	}()

	if row <= 0 || col <= 0 || len(s.lines) == 0 || s.col <= 0 {
		// Nothing to keep yet
		s.row, s.col = max(row, 0), max(col, 0)
		s.lines = make([]line, s.row)
		for i := range s.lines {
			s.lines[i] = newLine(s.col)
		}
		return max(0, min(curRow, s.row-1)), max(0, min(curCol, s.col-1))
	}

	curRow = max(0, min(curRow, len(s.lines)-1))
	curCol = max(0, min(curCol, s.col-1))

	// Blank lines below the cursor are just empty screen, not content
	last := curRow
	for i := len(s.lines) - 1; i > curRow; i-- {
		if s.lines[i].wrapped || !lineBlank(s.lines[i].cells) {
			last = i
			break
		}
	}

	// All lines in order, history first
	nHist := s.hist.len()
	phys := make([]line, 0, nHist+last+1)
	for i := 0; i < nHist; i++ {
		phys = append(phys, s.hist.at(i))
	}
	phys = append(phys, s.lines[:last+1]...)
	cursorPhys := nHist + curRow

	out := make([]line, 0, len(phys))
	newRow, newCol := 0, 0

	for i := 0; i < len(phys); {
		// A logical line is a run of wrapped lines and the line ending it
		j := i
		for j < len(phys)-1 && phys[j].wrapped {
			j++
		}

		var cells []cell
		if i == j {
			cells = phys[i].cells // no copy for the common single line
		} else {
			for k := i; k <= j; k++ {
				cells = append(cells, phys[k].cells...)
			}
		}

		// The cursor as an offset into the logical line
		curOff := -1
		if cursorPhys >= i && cursorPhys <= j {
			curOff = (cursorPhys-i)*s.col + curCol
		}

		// Drop the blank end, but keep it up to the cursor
		n := len(cells)
		for n > 0 && cells[n-1].blank() {
			n--
		}
		if curOff > n {
			n = min(curOff, len(cells))
		}

		// Wrap at the new width, the pieces share the logical line's cells.
		// Full slice expressions stop a later append from spilling over.
		start := len(out)
		for off := 0; off == 0 || off < n; off += col {
			end := min(off+col, n)
			out = append(out, line{cells: cells[off:end:end], wrapped: end < n})
		}

		if curOff >= 0 {
			newRow, newCol = start+curOff/col, curOff%col
			// The cursor is right after a line that fills the width
			for newRow >= len(out) {
				out = append(out, line{})
			}
		}

		i = j + 1
	}

	// The screen ends with the kept lines, but the cursor must stay on it
	top := max(0, len(out)-row)
	top = min(top, newRow)

	s.hist.reset()
	// Lines are wrapped again, the old line numbers mean nothing
	s.sel = selection{}
	for _, l := range out[max(0, top-HistoryLines):top] {
		s.hist.push(l)
	}
	s.histTotal = s.hist.len()

	// Screen lines are full width as the terminal writes into them
	s.row, s.col = row, col
	s.lines = make([]line, row)
	for i := range s.lines {
		s.lines[i] = newLine(col)
		if top+i < len(out) {
			copy(s.lines[i].cells, out[top+i].cells)
			s.lines[i].wrapped = out[top+i].wrapped
		}
	}

	return newRow - top, min(newCol, col-1)
}

// lineBlank tells if a line shows nothing
func lineBlank(cells []cell) bool {
	for _, c := range cells {
		if !c.blank() {
			return false
		}
	}
	return true
}

// syncRows makes one row object for each screen row. Called on the UI thread
// with mu held, after Reflow changed the number of rows.
func (s *Screen) syncRows() {
	if len(s.rows) == s.row && len(s.objects) == s.row+2 {
		return
	}

	for len(s.rows) < s.row {
		s.rows = append(s.rows, newScreenLine())
	}
	s.rows = s.rows[:s.row]

	// Rows first, the cursor is drawn on top
	s.objects = make([]fyne.CanvasObject, 0, s.row+2)
	for _, l := range s.rows {
		s.objects = append(s.objects, l.box)
	}
	s.objects = append(s.objects, s.cursor, s.cursorText)

	s.layoutRows()
	canvas.Refresh(s)
}

func (s *Screen) PositionForCursorLocation(row, col int) fyne.Position {
	return fyne.NewPos(float32(col)*s.cellSize.Width, float32(row)*s.cellSize.Height)
}

// Flush asks the UI thread to draw what changed. If a flush is already queued
// it is not queued again, the queued one draws the latest state, so heavy
// output costs one fyne.Do per frame instead of one per read.
func (s *Screen) Flush(row, col int) {
	s.mu.Lock()
	s.curRow, s.curCol = row, col
	if s.flushing {
		s.mu.Unlock()
		return
	}
	s.flushing = true
	s.mu.Unlock()

	fyne.Do(func() {
		s.mu.Lock()
		defer s.mu.Unlock()

		s.flushing = false
		s.render()
		s.updateCursor()
	})
}

// updateCursor places the cursor, it moves down with the view when scrolled
// back and is hidden when it is below the view. Called with mu held.
func (s *Screen) updateCursor() {
	row := s.curRow + s.offset
	if row >= s.row {
		s.cursor.Hide()
		s.cursorText.Hide()
		return
	}

	pos := fyne.NewPos(float32(s.curCol)*s.cellSize.Width, float32(row)*s.cellSize.Height)
	s.cursor.Move(pos)
	s.cursor.Show()

	// Draw the character under the cursor inverted: in its background color,
	// or the theme background when it has none, on top of the block
	var c cell
	if s.curRow >= 0 && s.curRow < len(s.lines) && s.curCol >= 0 && s.curCol < len(s.lines[s.curRow].cells) {
		c = s.lines[s.curRow].cells[s.curCol]
	}
	if c.r == 0 || c.r == ' ' {
		s.cursorText.Hide()
		return
	}

	fg := c.bg
	if fg == nil {
		fg = theme.Color(theme.ColorNameBackground)
	}
	text := s.cursorText
	if text.Text != string(c.r) || text.Color != fg || text.TextStyle != c.style || text.TextSize != theme.TextSize() {
		text.Text = string(c.r)
		text.Color = fg
		text.TextStyle = c.style
		text.TextSize = theme.TextSize()
		text.Refresh()
	}
	text.Move(pos)
	text.Resize(s.cellSize)
	text.Show()
}

// Scrolled shows the history lines when the mouse wheel scrolls up
func (s *Screen) Scrolled(ev *fyne.ScrollEvent) {
	s.scrollAcc += ev.Scrolled.DY
	// Multiply first so a whole notch is exact, 25/3 is not exact in float
	n := int(s.scrollAcc * scrollNotchLines / scrollNotchUnits)
	if n == 0 {
		return
	}
	s.scrollAcc -= float32(n) * scrollNotchUnits / scrollNotchLines

	s.mu.Lock()
	defer s.mu.Unlock()

	s.scrollTo(s.offset + n)
}

// ScrollLines scrolls the view n lines back into the history, forward when
// n < 0, e.g. while a selection is dragged past the top or bottom
func (s *Screen) ScrollLines(n int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.scrollTo(s.offset + n)
}

// ScrollToBottom goes back to the live screen, e.g. when a key is typed
func (s *Screen) ScrollToBottom() {
	s.scrollAcc = 0

	s.mu.Lock()
	defer s.mu.Unlock()

	s.scrollTo(0)
}

// scrollTo moves the view offset lines back into the history. Called on the
// UI thread with mu held.
func (s *Screen) scrollTo(offset int) {
	offset = max(0, min(offset, s.hist.len()))
	if offset == s.offset {
		return
	}

	s.offset = offset
	s.viewDirty = true
	s.render()
	s.updateCursor()
}

func (s *Screen) MoveCursor(row, col int) {
}

// ScrollUp rotates the buffer right away, the row objects are rotated on the
// next render so rows that are already drawn are just moved up.
func (s *Screen) ScrollUp() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.row <= 0 {
		return
	}

	last := s.row - 1

	// Move the top line into the history, reuse the line it drops if any
	top := s.lines[0]
	copy(s.lines, s.lines[1:])
	copy(s.dirty, s.dirty[1:])
	blank := s.hist.push(top)
	s.histTotal++
	if len(blank.cells) == s.col {
		clear(blank.cells)
		blank.wrapped = false
	} else {
		blank = newLine(s.col)
	}
	s.lines[last] = blank
	s.dirty[last] = true

	if s.offset > 0 {
		// Keep the view still while new output comes in
		s.offset = min(s.offset+1, s.hist.len())
		s.viewDirty = true
	} else {
		s.scrolled++
	}
}

// ScrollRegion scrolls rows top..bottom by n lines, up when n > 0 and down
// when n < 0. Lines leaving the region are dropped, not kept in the history,
// as it is part of the screen like vim's text area above its status line.
// Insert and delete lines are this too, from the cursor row to the bottom.
func (s *Screen) ScrollRegion(top, bottom, n int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	top, bottom = max(top, 0), min(bottom, s.row-1)
	if top >= bottom || n == 0 {
		return
	}

	region := s.lines[top : bottom+1]
	k := min(abs(n), len(region))

	// Rotate, the lines that went out come back blank on the other side
	if n > 0 {
		slices.Reverse(region[:k])
		slices.Reverse(region[k:])
		slices.Reverse(region)
		for i := len(region) - k; i < len(region); i++ {
			clearLine(&region[i])
		}
	} else {
		slices.Reverse(region)
		slices.Reverse(region[:k])
		slices.Reverse(region[k:])
		for i := 0; i < k; i++ {
			clearLine(&region[i])
		}
	}

	for i := top; i <= bottom; i++ {
		s.dirty[i] = true
	}
	if s.offset > 0 {
		s.viewDirty = true
	}
}

// InsertCells shifts the cells from col right by n, the ones pushed past the
// line end are dropped, the gap gets blank cells with background bg.
func (s *Screen) InsertCells(row, col, n int, bg color.Color) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if row < 0 || row >= s.row || col < 0 || col >= s.col || n <= 0 {
		return
	}

	cells := s.lines[row].cells
	n = min(n, s.col-col)
	copy(cells[col+n:], cells[col:])
	for i := col; i < col+n; i++ {
		cells[i] = cell{bg: bg}
	}
	s.dirty[row] = true
}

// DeleteCells removes n cells from col, the rest of the line moves left and
// blank cells with background bg fill the end.
func (s *Screen) DeleteCells(row, col, n int, bg color.Color) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if row < 0 || row >= s.row || col < 0 || col >= s.col || n <= 0 {
		return
	}

	cells := s.lines[row].cells
	n = min(n, s.col-col)
	copy(cells[col:], cells[col+n:])
	for i := s.col - n; i < s.col; i++ {
		cells[i] = cell{bg: bg}
	}
	s.dirty[row] = true
}

// clearLine blanks a line in place
func clearLine(l *line) {
	clear(l.cells)
	l.wrapped = false
}

// viewLine returns the line shown on row i of the view. Called with mu held.
func (s *Screen) viewLine(i int) []cell {
	n := s.hist.len()
	idx := n - s.offset + i
	if idx < n {
		return s.hist.at(idx).cells
	}
	return s.lines[idx-n].cells
}

// render rebuilds the objects of the rows changed since the last render.
// Called on the UI thread with mu held.
func (s *Screen) render() {
	s.syncRows()

	if s.offset > 0 || s.viewDirty {
		// The view is scrolled back or just moved, redraw every row. Rows that
		// look the same are cheap, their objects are not refreshed.
		s.scrolled = 0
		s.viewDirty = false
		for i := range s.rows {
			s.renderLine(i, s.viewLine(i), s.histTotal-s.offset+i)
			s.dirty[i] = false
		}
		return
	}

	if n := len(s.rows); n > 0 && s.scrolled > 0 {
		// Each scroll moved the top row to the bottom, do the same to the
		// row objects, the rows that came to the bottom are marked dirty.
		k := s.scrolled % n
		slices.Reverse(s.rows[:k])
		slices.Reverse(s.rows[k:])
		slices.Reverse(s.rows)
		s.scrolled = 0
		s.layoutRows()
	}

	for i, d := range s.dirty {
		if d {
			s.renderLine(i, s.lines[i].cells, s.histTotal+i)
			s.dirty[i] = false
		}
	}
}

func (s *Screen) renderLine(i int, cells []cell, abs int) {
	l := s.rows[i]
	cells = s.highlight(l, cells, abs)
	// History lines may be shorter than the screen, never wider after Reflow
	cells = cells[:min(len(cells), s.col)]
	w, h := s.cellSize.Width, s.cellSize.Height
	fgDefault := theme.Color(theme.ColorNameForeground)
	textSize := theme.TextSize()

	objs := l.next[:0]

	// Backgrounds, one rectangle for each stretch of the same color, so no
	// seam shows where only the text color or style changes.
	nRects := 0
	for start := 0; start < len(cells); {
		bg := cells[start].bg
		end := start + 1
		for end < len(cells) && cells[end].bg == bg {
			end++
		}

		if bg != nil {
			rect := l.rect(nRects)
			nRects++
			if rect.FillColor != bg {
				rect.FillColor = bg
				rect.Refresh()
			}

			// Fyne rounds a rectangle's position and size to device pixels
			// separately, so with a fractional cell width its end can fall
			// one pixel short of where the next one starts, leaving a gap.
			// When another background follows, reach 1 unit into it, it is
			// drawn later and covers the overlap. Not when none follows, the
			// color would bleed into the next cell.
			width := float32(end-start) * w
			if end < len(cells) && cells[end].bg != nil {
				width++
			}

			// Move and Resize repaint only when the value changes
			rect.Move(fyne.NewPos(float32(start)*w, 0))
			rect.Resize(fyne.NewSize(width, h))
			objs = append(objs, rect)
		}

		start = end
	}

	nTexts := 0
	for start := 0; start < len(cells); {
		c := cells[start]

		// Only ASCII is merged, other runes may be drawn by a fallback font
		// with a different width, so they get their own text to stay on the grid.
		end := start + 1
		if c.r < 0x80 {
			for end < len(cells) && cells[end].r < 0x80 && cells[end].sameText(c) {
				end++
			}
		}

		buf := l.buf[:0]
		blank := true
		for _, cc := range cells[start:end] {
			r := cc.r
			if r == 0 {
				r = ' '
			}
			if r != ' ' {
				blank = false
			}
			buf = utf8.AppendRune(buf, r)
		}
		l.buf = buf

		// Spaces only need drawing when a line goes through them
		if !blank || c.style.Underline || c.style.Strikethrough {
			fg := c.fg
			if fg == nil {
				fg = fgDefault
			}

			text := l.text(nTexts)
			nTexts++

			// string(buf) in a comparison does not allocate, the new string
			// is only made when the text really changed.
			if text.Text != string(buf) || text.Color != fg || text.TextStyle != c.style || text.TextSize != textSize {
				text.Text = string(buf)
				text.Color = fg
				text.TextStyle = c.style
				text.TextSize = textSize
				text.Refresh()
			}
			text.Move(fyne.NewPos(float32(start)*w, 0))
			text.Resize(fyne.NewSize(float32(end-start)*w, h))
			objs = append(objs, text)
		}

		start = end
	}

	// Rects were appended before texts, so no text is covered by a later
	// run's background. The container is only refreshed when its object list
	// changed, not its children, which refresh themselves above when needed.
	if !slices.Equal(objs, l.box.Objects) {
		l.next, l.box.Objects = l.box.Objects, objs
		canvas.Refresh(l.box)
	} else {
		l.next = objs
	}
}

func (s *Screen) layoutRows() {
	size := fyne.NewSize(float32(s.col)*s.cellSize.Width, s.cellSize.Height)
	for i, l := range s.rows {
		l.box.Move(fyne.NewPos(0, float32(i)*s.cellSize.Height))
		l.box.Resize(size)
	}
}

func (s *Screen) CreateRenderer() fyne.WidgetRenderer {
	r := &screenRenderer{
		screen: s,
	}

	return r
}

var _ fyne.WidgetRenderer = (*screenRenderer)(nil)

type screenRenderer struct {
	screen *Screen
}

func (r *screenRenderer) Destroy() {
}

func (r *screenRenderer) Layout(size fyne.Size) {
	r.screen.mu.Lock()
	defer r.screen.mu.Unlock()

	r.screen.layoutRows()
}

// MinSize is one cell and does not follow the content, otherwise the parent
// Pane would be laid out again on output.
func (r *screenRenderer) MinSize() fyne.Size {
	return r.screen.cellSize
}

func (r *screenRenderer) Objects() []fyne.CanvasObject {
	return r.screen.objects
}

// Refresh redraws everything, e.g. after a theme change
func (r *screenRenderer) Refresh() {
	s := r.screen
	s.cellSize = measureCell()
	s.cursor.FillColor = theme.Color(theme.ColorNamePrimary)
	s.cursor.Resize(s.cellSize)
	s.cursor.Refresh()

	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.dirty {
		s.dirty[i] = true
	}
	s.render()
	s.layoutRows()
	s.updateCursor()
}

/*
 * ================================
 *  History
 * ================================
 */

// HistoryLines is the maximum number of lines kept after they scroll off the screen
const HistoryLines = 2000

// history is a ring buffer of the lines scrolled off the top of the screen
type history struct {
	lines []line
	head  int // index of the oldest line
	count int
}

func newHistory(max int) *history {
	return &history{lines: make([]line, max)}
}

// push adds a line as the newest. When full the oldest line is dropped and
// returned so its memory can be reused, otherwise it returns an empty line.
func (h *history) push(l line) line {
	max := len(h.lines)
	if max == 0 {
		return l
	}

	if h.count < max {
		h.lines[(h.head+h.count)%max] = l
		h.count++
		return line{}
	}

	old := h.lines[h.head]
	h.lines[h.head] = l
	h.head = (h.head + 1) % max
	return old
}

// at returns the i-th line, 0 is the oldest
func (h *history) at(i int) line {
	return h.lines[(h.head+i)%len(h.lines)]
}

func (h *history) len() int {
	return h.count
}

// reset empties the history, the lines are released for the GC
func (h *history) reset() {
	clear(h.lines)
	h.head, h.count = 0, 0
}
