package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"image/color"
)

var _ fyne.Widget = (*Screen)(nil)

type Screen struct {
	widget.TextGrid

	size fyne.Size

	row, col int
}

func NewScreen() *Screen {
	s := &Screen{}

	s.Scroll = fyne.ScrollNone

	s.ExtendBaseWidget(s)

	s.SetText("")

	return s
}

func (s *Screen) Project(r rune, row, col int, fg, bg color.Color) {
	textStyle := fyne.TextStyle{
		Monospace: true,

		Bold:          false,
		Italic:        false,
		Underline:     false,
		Strikethrough: false,
	}

	cellStyle := &widget.CustomTextGridStyle{
		FGColor:   fg,
		BGColor:   bg,
		TextStyle: textStyle,
	}

	cell := widget.TextGridCell{
		Rune:  r,
		Style: cellStyle,
	}

	fyne.Do(func() {
		s.SetCell(row, col, cell)
		s.Refresh()
	})
}

func (s *Screen) Resize(size fyne.Size) {
	if size.Width <= 0 || size.Height <= 0 || size == s.size {
		return
	}

	s.size = size
	maxPos := fyne.NewPos(size.Width, size.Height)
	r, c := s.CursorLocationForPosition(maxPos)
	s.row, s.col = r, c

	s.BaseWidget.Resize(size)
}
