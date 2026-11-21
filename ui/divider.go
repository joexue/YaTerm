package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// SplitDefaultThickness is the thickness of a divider, between the panes
var SplitDefaultThickness = float32(5)

var (
	_ fyne.CanvasObject  = (*Divider)(nil)
	_ fyne.Draggable     = (*Divider)(nil)
	_ desktop.Cursorable = (*Divider)(nil)
	_ desktop.Hoverable  = (*Divider)(nil)
)

// DividerObject is the split pane a divider sizes: its direction, and the
// ratio of its first pane, set while dragging (changing) and at the end
type DividerObject interface {
	fyne.CanvasObject

	SplitDirection() int
	SetRatio(float32, bool)
}

// Divider is the bar between the two panes of a split, dragged to resize them
type Divider struct {
	widget.BaseWidget

	obj            DividerObject
	hovered        bool
	dragging       bool
	startDragOff   *fyne.Position
	currentDragPos fyne.Position
	ratio          float32
}

// NewDivider makes the divider of the split pane obj
func NewDivider(obj DividerObject) *Divider {
	d := &Divider{
		obj: obj,
	}

	d.ExtendBaseWidget(d)
	return d
}

// CreateRenderer is a private method to Fyne which links this widget to its
// renderer
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

// Cursor is the resize cursor of the split's direction, see
// desktop.Cursorable
func (d *Divider) Cursor() desktop.Cursor {
	if d.obj.SplitDirection() == splitRight {
		return desktop.HResizeCursor
	}
	return desktop.VResizeCursor
}

// DragEnd sets the ratio the drag ended at, see fyne.Draggable
func (d *Divider) DragEnd() {
	d.startDragOff = nil
	d.dragging = false
	d.Refresh()
	d.obj.SetRatio(d.ratio, false)
}

// Dragged moves the divider with the mouse, the panes follow, see
// fyne.Draggable
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

	d.ratio = ratio
	d.obj.SetRatio(ratio, true)
}

// MouseIn highlights the divider, see desktop.Hoverable
func (d *Divider) MouseIn(event *desktop.MouseEvent) {
	d.hovered = true
	d.Refresh()
}

// MouseMoved does nothing, see desktop.Hoverable
func (d *Divider) MouseMoved(event *desktop.MouseEvent) {
	// Nothing to do
}

// MouseOut ends the highlight, see desktop.Hoverable
func (d *Divider) MouseOut() {
	d.hovered = false
	d.Refresh()
}

var _ fyne.WidgetRenderer = (*DividerRenderer)(nil)

// DividerRenderer draws a divider: a bar in the color of the tab bar, the
// accent color while hovered or dragged
type DividerRenderer struct {
	Divider    *Divider
	background *canvas.Rectangle
}

// Destroy does nothing, the renderer has nothing to free
func (r *DividerRenderer) Destroy() {
}

// Layout sizes the bar to the divider
func (r *DividerRenderer) Layout(size fyne.Size) {
	r.background.Resize(size)
}

// MinSize is the thickness across the split and a short length along it
func (r *DividerRenderer) MinSize() fyne.Size {
	if r.Divider.obj.SplitDirection() == splitRight {
		return fyne.NewSize(DividerThickness(r.Divider), DividerLength(r.Divider))
	}
	return fyne.NewSize(DividerLength(r.Divider), DividerThickness(r.Divider))
}

// Objects is the bar
func (r *DividerRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.background}
}

// Refresh colors the bar for the theme and the hover or drag
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

// DividerTheme is the theme of d, the current one when d is nil
func DividerTheme(d *Divider) fyne.Theme {
	if d == nil {
		return theme.Current()
	}

	return d.Theme()
}

// DividerThickness is the thickness of d across the split
func DividerThickness(d *Divider) float32 {
	pad := SplitDefaultThickness
	return pad
}

// DividerLength is the minimum length of d along the split
func DividerLength(d *Divider) float32 {
	th := DividerTheme(d)
	return th.Size(theme.SizeNamePadding) * 6
}
