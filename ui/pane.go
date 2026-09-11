package ui

import (
    "fmt"
    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/container"
    "fyne.io/fyne/v2/widget"
    "yaterm/terminal"
)

const (
    None = iota
    Horizontal
    Vertical
)

type Pane struct {
	widget.BaseWidget

    container fyne.CanvasObject
    splitDirection int
    term *terminal.Terminal

    leading *Pane
    trailing *Pane

    // Callback for TabControl or parent pane
    OnClose func()
}

func NewPane() *Pane {
    t := terminal.New()

    c := container.NewMax(t)
    p := &Pane {
        container: c,
        splitDirection: None,
        term: t,
        leading: nil,
        trailing: nil,
    }

	p.ExtendBaseWidget(p)

    t.OnSplit = func(direction int) {
        p.Split(direction)
    }

    go func() {
        _ = t.RunLocalShell()
        if f := p.OnClose; f != nil {
            p.OnClose()
        }
    }()

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

func (p *Pane) Split(direction int) {
    fmt.Println(direction)
    if direction == None {
        return
    }
    /*
    p.splitDirection = direction

    p.leading = NewPane()
    p.trailing = NewPane()

    if direction == Horizontal {
    } else if direction == Vertical {

    }
    */
}

func (p *Pane) CreateRenderer() fyne.WidgetRenderer {
    return widget.NewSimpleRenderer(p.container)
}
