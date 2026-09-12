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

type Pane struct {
	widget.BaseWidget

    container *fyne.Container
    splitDirection int
    term *terminal.Terminal

    leading *Pane
    trailing *Pane

    root *Pane
    parent *Pane

	border *canvas.Rectangle
    focused bool

    // Callback for TabControl or parent pane
    OnClose func()

    // Callbacks for TabControl or parent pane, e.g. to highlight the
    // active tab or track the active leaf pane in a split layout.
    OnFocusGained func()
    OnFocusLost func()
}

func NewPane(parent *Pane, root *Pane, term *terminal.Terminal, run bool) *Pane {
    if term == nil {
        term = terminal.New()
    }

	border := canvas.NewRectangle(color.Transparent)
	border.StrokeWidth = theme.Padding() / 2 //theme.DefaultTheme().Size(theme.SizeNameInputBorder) * 2
	border.StrokeColor = theme.DefaultTheme().Color(theme.ColorNameInputBorder, theme.VariantDark)
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
        container: c,
        splitDirection: None,
        term: term,
        leading: nil,
        trailing: nil,
        parent: parent,
        root: root,

        border: border,
    }

	p.ExtendBaseWidget(p)

    //t.OnSplit = func(direction int) {
    //    p.Split(direction)
    //}

    term.OnTappedSecondary = func(pe *fyne.PointEvent) {
        p.TappedSecondary(pe)
    }
    term.OnFocusGained = func() {
        p.FocusGained()
    }
    term.OnFocusLost = func() {
        p.FocusLost()
    }

    if run {
        go func() {
            _ = term.RunLocalShell()
            if f := p.OnClose; f != nil {
                p.OnClose()
            }
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
    //TODO: find the fouced pane in the pane tree
	if c := fyne.CurrentApp().Driver().CanvasForObject(p.term); c != nil {
		c.Focus(p.term)
	}
}

// FocusGained is called (via the terminal's OnFocusGained) when this pane's
// terminal becomes the focused object on the canvas. It highlights the
// pane's border and notifies OnFocusGained, if set.
func (p *Pane) FocusGained() {
    p.focused = true
    if p.parent != nil {
        p.border.StrokeColor = theme.DefaultTheme().Color(theme.ColorNamePrimary, theme.VariantDark)
        p.border.Refresh()
    }

    if p.OnFocusGained != nil {
        p.OnFocusGained()
    }
}

// FocusLost is called (via the terminal's OnFocusLost) when this pane's
// terminal is no longer the focused object on the canvas.
func (p *Pane) FocusLost() {
    p.focused = false
    p.border.StrokeColor = theme.DefaultTheme().Color(theme.ColorNameInputBorder, theme.VariantDark)
    p.border.Refresh()

    if p.OnFocusLost != nil {
        p.OnFocusLost()
    }
}

// Focused reports whether this pane's terminal currently has focus.
func (p *Pane) Focused() bool {
    return p.focused
}

func (p *Pane) Split(direction int) {
    fmt.Println(direction)
    p.splitDirection = direction

    p.leading = NewPane(p, p.root, p.term, false)
    p.trailing = NewPane(p, p.root, nil, true)

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

    fyne.Do(func() {
        p.trailing.Focus()
    })
}

func (p *Pane) CreateRenderer() fyne.WidgetRenderer {
    return widget.NewSimpleRenderer(p.container)
}

func (p *Pane) TappedSecondary(pe *fyne.PointEvent) {
	if c := fyne.CurrentApp().Driver().CanvasForObject(p); c != nil {
        hsplitItem := fyne.NewMenuItemWithIcon(lang.L("Horizontal Split"), assert.HSplitIconRes, func() {
            fmt.Println("split Horizontal")
            p.Split(Horizontal)
        })
        vsplitItem := fyne.NewMenuItemWithIcon(lang.L("Vertical Split"), assert.VSplitIconRes, func() {
            fmt.Println("split Vertical")
            p.Split(Vertical)
        })

        menuItems := []*fyne.MenuItem {hsplitItem, vsplitItem}
	    popUpMenu := widget.NewPopUpMenu(fyne.NewMenu("", menuItems...), c)
	    popUpMenu.ShowAtRelativePosition(pe.Position, p)
    }
}
