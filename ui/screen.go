package ui

import (
	"image/color"
	"math"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"yaterm/terminal"
)

var (
	_ fyne.Widget     = (*Screen)(nil)
	_ terminal.Screen = (*Screen)(nil)
)

// cell is one character on the screen, colors are already resolved for
// inverse/faint/hidden, a nil fg or bg means the theme default.
type cell struct {
	r      rune
	fg, bg color.Color
	style  fyne.TextStyle
}

func (c cell) sameAttr(o cell) bool {
	return c.fg == o.fg && c.bg == o.bg && c.style == o.style
}

// screenLine draws one row. Cells with the same attributes are merged into
// one run, drawn by a single background rectangle and a single text object.
// The objects are pooled and reused between redraws.
type screenLine struct {
	box   *fyne.Container
	rects []*canvas.Rectangle
	texts []*canvas.Text
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

	// Only touched on the UI thread, the terminal goroutine goes through fyne.Do
	lines [][]cell
	rows  []*screenLine
	dirty []bool

	objects []fyne.CanvasObject

	row, col int
	cellSize fyne.Size

	size fyne.Size
}

func NewScreen() *Screen {
	s := &Screen{
		cursor:   canvas.NewRectangle(theme.Color(theme.ColorNamePrimary)),
		cellSize: measureCell(),
	}

	s.objects = []fyne.CanvasObject{s.cursor}
	s.ExtendBaseWidget(s)

	return s
}

// measureCell keeps the font's real advance width so a merged run of text
// lines up with the cell grid, the height is rounded so rows stack without seams.
func measureCell() fyne.Size {
	size := fyne.MeasureText("M", theme.TextSize(), fyne.TextStyle{Monospace: true})
	size.Height = float32(math.Round(float64(size.Height)))
	return size
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

	fyne.Do(func() {
		// The terminal may still use the old size right after a resize
		if row < 0 || row >= s.row || col < 0 || col >= s.col {
			return
		}

		s.lines[row][col] = c
		s.dirty[row] = true
	})
}

func (s *Screen) Resize(size fyne.Size) {
	if size.Width <= 0 || size.Height <= 0 || size == s.size {
		return
	}

	s.size = size
	// Keep row/col consistent with what Pane hands to the terminal.
	s.row, s.col = s.CursorLocationForPosition(fyne.NewPos(size.Width, size.Height))
	s.resizeBuffer()

	s.cursor.Resize(s.cellSize)
	s.BaseWidget.Resize(size)
	s.render()
}

// resizeBuffer fits the cell buffer and the row objects to s.row x s.col,
// keeping the content that is still on screen.
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

	s.dirty = make([]bool, s.row)
	for i := range s.dirty {
		s.dirty[i] = true
	}

	// Rows first, the cursor is drawn on top
	s.objects = make([]fyne.CanvasObject, 0, s.row+1)
	for _, l := range s.rows {
		s.objects = append(s.objects, l.box)
	}
	s.objects = append(s.objects, s.cursor)
}

func (s *Screen) CursorLocationForPosition(pos fyne.Position) (int, int) {
	row := int(pos.Y / s.cellSize.Height)
	col := int(pos.X / s.cellSize.Width)
	return max(row, 0), max(col, 0)
}

func (s *Screen) PositionForCursorLocation(row, col int) fyne.Position {
	return fyne.NewPos(float32(col)*s.cellSize.Width, float32(row)*s.cellSize.Height)
}

func (s *Screen) Flush(row, col int) {
	fyne.Do(func() {
		s.render()
		s.cursor.Move(s.PositionForCursorLocation(row, col))
	})
}

func (s *Screen) MoveCursor(row, col int) {
}

// ScrollUp is called from the terminal goroutine, so it goes through fyne.Do
// to stay ordered with the cell updates queued by Project.
func (s *Screen) ScrollUp() {
	fyne.Do(func() {
		if s.row <= 0 {
			return
		}

		last := s.row - 1

		// Rotate the top line to the bottom and blank it, the other rows keep
		// their drawn objects and are just moved up.
		top, topRow := s.lines[0], s.rows[0]
		copy(s.lines, s.lines[1:])
		copy(s.rows, s.rows[1:])
		copy(s.dirty, s.dirty[1:])
		clear(top)
		s.lines[last], s.rows[last] = top, topRow
		s.dirty[last] = true

		s.layoutRows()
	})
}

// render rebuilds the objects of the rows changed since the last render
func (s *Screen) render() {
	for i, d := range s.dirty {
		if d {
			s.renderLine(i)
			s.dirty[i] = false
		}
	}
}

func (s *Screen) renderLine(i int) {
	l, cells := s.rows[i], s.lines[i]
	w, h := s.cellSize.Width, s.cellSize.Height
	fgDefault := theme.Color(theme.ColorNameForeground)
	textSize := theme.TextSize()

	var (
		rects []fyne.CanvasObject
		texts []fyne.CanvasObject
		sb    strings.Builder
	)

	for start := 0; start < len(cells); {
		c := cells[start]

		// Only ASCII is merged, other runes may be drawn by a fallback font
		// with a different width, so they get their own text to stay on the grid.
		end := start + 1
		if c.r < 0x80 {
			for end < len(cells) && cells[end].r < 0x80 && cells[end].sameAttr(c) {
				end++
			}
		}

		pos := fyne.NewPos(float32(start)*w, 0)
		size := fyne.NewSize(float32(end-start)*w, h)

		if c.bg != nil {
			rect := l.rect(len(rects))
			rect.FillColor = c.bg
			rect.Move(pos)
			rect.Resize(size)
			rects = append(rects, rect)
		}

		sb.Reset()
		blank := true
		for _, cc := range cells[start:end] {
			r := cc.r
			if r == 0 {
				r = ' '
			}
			if r != ' ' {
				blank = false
			}
			sb.WriteRune(r)
		}

		// Spaces only need drawing when a line goes through them
		if !blank || c.style.Underline || c.style.Strikethrough {
			fg := c.fg
			if fg == nil {
				fg = fgDefault
			}

			text := l.text(len(texts))
			text.Text = sb.String()
			text.Color = fg
			text.TextStyle = c.style
			text.TextSize = textSize
			text.Move(pos)
			text.Resize(size)
			texts = append(texts, text)
		}

		start = end
	}

	// Backgrounds first so no text is covered by a later run's background
	l.box.Objects = append(rects, texts...)
	l.box.Refresh()
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
