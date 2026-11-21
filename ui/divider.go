package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// splitDefaultThickness is the thickness of a divider, between the panes
var splitDefaultThickness = float32(5)

var (
	_ fyne.CanvasObject  = (*divider)(nil)
	_ fyne.Draggable     = (*divider)(nil)
	_ desktop.Cursorable = (*divider)(nil)
	_ desktop.Hoverable  = (*divider)(nil)
)

// dividerObject is the split pane a divider sizes: its direction, and the
// ratio of its first pane, set while dragging (changing) and at the end
type dividerObject interface {
	fyne.CanvasObject

	SplitDirection() int
	SetRatio(float32, bool)
}

// divider is the bar between the two panes of a split, dragged to resize them
type divider struct {
	widget.BaseWidget

	obj            dividerObject
	hovered        bool
	dragging       bool
	startDragOff   *fyne.Position
	currentDragPos fyne.Position
	ratio          float32
}

// newDivider makes the divider of the split pane obj
func newDivider(obj dividerObject) *divider {
	d := &divider{
		obj: obj,
	}

	d.ExtendBaseWidget(d)
	return d
}

// CreateRenderer is a private method to Fyne which links this widget to its
// renderer
func (d *divider) CreateRenderer() fyne.WidgetRenderer {
	d.ExtendBaseWidget(d)
	th := d.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()

	background := canvas.NewRectangle(th.Color(theme.ColorNameMenuBackground, v))
	return &dividerRenderer{
		divider:    d,
		background: background,
	}
}

// Cursor is the resize cursor of the split's direction, see
// desktop.Cursorable
func (d *divider) Cursor() desktop.Cursor {
	if d.obj.SplitDirection() == splitRight {
		return desktop.HResizeCursor
	}
	return desktop.VResizeCursor
}

// DragEnd sets the ratio the drag ended at, see fyne.Draggable
func (d *divider) DragEnd() {
	d.startDragOff = nil
	d.dragging = false
	d.Refresh()
	d.obj.SetRatio(d.ratio, false)
}

// Dragged moves the divider with the mouse, the panes follow, see
// fyne.Draggable
func (d *divider) Dragged(e *fyne.DragEvent) {
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
		ratio = x / (d.obj.Size().Width - dividerThickness(d))
	} else {
		ratio = y / (d.obj.Size().Height - dividerThickness(d))
	}

	d.ratio = ratio
	d.obj.SetRatio(ratio, true)
}

// MouseIn highlights the divider, see desktop.Hoverable
func (d *divider) MouseIn(event *desktop.MouseEvent) {
	d.hovered = true
	d.Refresh()
}

// MouseMoved does nothing, see desktop.Hoverable
func (d *divider) MouseMoved(event *desktop.MouseEvent) {
	// Nothing to do
}

// MouseOut ends the highlight, see desktop.Hoverable
func (d *divider) MouseOut() {
	d.hovered = false
	d.Refresh()
}

var _ fyne.WidgetRenderer = (*dividerRenderer)(nil)

// dividerRenderer draws a divider: a bar in the color of the tab bar, the
// accent color while hovered or dragged
type dividerRenderer struct {
	divider    *divider
	background *canvas.Rectangle
}

// Destroy does nothing, the renderer has nothing to free
func (r *dividerRenderer) Destroy() {
}

// Layout sizes the bar to the divider
func (r *dividerRenderer) Layout(size fyne.Size) {
	r.background.Resize(size)
}

// MinSize is the thickness across the split and a short length along it
func (r *dividerRenderer) MinSize() fyne.Size {
	if r.divider.obj.SplitDirection() == splitRight {
		return fyne.NewSize(dividerThickness(r.divider), dividerLength(r.divider))
	}
	return fyne.NewSize(dividerLength(r.divider), dividerThickness(r.divider))
}

// Objects is the bar
func (r *dividerRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.background}
}

// Refresh colors the bar for the theme and the hover or drag
func (r *dividerRenderer) Refresh() {
	th := r.divider.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()

	// The color of the tab bar, so it stands out from the terminal's
	// background in dark and light schemes alike, and the accent color while
	// the mouse is on it or drags it
	if r.divider.hovered || r.divider.dragging {
		r.background.FillColor = th.Color(theme.ColorNamePrimary, v)
	} else {
		r.background.FillColor = th.Color(theme.ColorNameMenuBackground, v)
	}
	r.background.Refresh()
	r.Layout(r.divider.Size())
}

// dividerTheme is the theme of d, the current one when d is nil
func dividerTheme(d *divider) fyne.Theme {
	if d == nil {
		return theme.Current()
	}

	return d.Theme()
}

// dividerThickness is the thickness of d across the split
func dividerThickness(d *divider) float32 {
	pad := splitDefaultThickness
	return pad
}

// dividerLength is the minimum length of d along the split
func dividerLength(d *divider) float32 {
	th := dividerTheme(d)
	return th.Size(theme.SizeNamePadding) * 6
}
