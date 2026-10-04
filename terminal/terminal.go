package terminal

import (
	"fmt"
	"image/color"
	"yaterm/pty"
)

const bufLen = 32768

type Terminal struct {
	row, col int
	pty      *pty.Pty

	cursorX int
	cursorY int

	savedRow int
	savedCol int

	state  int
	escape bool

	csiCode string
	oscCode string

	fg, bg                                           color.Color
	bold, italic, underline, strikethrough, blinking bool

	OnExit    func()
	OnProject func(rune, int, int, color.Color, color.Color)
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
	fmt.Println("xxxxxxxxxxxxxxxxx resize", row, col)
	if t.pty != nil {
		t.pty.Resize(row, col, width, height)
	}
}

func (t *Terminal) Size() (int, int) {
	return t.row, t.col
}

func (t *Terminal) RunCmd(cmd string) {
	err := t.pty.RunCmd(cmd)

	if err != nil {
	} else {
		buf := make([]byte, bufLen)
		for {
			num, err := t.pty.Read(buf)
			if err != nil {
				break
			}

			fmt.Println("xxxxxxxxxxxxxxx", num, err)
			t.ProcessOutput(buf[:num], num)
		}

	}

	if f := t.OnExit; f != nil {
		f()
	}

	return
}

func (t *Terminal) Exit() {
	t.pty.Close()
}

func (t *Terminal) Write(bytes []byte) (int, error) {
	return t.pty.Write(bytes)
}
