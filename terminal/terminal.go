package terminal

import (
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

func (t *Terminal) MouseDown(ev *desktop.MouseEvent) {
	if ev.Button == desktop.MouseButtonSecondary {
		return
	}

	if ev.Button == desktop.MouseButtonTertiary {
		ev.Button = desktop.MouseButtonSecondary
	}

	t.Terminal.MouseDown(ev)
}
