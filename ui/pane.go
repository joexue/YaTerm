package ui

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"image/color"
	"yaterm/assert"
	"yaterm/terminal"
)

const (
	SplitNone = iota
	SplitHorizontal
	SplitVertical
)

var _ EventReceiver = (*Pane)(nil)
var id int = 0

type Pane struct {
	widget.BaseWidget

	id   int
	term *terminal.Terminal

	split   int
	ratio   float32
	first   *Pane
	second  *Pane
	divider *Divider

	root      *Pane
	parent    *Pane
	lastFocus *Pane

	border *canvas.Rectangle
	event  *EventGlass

	// Callback from creater, called when the whole pane tree is finished
	OnTearDown func()
}

func NewPane(parent *Pane, term *terminal.Terminal, run bool) *Pane {
	if term == nil {
		term = terminal.New()
	}

	b := canvas.NewRectangle(color.Transparent)
	b.StrokeWidth = 1
	b.StrokeColor = color.Transparent

	p := &Pane{
		id:     id,
		term:   term,
		split:  SplitNone,
		first:  nil,
		second: nil,
		parent: parent,
		border: b,
	}

	ev := NewEventGlass(p)
	p.event = ev

	if p.parent == nil {
		p.lastFocus = p
		p.root = p
	} else {
		p.root = p.parent.root
	}

	p.ExtendBaseWidget(p)

	term.OnExit = p.TryClose
	if run {
		go func() {
			_ = term.RunLocalShell()
			// here, the p may change, so we cannot call p.TryClose
			if f := term.OnExit; f != nil {
				f()
			}
		}()
	}

	id++
	return p
}

// Called by TabControl
func (p *Pane) TearDown() {
	if p.split == SplitNone {
		t := p.term
		t.OnExit = nil
		t.Exit()
	} else {
		p.first.TearDown()
		p.second.TearDown()
	}
}

// Called by TabControl
func (p *Pane) TryFocus() {
	fyne.Do(func() {
		focusPane := p.root.lastFocus

		if c := fyne.CurrentApp().Driver().CanvasForObject(focusPane); c != nil {
			c.Focus(focusPane)
		}
	})
}

func (p *Pane) TryClose() {
	fyne.Do(func() {
		p.Close()
	})
}

func (p *Pane) Split(direction int, ratio float32) {
	p.split = direction
	p.first = NewPane(p, p.term, false)
	p.second = NewPane(p, nil, true)
	p.term = nil

	p.divider = NewDivider(p)
	p.ratio = ratio
	p.Refresh()

	p.root.lastFocus = p.second

	fyne.Do(func() {
		p.second.TryFocus()
	})
}

func (p *Pane) Close() {
	if p == p.root {
		if f := p.OnTearDown; f != nil {
			f()
		}
		return
	}

	pa := p.parent
	var take *Pane
	if pa.first == p {
		take = pa.second
	} else {
		take = pa.first
	}

	pa.split = take.split
	pa.first = take.first
	pa.second = take.second
	pa.term = take.term
	if pa.split == SplitNone {
		pa.term = take.term
		pa.term.OnExit = pa.TryClose
	} else {
		pa.first.parent = pa
		pa.second.parent = pa
	}

	pa.root.Refresh()
}

func (p *Pane) SetRatio(ratio float32) {
	p.ratio = ratio
	p.Refresh()
}

func (p *Pane) CreateRenderer() fyne.WidgetRenderer {
	r := &paneRenderer{
		pane: p,
	}

	return r
}

func (p *Pane) Tapped(pe *fyne.PointEvent) {
	// Do nothing for now
}

func (p *Pane) TappedSecondary(pe *fyne.PointEvent) {
	if c := fyne.CurrentApp().Driver().CanvasForObject(p); c != nil {
		hsplitItem := fyne.NewMenuItemWithIcon("Horizontal Split", theme.NewThemedResource(assert.HSplitIconRes), func() {
			p.Split(SplitHorizontal, 0.5)
		})
		vsplitItem := fyne.NewMenuItemWithIcon("Vertical Split", theme.NewThemedResource(assert.VSplitIconRes), func() {
			p.Split(SplitVertical, 0.5)
		})

		separatorItem := fyne.NewMenuItemSeparator()

		closePaneIterm := fyne.NewMenuItemWithIcon("Close Pane", theme.Icon(theme.IconNameWindowClose), func() {
			p.term.Exit()
		})
		closeTabIterm := fyne.NewMenuItemWithIcon("Close Tab", theme.Icon(theme.IconNameWindowClose), func() {
			p.root.TearDown()
			if f := p.root.OnTearDown; f != nil {
				f()
			}
		})

		menuItems := []*fyne.MenuItem{hsplitItem, vsplitItem, separatorItem, closePaneIterm, closeTabIterm}
		popUpMenu := widget.NewPopUpMenu(fyne.NewMenu("", menuItems...), c)
		popUpMenu.ShowAtRelativePosition(pe.Position, p)
	}
}

func (p *Pane) FocusGained() {
	p.root.lastFocus = p
	fmt.Println("Focus: ", p.id, p.Size())
	if p.parent != nil {
		p.border.StrokeColor = theme.Color(theme.ColorNamePrimary)
		p.border.Refresh()
	}

	p.term.FocusGained()
}

func (p *Pane) FocusLost() {
	p.border.StrokeColor = color.Transparent
	p.border.Refresh()
	if p.term != nil {
		p.term.FocusLost()
	}
}

func (p *Pane) TypedRune(r rune) {
	p.term.TypedRune(r)
}

func (p *Pane) TypedKey(ke *fyne.KeyEvent) {
	p.term.TypedKey(ke)
}

var _ fyne.WidgetRenderer = (*paneRenderer)(nil)

type paneRenderer struct {
	pane *Pane
}

func (r *paneRenderer) Destroy() {
}

func (r *paneRenderer) Layout(size fyne.Size) {
	w := size.Width
	h := size.Height
	switch r.pane.split {
	case SplitNone:
		newSize := size.SubtractWidthHeight(r.pane.border.StrokeWidth, r.pane.border.StrokeWidth)
		newPos := fyne.NewPos(r.pane.border.StrokeWidth, r.pane.border.StrokeWidth)

		pad := fyne.NewSize(2, 2)
		offset := fyne.NewPos(2, 0)

		r.pane.border.Resize(size)
		r.pane.border.Move(fyne.NewPos(0, 0))

		r.pane.term.Resize(newSize.Subtract(pad))
		r.pane.term.Move(newPos.Add(offset))

		r.pane.event.Resize(newSize.Subtract(pad))
		r.pane.event.Move(newPos.Add(offset))

	case SplitHorizontal:
		w1 := (w - SplitDefaultThickness) * r.pane.ratio
		w2 := w - SplitDefaultThickness - w1

		r.pane.first.Resize(fyne.NewSize(w1, h))
		r.pane.first.Move(fyne.NewPos(0, 0))

		r.pane.divider.Resize(fyne.NewSize(SplitDefaultThickness, h))
		r.pane.divider.Move(fyne.NewPos(w1, 0))

		r.pane.second.Resize(fyne.NewSize(w2, h))
		r.pane.second.Move(fyne.NewPos(w1+SplitDefaultThickness, 0))

	case SplitVertical:
		h1 := (h - SplitDefaultThickness) * r.pane.ratio
		h2 := h - SplitDefaultThickness - h1

		r.pane.first.Resize(fyne.NewSize(w, h1))
		r.pane.first.Move(fyne.NewPos(0, 0))

		r.pane.divider.Resize(fyne.NewSize(w, SplitDefaultThickness))
		r.pane.divider.Move(fyne.NewPos(0, h1))

		r.pane.second.Resize(fyne.NewSize(w, h2))
		r.pane.second.Move(fyne.NewPos(0, h1+SplitDefaultThickness))
	}
}

func (r *paneRenderer) MinSize() fyne.Size {
	w := r.pane.border.StrokeWidth * 2
	if r.pane.split == SplitNone {
		return fyne.NewSize(w, w)
	} else {
		s1 := r.pane.first.MinSize()
		s2 := r.pane.second.MinSize()
		return s1.Add(s2)
	}
}

func (r *paneRenderer) Objects() []fyne.CanvasObject {
	if r.pane.split == SplitNone {
		return []fyne.CanvasObject{r.pane.border, r.pane.term, r.pane.event}
	} else {
		return []fyne.CanvasObject{r.pane.first, r.pane.divider, r.pane.second}
	}
}

func (r *paneRenderer) Refresh() {
	switch r.pane.split {
	case SplitNone:
		r.pane.border.Refresh()
		r.pane.term.Refresh()
		r.pane.event.Refresh()
	case SplitHorizontal, SplitVertical:
		r.pane.divider.Refresh()
		r.pane.first.Refresh()
		r.pane.second.Refresh()
	}
	r.Layout(r.pane.Size())
}
