package ui

import (
    "fmt"
    "image/color"
    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/canvas"
    "fyne.io/fyne/v2/container"
    "fyne.io/fyne/v2/theme"
    "fyne.io/fyne/v2/widget"
    "yaterm/terminal"
    "fyne.io/fyne/v2/lang"
    "yaterm/assert"
)

const (
    None = iota
    Horizontal
    Vertical
)

var id int = 0
type Pane struct {
	widget.BaseWidget

    id int
    container *fyne.Container
    splitDirection int
    term *terminal.Terminal

    leading *Pane
    trailing *Pane

    root *Pane
    parent *Pane
    lastFocus *Pane

	border *canvas.Rectangle
    focused bool

    // Callback for TabControl or parent pane
    OnClose func()
}

func MergeOrClosePane(pane *Pane, term *terminal.Terminal) {
    if pane.term == term {
        if f := pane.OnClose; f != nil {
            pane.OnClose()
        }
        return
    }

    var p *Pane = nil
    if pane.leading != nil && pane.leading.term == term {
        p = pane.trailing
    }

    if pane.trailing != nil && pane.trailing.term == term {
        p = pane.leading
    }

    if p != nil {
        pane.term = p.term
        pane.leading = nil
        pane.trailing = nil

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

    if pane.leading != nil {
        MergeOrClosePane(pane.leading, term)
    }

    if pane.trailing != nil {
        MergeOrClosePane(pane.trailing, term)
    }
}

func NewPane(parent *Pane, term *terminal.Terminal, run bool) *Pane {
    id++
    if term == nil {
        term = terminal.New()
    }

	border := canvas.NewRectangle(color.Transparent)
	border.StrokeWidth = theme.Padding() / 2 //theme.DefaultTheme().Size(theme.SizeNameInputBorder) * 2
	border.StrokeColor = color.Transparent //theme.DefaultTheme().Color(theme.ColorNameInputBorder, theme.VariantDark)
	border.CornerRadius = 0//theme.DefaultTheme().Size(theme.SizeNameInputRadius)

    // The terminal's TextGrid renders with Scroll = ScrollNone, which means
    // Fyne gives it no clip region: rows that were laid out while the
    // terminal was wider than it is now keep painting past the pane's
    // right/bottom edge instead of being cut off. Once a pane is split and
    // shrinks, that overflow bleeds straight into the sibling pane, making
    // it look like the new pane's content is being overwritten by the old
    // one. Wrapping the terminal in container.NewClip confines its painting
    // to the pane's actual bounds.
    c := container.NewStack(border, container.NewClip(container.NewPadded(term)))
    p := &Pane {
        id: id,
        container: c,
        splitDirection: None,
        term: term,
        leading: nil,
        trailing: nil,
        parent: parent,
        border: border,
    }

    p.container.Add(NewEventGlass(p))

    if p.parent == nil {
        p.lastFocus = p
        p.root = p
    } else {
        p.root = p.parent.root
    }

	p.ExtendBaseWidget(p)

    //t.OnSplit = func(direction int) {
    //    p.Split(direction)
    //}

    if run {
        go func() {
            _ = term.RunLocalShell()
            //if f := p.OnClose; f != nil {
            //    p.OnClose()
            //}
            fyne.Do(func() {
                MergeOrClosePane(p.root, term)
            })
        }()
    }

    return p
}

// Called by TabControl
func (p *Pane) Close() {
    t := p.term
    t.Exit()
}

// Called by TabControl
func (p *Pane) Focus() {
/*
    var focusPane *Pane

    if p.term != nil {
        focusPane = p
    } else {
        focusPane = p.root.lastFocus
    }
    //TODO: find the fouced pane in the pane tree
	if c := fyne.CurrentApp().Driver().CanvasForObject(focusPane.term); c != nil {
		c.Focus(focusPane.term)
	}
    */
}


func (p *Pane) Split(direction int) {
    p.splitDirection = direction

    p.leading = NewPane(p, p.term, false)

    p.leading.term.OnTappedSecondary = func(pe *fyne.PointEvent) {
        p.leading.TappedSecondary(pe)
    }

    p.leading.term.OnFocusGained = func() {
        p.leading.FocusGained()
    }

    p.leading.term.OnFocusLost = func() {
        p.leading.FocusLost()
    }

    p.term = nil

    p.trailing = NewPane(p, nil, true)

    p.container.RemoveAll()
    if direction == Horizontal {
        p.container.Add(container.NewHSplit(p.leading, p.trailing))
    } else if direction == Vertical {
        p.container.Add(container.NewVSplit(p.leading, p.trailing))
    }

    // Container.Add() only re-lays-out the children; it does not tell the
    // canvas to repaint. Without this, the old (pre-split) terminal content
    // stays on screen until something else forces a redraw, and the new
    // pane's content ends up painted over the stale pixels instead of into
    // freshly cleared space.
    p.Refresh()

    p.root.lastFocus = p.trailing

    fyne.Do(func() {
        p.trailing.Focus()
    })
}

func (p *Pane) CreateRenderer() fyne.WidgetRenderer {
    return widget.NewSimpleRenderer(p.container)
}


func (p *Pane) Tapped(pe *fyne.PointEvent) {
}

func (p *Pane) TappedSecondary(pe *fyne.PointEvent) {
	if c := fyne.CurrentApp().Driver().CanvasForObject(p); c != nil {
        hsplitItem := fyne.NewMenuItemWithIcon(lang.L("Horizontal Split"), assert.HSplitIconRes, func() {
            p.Split(Horizontal)
        })
        vsplitItem := fyne.NewMenuItemWithIcon(lang.L("Vertical Split"), assert.VSplitIconRes, func() {
            p.Split(Vertical)
        })

        menuItems := []*fyne.MenuItem {hsplitItem, vsplitItem}
	    popUpMenu := widget.NewPopUpMenu(fyne.NewMenu("", menuItems...), c)
	    popUpMenu.ShowAtRelativePosition(pe.Position, p)
    }
}

func (p *Pane) FocusGained() {
    fmt.Println("gain id: ", p.id)
    p.root.lastFocus = p
    p.focused = true

    if p.parent != nil {
        p.border.StrokeColor = theme.DefaultTheme().Color(theme.ColorNamePrimary, theme.VariantDark)
        p.border.Refresh()
    }
}

// FocusLost is called (via the terminal's OnFocusLost) when this pane's
// terminal is no longer the focused object on the canvas.
func (p *Pane) FocusLost() {
    fmt.Println("lost id: ", p.id)
    p.focused = false
    p.border.StrokeColor = color.Transparent //theme.DefaultTheme().Color(theme.ColorNameInputBorder, theme.VariantDark)
    p.border.Refresh()
}

func (p *Pane) TypedRune(r rune) {
    p.term.TypedRune(r)
}

func (p *Pane) TypedKey(ke *fyne.KeyEvent) {
    p.term.TypedKey(ke)
}
