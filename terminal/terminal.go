package terminal

import (
	"yaterm/pty"
)

type Terminal struct {
	row, col int
	pty *pty.Pty

	OnExit func()
}

func New() *Terminal {
	t := &Terminal{
		pty: pty.New(),
	}

	return t
}

func (t *Terminal) Resize(row, col int, width, height float32) {
	t.row = row
	t.col = col
	if t.pty != nil {
		t.pty.Resize(row, col, width, height)
	}
}

func (t *Terminal) Size() (int, int) {
	return t.row, t.col
}
