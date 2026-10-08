package ui

import (
	"image/color"
	"math"
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
	next []fyne.CanvasObject
	buf  []byte
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

	// mu guards the cell buffer, written directly by the terminal goroutine
	// and read on the UI thread when a flush renders it.
	mu       sync.Mutex
	lines    [][]cell
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

	// UI thread only
	rows      []*screenLine
	objects   []fyne.CanvasObject
	cellSize  fyne.Size
	scrollAcc float32

	size fyne.Size
}

func NewScreen() *Screen {
	s := &Screen{
		cursor:   canvas.NewRectangle(theme.Color(theme.ColorNamePrimary)),
		cellSize: measureCell(1),
		hist:     newHistory(HistoryLines),
	}

	s.objects = []fyne.CanvasObject{s.cursor}
	s.ExtendBaseWidget(s)

	return s
}

// measureCell keeps the font's real advance width so a merged run of text
// lines up with the cell grid. The height is rounded to whole device pixels,
// not Fyne units, so rows stack without seams on a scaled (e.g. 125%) display.
func measureCell(scale float32) fyne.Size {
	size := fyne.MeasureText("M", theme.TextSize(), fyne.TextStyle{Monospace: true})
	size.Height = float32(math.Round(float64(size.Height*scale))) / scale
	return size
}

// scale returns the scale of the canvas the screen is on. Before the screen is
// first drawn it has no canvas yet, so fall back to the window's canvas.
func (s *Screen) scale() float32 {
	app := fyne.CurrentApp()
	if app == nil {
		return 1
	}

	if c := app.Driver().CanvasForObject(s); c != nil {
		return c.Scale()
	}

	if windows := app.Driver().AllWindows(); len(windows) > 0 {
		return windows[0].Canvas().Scale()
	}

	return 1
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

	s.lines[row][col] = c
	s.dirty[row] = true
}

func (s *Screen) Resize(size fyne.Size) {
	if size.Width <= 0 || size.Height <= 0 || size == s.size {
		return
	}

	s.size = size
	// Keep row/col consistent with what Pane hands to the terminal.
	row, col := s.CursorLocationForPosition(fyne.NewPos(size.Width, size.Height))

	s.mu.Lock()
	s.row, s.col = row, col
	s.resizeBuffer()
	s.mu.Unlock()

	s.cursor.Resize(s.cellSize)
	s.BaseWidget.Resize(size)

	s.mu.Lock()
	s.render()
	s.mu.Unlock()
}

// resizeBuffer fits the cell buffer and the row objects to s.row x s.col,
// keeping the content that is still on screen. Called with mu held.
func (s *Screen) resizeBuffer() {
	lines := make([][]cell, s.row)
	for i := range lines {
		lines[i] = make([]cell, s.col)
		if i < len(s.lines) {
			copy(lines[i], s.lines[i])
		}
	}
	s.lines = lines

	for len(s.rows) < s.row {
		s.rows = append(s.rows, newScreenLine())
	}
	s.rows = s.rows[:s.row]

	// Everything is redrawn, so pending scrolls no longer matter
	s.dirty = make([]bool, s.row)
	for i := range s.dirty {
		s.dirty[i] = true
	}
	s.scrolled = 0
	s.offset = min(s.offset, s.hist.len())

	// Rows first, the cursor is drawn on top
	s.objects = make([]fyne.CanvasObject, 0, s.row+1)
	for _, l := range s.rows {
		s.objects = append(s.objects, l.box)
	}
	s.objects = append(s.objects, s.cursor)
}

// CursorLocationForPosition also updates the cell size for the current scale,
// Pane calls it to size the terminal right before resizing the screen.
func (s *Screen) CursorLocationForPosition(pos fyne.Position) (int, int) {
	s.cellSize = measureCell(s.scale())

	row := int(pos.Y / s.cellSize.Height)
	col := int(pos.X / s.cellSize.Width)
	return max(row, 0), max(col, 0)
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
		return
	}

	s.cursor.Move(s.PositionForCursorLocation(row, s.curCol))
	s.cursor.Show()
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
	if len(blank) == s.col {
		clear(blank)
	} else {
		blank = make([]cell, s.col)
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

// viewLine returns the line shown on row i of the view. Called with mu held.
func (s *Screen) viewLine(i int) []cell {
	n := s.hist.len()
	idx := n - s.offset + i
	if idx < n {
		return s.hist.at(idx)
	}
	return s.lines[idx-n]
}

// render rebuilds the objects of the rows changed since the last render.
// Called on the UI thread with mu held.
func (s *Screen) render() {
	if s.offset > 0 || s.viewDirty {
		// The view is scrolled back or just moved, redraw every row. Rows that
		// look the same are cheap, their objects are not refreshed.
		s.scrolled = 0
		s.viewDirty = false
		for i := range s.rows {
			s.renderLine(i, s.viewLine(i))
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
			s.renderLine(i, s.lines[i])
			s.dirty[i] = false
		}
	}
}

func (s *Screen) renderLine(i int, cells []cell) {
	l := s.rows[i]
	// History lines keep the width they had, cut them to the screen
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
			// Move and Resize repaint only when the value changes
			rect.Move(fyne.NewPos(float32(start)*w, 0))
			rect.Resize(fyne.NewSize(float32(end-start)*w, h))
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
	s.cellSize = measureCell(s.scale())
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
}

// blend mixes a and b half and half, used to draw faint text
func blend(a, b color.Color) color.Color {
	ar, ag, ab, aa := a.RGBA()
	br, bg, bb, ba := b.RGBA()

	return color.RGBA64{
		R: uint16((ar + br) / 2),
		G: uint16((ag + bg) / 2),
		B: uint16((ab + bb) / 2),
		A: uint16((aa + ba) / 2),
	}
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
	lines [][]cell
	head  int // index of the oldest line
	count int
}

func newHistory(max int) *history {
	return &history{lines: make([][]cell, max)}
}

// push adds a line as the newest. When full the oldest line is dropped and
// returned so its memory can be reused, otherwise it returns nil.
func (h *history) push(line []cell) []cell {
	max := len(h.lines)
	if max == 0 {
		return line
	}

	if h.count < max {
		h.lines[(h.head+h.count)%max] = line
		h.count++
		return nil
	}

	old := h.lines[h.head]
	h.lines[h.head] = line
	h.head = (h.head + 1) % max
	return old
}

// at returns the i-th line, 0 is the oldest
func (h *history) at(i int) []cell {
	return h.lines[(h.head+i)%len(h.lines)]
}

func (h *history) len() int {
	return h.count
}
