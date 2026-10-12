package ui

import (
	"image/color"
	"math"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

/*
 * ================================
 *  Scroll bar of the history. It lies over the right edge of the screen and
 *  takes no columns, so it is hidden most of the time: it shows when the
 *  wheel scrolls or the mouse comes to the right edge, and hides again a
 *  moment after, unless the mouse is on it or drags it.
 * ================================
 */

const (
	// scrollBarWidth is the width that takes the mouse, the thumb is drawn
	// thinner until the mouse is on it
	scrollBarWidth     = float32(12)
	scrollThumbWidth   = float32(6)
	scrollThumbHovered = float32(10)
	scrollThumbMin     = float32(20)
	scrollThumbMargin  = float32(1)
	// scrollEdge is how near the right edge the mouse shows the bar
	scrollEdge = float32(16)
	// scrollHideDelay is how long the bar stays after the last activity
	scrollHideDelay = 1200 * time.Millisecond
)

var (
	_ fyne.Draggable     = (*scrollBar)(nil)
	_ fyne.Tappable      = (*scrollBar)(nil)
	_ desktop.Hoverable  = (*scrollBar)(nil)
	_ desktop.Cursorable = (*scrollBar)(nil)
)

type scrollBar struct {
	widget.BaseWidget

	screen *Screen
	track  *canvas.Rectangle
	thumb  *canvas.Rectangle

	// The view, set by the screen: lines back into the history, the lines
	// in the history and the screen rows
	offset, hist, rows int

	hovered  bool
	dragging bool
	grab     float32 // where in the thumb the drag holds it
	hideGen  int     // a newer reveal makes older hide timers do nothing
}

func newScrollBar(s *Screen) *scrollBar {
	b := &scrollBar{
		screen: s,
		track:  canvas.NewRectangle(color.Transparent),
		thumb:  canvas.NewRectangle(color.Transparent),
	}
	b.thumb.CornerRadius = scrollThumbHovered / 2
	b.ExtendBaseWidget(b)
	b.Hide()
	return b
}

// update takes the view from the screen. Called on the UI thread with the
// screen's lock held, so it must not call back into the screen.
func (b *scrollBar) update(offset, hist, rows int) {
	if b.offset == offset && b.hist == hist && b.rows == rows {
		return
	}
	b.offset, b.hist, b.rows = offset, hist, rows
	if hist == 0 {
		b.Hide()
		return
	}
	if b.Visible() {
		b.layoutThumb()
	}
}

// reveal shows the bar when there is history to scroll, and hides it after
// a while without activity
func (b *scrollBar) reveal() {
	if b.hist == 0 {
		return
	}
	if !b.Visible() {
		b.layoutThumb()
		b.Show()
	}

	b.hideGen++
	gen := b.hideGen
	time.AfterFunc(scrollHideDelay, func() {
		fyne.Do(func() {
			if gen != b.hideGen {
				return
			}
			if b.hovered || b.dragging {
				b.reveal()
				return
			}
			b.Hide()
		})
	})
}

// thumbGeometry is the thumb's top and height, and how far it can move
func (b *scrollBar) thumbGeometry() (top, height, travel float32) {
	h := b.Size().Height
	total := b.hist + b.rows
	if b.hist == 0 || total == 0 || h <= 0 {
		return 0, h, 0
	}
	height = min(h, max(scrollThumbMin, h*float32(b.rows)/float32(total)))
	travel = h - height
	top = travel * float32(b.hist-b.offset) / float32(b.hist)
	return top, height, travel
}

func (b *scrollBar) layoutThumb() {
	fg := theme.Color(theme.ColorNameForeground)
	r, g, bl, _ := fg.RGBA()
	shade := func(a uint8) color.Color {
		return color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(bl >> 8), A: a}
	}

	width := scrollThumbWidth
	thumbAlpha, trackAlpha := uint8(0x60), uint8(0)
	if b.hovered || b.dragging {
		width = scrollThumbHovered
		thumbAlpha, trackAlpha = 0xa0, 0x18
	}

	size := b.Size()
	b.track.FillColor = shade(trackAlpha)
	b.track.Resize(size)
	b.track.Refresh()

	top, height, _ := b.thumbGeometry()
	b.thumb.FillColor = shade(thumbAlpha)
	b.thumb.CornerRadius = width / 2
	b.thumb.Move(fyne.NewPos(size.Width-width-scrollThumbMargin, top))
	b.thumb.Resize(fyne.NewSize(width, height))
	b.thumb.Refresh()
}

// scrollToTop moves the view so the thumb's top is at y
func (b *scrollBar) scrollToTop(y float32) {
	_, _, travel := b.thumbGeometry()
	if travel <= 0 {
		return
	}
	y = max(0, min(y, travel))
	b.screen.ScrollTo(b.hist - int(math.Round(float64(y/travel*float32(b.hist)))))
}

func (b *scrollBar) Dragged(ev *fyne.DragEvent) {
	if !b.dragging {
		b.dragging = true
		// Held where it was taken, or by its middle when the drag started on
		// the track
		top, height, _ := b.thumbGeometry()
		if y := ev.Position.Y - ev.Dragged.DY; y >= top && y < top+height {
			b.grab = y - top
		} else {
			b.grab = height / 2
		}
		b.layoutThumb()
	}
	b.scrollToTop(ev.Position.Y - b.grab)
}

func (b *scrollBar) DragEnd() {
	b.dragging = false
	b.layoutThumb()
	b.reveal()
}

// Tapped on the track pages up or down, like other scroll bars
func (b *scrollBar) Tapped(ev *fyne.PointEvent) {
	top, height, _ := b.thumbGeometry()
	switch {
	case ev.Position.Y < top:
		b.screen.ScrollLines(b.rows)
	case ev.Position.Y >= top+height:
		b.screen.ScrollLines(-b.rows)
	}
	b.reveal()
}

func (b *scrollBar) MouseIn(*desktop.MouseEvent) {
	b.hovered = true
	b.layoutThumb()
	b.reveal()
}

func (b *scrollBar) MouseMoved(*desktop.MouseEvent) {}

func (b *scrollBar) MouseOut() {
	b.hovered = false
	b.layoutThumb()
	b.reveal()
}

func (b *scrollBar) Cursor() desktop.Cursor {
	return desktop.DefaultCursor
}

func (b *scrollBar) CreateRenderer() fyne.WidgetRenderer {
	return &scrollBarRenderer{bar: b, objects: []fyne.CanvasObject{b.track, b.thumb}}
}

type scrollBarRenderer struct {
	bar     *scrollBar
	objects []fyne.CanvasObject
}

func (r *scrollBarRenderer) Layout(fyne.Size)             { r.bar.layoutThumb() }
func (r *scrollBarRenderer) MinSize() fyne.Size           { return fyne.NewSize(scrollBarWidth, scrollThumbMin) }
func (r *scrollBarRenderer) Objects() []fyne.CanvasObject { return r.objects }
func (r *scrollBarRenderer) Refresh()                     { r.bar.layoutThumb() }
func (r *scrollBarRenderer) Destroy()                     {}
