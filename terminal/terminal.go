package terminal

import (
	"strings"
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

	// ScrollRegion scrolls rows top..bottom up by n lines (down when n < 0),
	// lines leaving the region are dropped, not kept in the history
	ScrollRegion(top, bottom, n int)

	// InsertCells and DeleteCells shift the rest of a row right or left by n,
	// the new cells are blank with background bg
	InsertCells(row, col, n int, bg color.Color)
	DeleteCells(row, col, n int, bg color.Color)

	// SetWrapped marks a row as continuing on the next one (auto wrapped), or not
	SetWrapped(row int, wrapped bool)

	// Reflow fits the content to a new size, re-wrapping lines at the new
	// width, and returns where the cursor moved to.
	Reflow(row, col, curRow, curCol int) (int, int)

	// SetAlternate switches to the alternate screen (on), a blank one with
	// no history, e.g. for vim, keeping the main screen and the cursor on it
	// (curRow, curCol). Back to the main screen it returns that cursor,
	// moved by the reflows done meanwhile.
	SetAlternate(on bool, curRow, curCol int) (int, int)
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
	state int

	pty    Pty
	screen Screen

	writeMu sync.Mutex // guards writing to the pty

	// stateMu guards the terminal state below, output is processed on the
	// reader goroutine and Resize comes from the UI thread.
	stateMu sync.Mutex

	row, col int

	cursorX, cursorY   int
	savedRow, savedCol int
	// "ESC 7" saves the character sets with the cursor, like xterm
	savedCharsets [2]rune
	savedShifted  int
	wrapPending   bool

	// Scroll region set by "CSI top;bottom r", 0 based and inclusive.
	// Full screen when unset, bottom 0 means the last row.
	scrollTop, scrollBottom int

	// Character sets of G0 and G1 ("ESC ( X", "ESC ) X"), 0 means ASCII,
	// and which one is in use, 0 for G0 (SI) or 1 for G1 (SO)
	charsets [2]rune
	shifted  int
	param    []rune
	leftOver []byte // the start of a UTF-8 character the last read ended in

	fg, bg color.Color
	style  Style

	colorSchema [16]color.Color

	cwd   string // set by OSC 7
	title string // set by OSC 0 or 2

	// Callbacks to the UI, called on the reader goroutine with the state lock
	// held, so they must not call back into the terminal.
	// OnTitleChange gets the new title (OSC 0/2), OnClipboard text a program
	// puts into the clipboard (OSC 52), OnQueryColor the color asked for by
	// OSC 10 (text), 11 (background) or 12 (cursor).
	OnTitleChange func(title string)
	OnClipboard   func(text string)
	OnQueryColor  func(code int) color.Color

	OnExit func()

	// OnTmuxControl is asked for a handler when tmux starts its control mode
	// (tmux -CC), nil or a nil handler means it is not allowed. Called on the
	// reader goroutine with the state lock held.
	OnTmuxControl func() TmuxHandler

	tmux TmuxHandler // the control mode in use, nil when none

	// alternate tells the alternate screen is shown, see setAlternate
	alternate bool
}

func New(screen Screen) *Terminal {
	t := &Terminal{
		pty:    pty.New(),
		screen: screen,
		title:  "",
	}

	return t
}

// NewWithPty makes a terminal on a given pty, e.g. one that sends the input
// to a tmux pane, its output is given with ProcessOutput, RunCmd is not used
func NewWithPty(screen Screen, p Pty) *Terminal {
	return &Terminal{
		pty:    p,
		screen: screen,
	}
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
		// The old region does not fit the new size, the program sets it again
		t.scrollTop, t.scrollBottom = 0, 0
		t.screen.Flush(t.cursorShown())
	}
	// In tmux control mode the pty keeps its size: tmux is told its size by
	// commands. And ConPTY (Windows) asks for the cursor position ("ESC [6n")
	// on a resize, to sync its copy of the screen. That ESC between the lines
	// of tmux would be taken as the start of the ST and end the mode. endTmux
	// gives the pty the size then.
	inTmux := t.tmux != nil
	t.stateMu.Unlock()

	if t.pty != nil && !inTmux {
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

	// The program ended inside the tmux control mode, e.g. its pane closed
	t.stateMu.Lock()
	if t.tmux != nil {
		t.endTmux()
	}
	t.stateMu.Unlock()

	if f := t.OnExit; f != nil {
		f()
	}

	return
}

func (t *Terminal) Exit() {
	t.pty.Close()
}

func (t *Terminal) Write(bytes []byte) (int, error) {
	t.writeMu.Lock()
	defer t.writeMu.Unlock()
	return t.pty.Write(bytes)
}

func (t *Terminal) Read(bytes []byte) (int, error) {
	return t.pty.Read(bytes)
}

func (t *Terminal) GetScreen() Screen {
	return t.screen
}

func (t *Terminal) setTitle(title string) {
	t.title = title
	if f := t.OnTitleChange; f != nil {
		f(title)
	}
}

func (t *Terminal) setCwd(cwd string) {
	t.cwd = cwd
}

func (t *Terminal) SetColors(schema [16]color.Color) {
	t.colorSchema = schema
}

// Print shows a message of YaTerm on the screen, plain text and new lines.
// It goes past the parser, which may be in the middle of something, e.g. the
// tmux control mode would take it as a line for tmux. It takes the state
// lock, so not for the reader goroutine while it holds it (e.g. in a
// callback).
func (t *Terminal) Print(text string) {
	text = strings.ReplaceAll(text, "\n", "\r\n")

	t.stateMu.Lock()
	defer t.stateMu.Unlock()
	for _, r := range text {
		t.processGround(r)
	}
	t.screen.Flush(t.cursorShown())
}

// Title is the title the program set with OSC 0 or 2, empty if none
func (t *Terminal) Title() string {
	t.stateMu.Lock()
	defer t.stateMu.Unlock()

	return t.title
}
