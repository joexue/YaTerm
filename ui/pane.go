package ui

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"image/color"
	"unicode/utf8"
	"yaterm/assert"
	"yaterm/terminal"
)

var (
	_ fyne.CanvasObject      = (*Pane)(nil)
	_ fyne.Widget            = (*Pane)(nil)
	_ fyne.Focusable         = (*Pane)(nil)
	_ fyne.Tabbable          = (*Pane)(nil)
	_ fyne.Tappable          = (*Pane)(nil)
	_ fyne.SecondaryTappable = (*Pane)(nil)
)

const (
	SplitNone = iota
	SplitHorizontal
	SplitVertical
)

var PaneBorderWidth = float32(1)
var PanePadding = (PaneBorderWidth * 2)

var id int = 0

type Pane struct {
	widget.BaseWidget

	id   int
	term *terminal.Terminal
	size fyne.Size

	split   int
	ratio   float32
	first   *Pane
	second  *Pane
	divider *Divider

	root      *Pane
	parent    *Pane
	lastFocus *Pane

	border *canvas.Rectangle

	screen *Screen

	// Callback from creater, called when the whole pane tree is finished
	OnTearDown func()
}

func NewPane(parent *Pane, term *terminal.Terminal, screen *Screen, run bool) *Pane {
	if term == nil {
		term = terminal.New()
	}

	if screen == nil {
		screen = NewScreen()
	}

	b := canvas.NewRectangle(color.Transparent)
	b.StrokeWidth = PaneBorderWidth
	b.StrokeColor = color.Transparent

	p := &Pane{
		id:     id,
		term:   term,
		split:  SplitNone,
		first:  nil,
		second: nil,
		parent: parent,
		border: b,
		screen: screen,
	}

	if p.parent == nil {
		p.lastFocus = p
		p.root = p
	} else {
		p.root = p.parent.root
	}

	p.ExtendBaseWidget(p)

	term.OnExit = p.TryClose
	term.SetScreen(p.screen)
	//term.screen = p.screen
	if run {
		go func() {
			term.RunCmd("xxx")
			fmt.Println("xxxxxxxxx exit", p.id)
		}()
	}

	id++

	return p
}

// Called by TabControl
func (p *Pane) TearDown() {
	if p.split == SplitNone {
		//t := p.term
		//t.OnExit = nil
		//t.Exit()
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
	//pa.term = take.term

	if pa.split == SplitNone {
		pa.term = take.term
		//pa.term.SetScreen(take.screen)
		pa.term.OnExit = pa.TryClose
	} else {
		pa.first.parent = pa
		pa.second.parent = pa
	}

	pa.root.Refresh()
}

func (p *Pane) TryClose() {
	fyne.Do(func() {
		p.Close()
	})
}

func (p *Pane) Split(direction int, ratio float32) {
	p.split = direction
	p.first = NewPane(p, p.term, p.screen, false)
	p.second = NewPane(p, nil, nil, true)
	p.term = nil

	p.divider = NewDivider(p)
	p.ratio = ratio
	p.Refresh()

	p.root.lastFocus = p.second

	fyne.Do(func() {
		p.second.TryFocus()
	})
}

func (p *Pane) SetRatio(ratio float32) {
	p.ratio = ratio
	p.Refresh()
}

func (p *Pane) Resize(size fyne.Size) {
	if size.Width <= 0 || size.Height <= 0 || size == p.size {
		return
	}

	p.size = size
	if p.term != nil {
		maxPos := fyne.NewPos(size.Width-2*PanePadding, size.Height-2*PanePadding)
		r, c := p.screen.CursorLocationForPosition(maxPos)
		p.term.Resize(r, c, size.Width-2*PanePadding, size.Height-2*PanePadding)
	}

	p.BaseWidget.Resize(size)
}

func (p *Pane) Tapped(pe *fyne.PointEvent) {
	if c := fyne.CurrentApp().Driver().CanvasForObject(p); c != nil {
		c.Focus(p)
	}
}

func (p *Pane) TappedSecondary(pe *fyne.PointEvent) {
	if c := fyne.CurrentApp().Driver().CanvasForObject(p); c != nil {
		c.Focus(p)
	}

	if c := fyne.CurrentApp().Driver().CanvasForObject(p); c != nil {
		hsplitItem := fyne.NewMenuItemWithIcon("Horizontal Split", theme.NewThemedResource(assert.HSplitIconRes), func() {
			p.Split(SplitHorizontal, 0.5)
		})
		vsplitItem := fyne.NewMenuItemWithIcon("Vertical Split", theme.NewThemedResource(assert.VSplitIconRes), func() {
			p.Split(SplitVertical, 0.5)
		})

		separatorItem := fyne.NewMenuItemSeparator()

		closePaneIterm := fyne.NewMenuItemWithIcon("Close Pane", theme.Icon(theme.IconNameWindowClose), func() {
			p.term.OnExit = nil
			p.term.Exit()
			p.TryClose()
		})
		closeTabIterm := fyne.NewMenuItemWithIcon("Close Tab", theme.Icon(theme.IconNameWindowClose), func() {
			p.root.TearDown()
			if f := p.root.OnTearDown; f != nil {
				f()
			}
		})

		showMarkDownIterm := fyne.NewMenuItemWithIcon("Show as Markdown", theme.NewThemedResource(assert.MarkdownIconRes), func() {
			NewMarkdownTab(make([]rune, 1, 1))
		})

		typeAgentItem := fyne.NewMenuItemWithIcon("Type Agent", theme.Icon(theme.IconNameAccount), func() {
		})

		menuItems := []*fyne.MenuItem{hsplitItem, vsplitItem, separatorItem, closePaneIterm, closeTabIterm, separatorItem, showMarkDownIterm, typeAgentItem}
		popUpMenu := widget.NewPopUpMenu(fyne.NewMenu("", menuItems...), c)
		popUpMenu.ShowAtRelativePosition(pe.Position, p)
	}
}

func (p *Pane) FocusGained() {
	p.root.lastFocus = p
	w, h := p.term.Size()
	fmt.Println("Focus: ", p.id, p.Size(), w, h)
	if p.parent != nil {
		p.border.StrokeColor = theme.Color(theme.ColorNamePrimary)
		p.border.Refresh()
	}
}

func (p *Pane) FocusLost() {
	p.border.StrokeColor = color.Transparent
	p.border.Refresh()
}

func (p *Pane) TypedRune(r rune) {
	p.screen.ScrollToBottom()
	b := make([]byte, utf8.UTFMax)
	size := utf8.EncodeRune(b, r)
	_, _ = p.term.Write(b[:size])
}

func (p *Pane) TypedKey(ke *fyne.KeyEvent) {
	p.screen.ScrollToBottom()
	switch ke.Name {
	case fyne.KeyReturn:
		_, _ = p.term.Write([]byte{'\r'})

	// DEL, as xterm sends. ConPTY on Windows reads 0x08 as Ctrl+Backspace.
	case fyne.KeyBackspace:
		_, _ = p.term.Write([]byte{0x7f})
	}
}

func (p *Pane) AcceptsTab() bool {
	return true
}

func (p *Pane) CreateRenderer() fyne.WidgetRenderer {
	r := &paneRenderer{
		pane: p,
	}

	return r
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
		pad := fyne.NewSize(PanePadding, PanePadding)
		offset := fyne.NewPos(PanePadding, PanePadding)

		r.pane.border.Resize(size)
		r.pane.border.Move(fyne.NewPos(0, 0))

		r.pane.screen.Resize(size.Subtract(pad).Subtract(pad))
		r.pane.screen.Move(offset)

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
	return fyne.NewSize(w, w)
}

func (r *paneRenderer) Objects() []fyne.CanvasObject {
	if r.pane.split == SplitNone {
		return []fyne.CanvasObject{r.pane.border, r.pane.screen}
	} else {
		return []fyne.CanvasObject{r.pane.first, r.pane.divider, r.pane.second}
	}
}

func (r *paneRenderer) Refresh() {
	switch r.pane.split {
	case SplitNone:
		r.pane.border.Refresh()
	case SplitHorizontal, SplitVertical:
		r.pane.divider.Refresh()
		r.pane.first.Refresh()
		r.pane.second.Refresh()
	}
	r.Layout(r.pane.Size())
}
