package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

var SplitDefaultThickness = float32(5)

var (
	_ fyne.CanvasObject  = (*Divider)(nil)
	_ fyne.Draggable     = (*Divider)(nil)
	_ desktop.Cursorable = (*Divider)(nil)
	_ desktop.Hoverable  = (*Divider)(nil)
)

type DividerObject interface {
	fyne.CanvasObject

	SplitDirection() int
	SetRatio(float32)
}

type Divider struct {
	widget.BaseWidget

	obj            DividerObject
	hovered        bool
	dragging       bool
	startDragOff   *fyne.Position
	currentDragPos fyne.Position
}

func NewDivider(obj DividerObject) *Divider {
	d := &Divider{
		obj: obj,
	}

	d.ExtendBaseWidget(d)
	return d
}

// CreateRenderer is a private method to Fyne which links this widget to its renderer
func (d *Divider) CreateRenderer() fyne.WidgetRenderer {
	d.ExtendBaseWidget(d)
	th := d.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()

	background := canvas.NewRectangle(th.Color(theme.ColorNameMenuBackground, v))
	return &DividerRenderer{
		Divider:    d,
		background: background,
	}
}

func (d *Divider) Cursor() desktop.Cursor {
	if d.obj.SplitDirection() == splitRight {
		return desktop.HResizeCursor
	}
	return desktop.VResizeCursor
}

func (d *Divider) DragEnd() {
	d.startDragOff = nil
	d.dragging = false
	d.Refresh()
}

func (d *Divider) Dragged(e *fyne.DragEvent) {
	if !d.dragging {
		d.dragging = true
		d.Refresh()
	}

	if d.startDragOff == nil {
		d.currentDragPos = d.Position().Add(e.Position)
		start := e.Position.Subtract(e.Dragged)
		d.startDragOff = &start
	} else {
		d.currentDragPos = d.currentDragPos.Add(e.Dragged)
	}

	x, y := d.currentDragPos.Components()
	var ratio float32
	if d.obj.SplitDirection() == splitRight {
		ratio = x / (d.obj.Size().Width - DividerThickness(d))
	} else {
		ratio = y / (d.obj.Size().Height - DividerThickness(d))
	}

	d.obj.SetRatio(ratio)
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
	if r.Divider.obj.SplitDirection() == splitRight {
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

	// The color of the tab bar, so it stands out from the terminal's
	// background in dark and light schemes alike, and the accent color while
	// the mouse is on it or drags it
	if r.Divider.hovered || r.Divider.dragging {
		r.background.FillColor = th.Color(theme.ColorNamePrimary, v)
	} else {
		r.background.FillColor = th.Color(theme.ColorNameMenuBackground, v)
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
	pad := SplitDefaultThickness
	return pad
}

func DividerLength(d *Divider) float32 {
	th := DividerTheme(d)
	return th.Size(theme.SizeNamePadding) * 6
}
