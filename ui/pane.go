package ui

import (
    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/container"
    //"github.com/fyne-io/terminal"
    "yaterm/terminal"
)

const (
    None = iota
    Horizontal
    Virtical
)

type Pane struct {
    container fyne.CanvasObject
    splitDirection int
    term *terminal.Terminal

    leading *Pane
    trailing *Pane

    // Callback for TabControl or parent pane
    OnClose func()
}

func NewPane(_ fyne.App, _ fyne.Window, _ *TabControl) *Pane {
    t := terminal.New()
    c := container.NewMax(t)
    p := &Pane {
        container: c,
        splitDirection: None,
        term: t,
        leading: nil,
        trailing: nil,
    }

    go func() {
        _ = t.RunLocalShell()
		if f := p.OnClose; f != nil {
            p.OnClose()
        }
    }()

    return p
}

func (p *Pane) GetContainer() fyne.CanvasObject {
    return p.container
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
