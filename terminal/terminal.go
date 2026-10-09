package terminal

import (
	//"fmt"
	"image/color"
	"sync"
	"yaterm/pty"
)

const bufLen = 32768

type Screen interface {
	Project(rune, int, int, color.Color, color.Color, Style)
	Flush(int, int)
	ScrollUp()
	MoveCursor(int, int)
}

type Pty interface {
	Resize(int, int)
	RunCmd(string) error
	Read(bytes []byte) (int, error)
	Write(bytes []byte) (int, error)
	Close()
}

// Style is the text style set by SGR "CSI ... m"
type Style struct {
	Bold, Faint, Italic, Underline, Blinking bool
	Inverse, Hidden, Strikethrough           bool
}

type Terminal struct {
	pty    Pty
	screen Screen

	mu sync.Mutex

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

	OnExit func()
}

func New(screen Screen) *Terminal {
	t := &Terminal{
		pty:    pty.New(),
		screen: screen,
	}

	return t
}

func (t *Terminal) Resize(row, col int) {
	t.row, t.col = row, col
	if t.pty != nil {
		t.pty.Resize(row, col)
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
			num, err := t.Read(buf)
			if err != nil {
				break
			}

			//fmt.Println("xxxxxxxxxxxxxxx", num, err)
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
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.pty.Write(bytes)
}

func (t *Terminal) Read(bytes []byte) (int, error) {
	return t.pty.Read(bytes)
}

func (t *Terminal) GetScreen() Screen {
	return t.screen
}
