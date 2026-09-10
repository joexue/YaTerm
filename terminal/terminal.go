package terminal

import (
    "fmt"
    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/driver/desktop"
    "github.com/fyne-io/terminal"
)

type Terminal struct {
    terminal.Terminal
}

func New() *Terminal {
	t := &Terminal{}
	t.ExtendBaseWidget(t)
	return t
}

func (t *Terminal) TappedSecondary(pe *fyne.PointEvent) {
    fmt.Println("Tapped")
}

func (t *Terminal) MouseDown(ev *desktop.MouseEvent) {
    if ev.Button == desktop.MouseButtonSecondary {
        if c := fyne.CurrentApp().Driver().CanvasForObject(t); c != nil {
            c.Focus(t)
        }
        return
    }


	if ev.Button == desktop.MouseButtonTertiary {
        ev.Button = desktop.MouseButtonSecondary
    }

    t.Terminal.MouseDown(ev)
}

