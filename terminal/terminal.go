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

	// SetWrapped marks a row as continuing on the next one (auto wrapped), or not
	SetWrapped(row int, wrapped bool)

	// Reflow fits the content to a new size, re-wrapping lines at the new
	// width, and returns where the cursor moved to.
	Reflow(row, col, curRow, curCol int) (int, int)
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

	mu sync.Mutex // guards writing to the pty

	// stateMu guards the terminal state below, output is processed on the
	// reader goroutine and Resize comes from the UI thread.
	stateMu sync.Mutex

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
	// A pane squeezed to nothing would reflow all text away, keep the old size
	if row <= 0 || col <= 0 {
		return
	}

	t.stateMu.Lock()
	if row != t.row || col != t.col {
		// The screen re-wraps its lines, the cursor moves with its text
		t.cursorY, t.cursorX = t.screen.Reflow(row, col, t.cursorY, t.cursorX)
		t.wrapPending = false
		t.row, t.col = row, col
		t.screen.Flush(t.cursorY, t.cursorX)
	}
	t.stateMu.Unlock()

	if t.pty != nil {
		t.pty.Resize(row, col)
	}
}

func (t *Terminal) Size() (int, int) {
	t.stateMu.Lock()
	defer t.stateMu.Unlock()

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
