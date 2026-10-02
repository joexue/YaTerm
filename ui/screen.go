package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"image/color"
)

var _ fyne.Widget = (*Screen)(nil)

type Screen struct {
	widget.TextGrid
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
