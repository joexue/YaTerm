package ui

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/lang"
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
	SplitMerging
)

var id int = 0

type Pane struct {
	widget.BaseWidget

	id        int
	container *fyne.Container
	term      *terminal.Terminal

	split  int
	first  *Pane
	second *Pane
	leaf   bool

	root      *Pane
	parent    *Pane
	lastFocus *Pane

	border *canvas.Rectangle

	// Callback from creater, called when the whole pane tree is finished
	OnTearDown func()
}

func NewPane(parent *Pane, term *terminal.Terminal, run bool) *Pane {
	id++

	if term == nil {
		term = terminal.New()
	}

	b := canvas.NewRectangle(color.Transparent)
	b.StrokeWidth = 2
	b.StrokeColor = color.Transparent

	c := container.NewStack(b, container.NewClip(container.NewPadded(term)))
	p := &Pane{
		id:        id,
		term:      term,
		leaf:      true,
		split:     SplitNone,
		first:     nil,
		second:    nil,
		parent:    parent,
		border:    b,
		container: c,
	}

	// Both event glass and term use padding to let them align
	p.container.Add(container.NewPadded(NewEventGlass(p)))

	if p.parent == nil {
		p.lastFocus = p
		p.root = p
	} else {
		p.root = p.parent.root
	}

	p.ExtendBaseWidget(p)

	if run {
		go func() {
			_ = term.RunLocalShell()
			p.TryClose()
		}()
	}

	/*
		go func() {
			fyne.Do(func() {
				if c := fyne.CurrentApp().Driver().CanvasForObject(p); c != nil {
					c.Focus(p)
				}
			})
		}()
	*/
	return p
}

func MergeOrClosePane(pane *Pane, term *terminal.Terminal) {
	/*
		if pane.term == term {
			if f := pane.OnClose; f != nil {
				pane.OnClose()
			}
			return
		}

		var p *Pane = nil
		if pane.first != nil && pane.first.term == term {
			p = pane.second
		}

		if pane.second != nil && pane.second.term == term {
			p = pane.first
		}

		if p != nil {
			pane.term = p.term
			pane.first = nil
			pane.second = nil

			fmt.Printf("remvoe p:%d take:%d term:%p \n", p.id, pane.id, pane.term)
			pane.term.OnTappedSecondary = func(pe *fyne.PointEvent) {
				pane.TappedSecondary(pe)
			}

			pane.term.OnFocusLost = func() {
				fmt.Printf("lost xxxx %d term:%p\n", pane.id, pane.term)
				pane.FocusLost()
			}

			pane.container.RemoveAll()
			pane.container.Refresh()
			pane.container.Add(pane.border)
			pane.container.Add(container.NewClip(container.NewPadded(pane.term)))
			pane.container.Refresh()

			//fyne.Do(func() {
			//pane.Focus()
			//})
			return
		}

		if pane.first != nil {
			MergeOrClosePane(pane.first, term)
		}

		if pane.second != nil {
			MergeOrClosePane(pane.second, term)
		}
	*/
}

// Called by TabControl
func (p *Pane) TearDown() {
	if p.leaf {
		t := p.term
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

func (p *Pane) Split(direction int) {
	p.split = direction
	p.leaf = false

	p.first = NewPane(p, p.term, false)
	p.term = nil

	p.second = NewPane(p, nil, true)

	p.container.RemoveAll()
	if direction == SplitHorizontal {
		p.container.Add(container.NewHSplit(p.first, p.second))
	} else if direction == SplitVertical {
		p.container.Add(container.NewVSplit(p.first, p.second))
	}

	p.Refresh()

	p.root.lastFocus = p.second

	fyne.Do(func() {
		p.second.TryFocus()
	})
}

func (p *Pane) Close() {
	fmt.Println("closing pane : ", p.id)
	if p.split == SplitMerging {
		return
	}

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

	pa.container.RemoveAll()
	if take.leaf {
		pa.leaf = true
		pa.split = SplitNone
		pa.first = nil
		pa.second = nil
		pa.term = take.term
		pa.container.Add(take.container.Objects[0])
		pa.container.Add(take.container.Objects[1])
		pa.container.Add(take.container.Objects[2])
	} else {
		pa.leaf = false
		pa.split = take.split
		pa.first = take.first
		pa.second = take.second
		pa.term = nil
		if pa.split == SplitHorizontal {
			pa.container.Add(container.NewHSplit(take.first, take.second))
		} else {
			pa.container.Add(container.NewVSplit(take.first, take.second))
		}
	}

	p.split = SplitMerging
}

func (p *Pane) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(p.container)
}

func (p *Pane) Tapped(pe *fyne.PointEvent) {
	// Do nothing for now
}

func (p *Pane) TappedSecondary(pe *fyne.PointEvent) {
	if c := fyne.CurrentApp().Driver().CanvasForObject(p); c != nil {
		hsplitItem := fyne.NewMenuItemWithIcon(lang.L("Horizontal Split"), assert.HSplitIconRes, func() {
			p.Split(SplitHorizontal)
		})
		vsplitItem := fyne.NewMenuItemWithIcon(lang.L("Vertical Split"), assert.VSplitIconRes, func() {
			p.Split(SplitVertical)
		})

		separatorItem := fyne.NewMenuItemSeparator()

		closePaneIterm := fyne.NewMenuItemWithIcon(lang.L("Close Pane"), assert.VSplitIconRes, func() {
			p.TryClose()
			p.term.Exit()
		})
		closeTabIterm := fyne.NewMenuItemWithIcon(lang.L("Close Tab"), assert.VSplitIconRes, func() {
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
	fmt.Println("Focus: ", p.id)
	if p.parent != nil {
		p.border.StrokeColor = theme.Color(theme.ColorNamePrimary)
		p.border.Refresh()
	}

	p.term.FocusGained()
}

// FocusLost is called (via the terminal's OnFocusLost) when this pane's
// terminal is no longer the focused object on the canvas.
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
