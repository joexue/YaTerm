package terminal

import (
	"fmt"
	"image/color"
	"yaterm/pty"
)

const bufLen = 32768

type Screen interface {
	Project(rune, int, int, color.Color, color.Color, Style)
	Flush(int, int)
	ScrollUp()
	MoveCursor(int, int)
}

// Style is the text style set by SGR "CSI ... m"
type Style struct {
	Bold, Faint, Italic, Underline, Blinking bool
	Inverse, Hidden, Strikethrough           bool
}

type Terminal struct {
	pty *pty.Pty

	row, col int

	cursorX, cursorY   int
	savedRow, savedCol int
	wrapPending        bool

	state           int
	decState        int
	escape          bool
	controlSequence string

	fg, bg color.Color
	style  Style

	screen Screen

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

func (t *Terminal) SetScreen(s Screen) {
	t.screen = s
}
