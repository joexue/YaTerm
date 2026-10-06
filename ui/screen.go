package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"image/color"
	"yaterm/terminal"
)

var (
	_ fyne.Widget     = (*Screen)(nil)
	_ terminal.Screen = (*Screen)(nil)
)

type Screen struct {
	widget.BaseWidget

	cursor   *canvas.Rectangle
	textGrid *widget.TextGrid

	row, col int

	size fyne.Size
}

func NewScreen() *Screen {
	s := &Screen{
		textGrid: widget.NewTextGrid(),
		cursor:   canvas.NewRectangle(theme.Color(theme.ColorNamePrimary)),
	}

	s.textGrid.Scroll = fyne.ScrollNone
	s.textGrid.SetText("")

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

	fyne.Do(func() {
		// Inverse, faint and hidden need real colors, so fill in the theme
		// defaults here on the UI thread where the theme is safe to read.
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

		cell := widget.TextGridCell{
			Rune: r,
			Style: &widget.CustomTextGridStyle{
				FGColor:   fg,
				BGColor:   bg,
				TextStyle: textStyle,
			},
		}

		s.textGrid.SetCell(row, col, cell)
	})
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

func (s *Screen) Resize(size fyne.Size) {
	if size.Width <= 0 || size.Height <= 0 || size == s.size {
		return
	}

	s.size = size
	// Keep row/col consistent with what Pane hands to the terminal.
	maxPos := fyne.NewPos(size.Width, size.Height)
	s.row, s.col = s.textGrid.CursorLocationForPosition(maxPos)

	s.cursor.Resize(fyne.NewSize(maxPos.X/float32(s.col), maxPos.Y/float32(s.row)))
	s.BaseWidget.Resize(size)
	//s.textGrid.Resize(size)
}

func (s *Screen) CursorLocationForPosition(pos fyne.Position) (int, int) {
	return s.textGrid.CursorLocationForPosition(pos)
}

func (s *Screen) Flush() {
	fyne.Do(func() {
		s.textGrid.Refresh()
		s.cursor.Refresh()
	})
}

// MoveCursor and ScrollUp are called from the terminal goroutine, so they go
// through fyne.Do to stay ordered with the cell updates queued by Project.
func (s *Screen) MoveCursor(row, col int) {
	fyne.Do(func() {
		pos := s.textGrid.PositionForCursorLocation(row, col)
		s.cursor.Move(pos)
	})
}

func (s *Screen) ScrollUp() {
	fyne.Do(func() {
		rows := s.row
		if rows <= 0 {
			rows = len(s.textGrid.Rows)
		}
		if rows <= 0 {
			return
		}

		// Pad to the full screen height so the bottom line is always blank
		// after the scroll, even if the lower rows were never written.
		for len(s.textGrid.Rows) < rows {
			s.textGrid.Rows = append(s.textGrid.Rows, widget.TextGridRow{})
		}

		s.textGrid.Rows = append(s.textGrid.Rows[1:rows], widget.TextGridRow{})
	})
}

func (s *Screen) Rows() int {
	return len(s.textGrid.Rows)
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
	r.screen.textGrid.Resize(size)
	r.screen.textGrid.Move(fyne.NewPos(0, 0))
}

func (r *screenRenderer) MinSize() fyne.Size {
	return r.screen.textGrid.MinSize()
}

func (r *screenRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.screen.textGrid, r.screen.cursor}
}

func (r *screenRenderer) Refresh() {
	r.screen.textGrid.Refresh()
	r.screen.cursor.Refresh()
}
