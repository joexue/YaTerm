package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

var SplitDefaultThickness = float32(4)

var (
	_ fyne.CanvasObject  = (*Divider)(nil)
	_ fyne.Draggable     = (*Divider)(nil)
	_ desktop.Cursorable = (*Divider)(nil)
	_ desktop.Hoverable  = (*Divider)(nil)
)

type Divider struct {
	widget.BaseWidget

	pane           *Pane
	hovered        bool
	startDragOff   *fyne.Position
	currentDragPos fyne.Position
}

func NewDivider(pane *Pane) *Divider {
	d := &Divider{
		pane: pane,
	}

	d.ExtendBaseWidget(d)
	return d
}

// CreateRenderer is a private method to Fyne which links this widget to its renderer
func (d *Divider) CreateRenderer() fyne.WidgetRenderer {
	d.ExtendBaseWidget(d)
	th := d.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()

	background := canvas.NewRectangle(th.Color(theme.ColorNameShadow, v))
	return &DividerRenderer{
		Divider:    d,
		background: background,
	}
}

func (d *Divider) Cursor() desktop.Cursor {
	if d.pane.split == SplitHorizontal {
		return desktop.HResizeCursor
	}
	return desktop.VResizeCursor
}

func (d *Divider) DragEnd() {
	d.startDragOff = nil
}

func (d *Divider) Dragged(e *fyne.DragEvent) {
	if d.startDragOff == nil {
		d.currentDragPos = d.Position().Add(e.Position)
		start := e.Position.Subtract(e.Dragged)
		d.startDragOff = &start
	} else {
		d.currentDragPos = d.currentDragPos.Add(e.Dragged)
	}

	x, y := d.currentDragPos.Components()
	var ratio, firstRatio, secondRatio float32
	if d.pane.split == SplitHorizontal {
		widthFree := float32(d.pane.Size().Width - DividerThickness(d))
		firstRatio = float32(d.pane.first.MinSize().Width) / widthFree
		secondRatio = 1. - (float32(d.pane.second.MinSize().Width) / widthFree)
		ratio = float32(x-d.startDragOff.X) / widthFree
	} else {
		heightFree := float32(d.pane.Size().Height - DividerThickness(d))
		firstRatio = float32(d.pane.first.MinSize().Height) / heightFree
		secondRatio = 1. - (float32(d.pane.second.MinSize().Height) / heightFree)
		ratio = float32(y-d.startDragOff.Y) / heightFree
	}

	if ratio < firstRatio {
		ratio = firstRatio
	}

	if ratio > secondRatio {
		ratio = secondRatio
	}

	d.pane.SetRatio(ratio)
}

func (d *Divider) MouseIn(event *desktop.MouseEvent) {
	d.hovered = true
	d.Refresh()
}

func (d *Divider) MouseMoved(event *desktop.MouseEvent) {
	// Nothing to do
}

func (d *Divider) MouseOut() {
	d.hovered = false
	d.Refresh()
}

var _ fyne.WidgetRenderer = (*DividerRenderer)(nil)

type DividerRenderer struct {
	Divider    *Divider
	background *canvas.Rectangle
}

func (r *DividerRenderer) Destroy() {
}

func (r *DividerRenderer) Layout(size fyne.Size) {
	r.background.Resize(size)
}

func (r *DividerRenderer) MinSize() fyne.Size {
	if r.Divider.pane.split == SplitHorizontal {
		return fyne.NewSize(DividerThickness(r.Divider), DividerLength(r.Divider))
	}
	return fyne.NewSize(DividerLength(r.Divider), DividerThickness(r.Divider))
}

func (r *DividerRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.background}
}

func (r *DividerRenderer) Refresh() {
	th := r.Divider.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()

	if r.Divider.hovered {
		r.background.FillColor = th.Color(theme.ColorNameHover, v)
	} else {
		r.background.FillColor = th.Color(theme.ColorNameShadow, v)
	}
	r.background.Refresh()
	r.Layout(r.Divider.Size())
}

func DividerTheme(d *Divider) fyne.Theme {
	if d == nil {
		return theme.Current()
	}

	return d.Theme()
}

func DividerThickness(d *Divider) float32 {
	th := DividerTheme(d)
	pad := th.Size(theme.SizeNameSplitThickness)
	if pad == 0 {
		pad = SplitDefaultThickness
	}
	return pad
}

func DividerLength(d *Divider) float32 {
	th := DividerTheme(d)
	return th.Size(theme.SizeNamePadding) * 6
}
