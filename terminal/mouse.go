package terminal

import (
	"fmt"
	"unicode/utf8"
)

/*
 * ================================
 *  Mouse reporting: a program (e.g. vim, htop, tmux) asks for the mouse with
 *  a DEC private mode, then the clicks, the wheel and the moves go to it as
 *  escape sequences, see
 *  https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h2-Mouse-Tracking
 * ================================
 */

// What a program asks to get, by the mode it set
const (
	mouseOff    = 0
	mouseX10    = 9    // presses only
	mouseNormal = 1000 // presses, releases and the wheel
	mouseButton = 1002 // and moves while a button is down
	mouseAny    = 1003 // and every move
)

// How the report is written, by the mode the program set
const (
	mouseEncDefault = 0    // CSI M and three bytes, up to column/row 223
	mouseEncUTF8    = 1005 // CSI M and three UTF-8 characters, up to 2015
	mouseEncSGR     = 1006 // CSI < b;x;y M (press) or m (release)
	mouseEncURXVT   = 1015 // CSI b;x;y M
)

// MouseAction is what the mouse did
type MouseAction int

const (
	MousePress MouseAction = iota
	MouseRelease
	MouseMotion
	MouseWheel
)

// The buttons of a press, release or motion (MouseNone for a motion with no
// button down), and the directions of the wheel
const (
	MouseLeft      = 0
	MouseMiddle    = 1
	MouseRight     = 2
	MouseNone      = 3
	MouseWheelUp   = 0
	MouseWheelDown = 1
)

// The modifier keys held, added to the button code
const (
	MouseShift = 4
	MouseAlt   = 8
	MouseCtrl  = 16
)

// setMouseMode sets or resets a mouse tracking mode or encoding
func (t *Terminal) setMouseMode(mode int, on bool) {
	switch mode {
	case mouseX10, mouseNormal, mouseButton, mouseAny:
		if on {
			t.mouseMode = mode
		} else if t.mouseMode == mode {
			t.mouseMode = mouseOff
		}
	case mouseEncUTF8, mouseEncSGR, mouseEncURXVT:
		if on {
			t.mouseEnc = mode
		} else if t.mouseEnc == mode {
			t.mouseEnc = mouseEncDefault
		}
	}
}

// mouseModeSet tells a mouse mode or encoding is set, for DECRQM
func (t *Terminal) mouseModeSet(mode int) bool {
	return t.mouseMode == mode || t.mouseEnc == mode
}

// MouseTracking tells the program gets the mouse, then a press goes to it
// and not to the UI, e.g. to select
func (t *Terminal) MouseTracking() bool {
	t.stateMu.Lock()
	defer t.stateMu.Unlock()
	return t.mouseMode != mouseOff
}

// SendMouse reports what the mouse did at a cell (0 based) to the program,
// false when its mode does not report that, e.g. a move in mode 1000. button
// is a Mouse button for a press, release or motion, a MouseWheel direction
// for the wheel; mods are the Mouse modifiers.
func (t *Terminal) SendMouse(action MouseAction, button, row, col, mods int) bool {
	t.stateMu.Lock()
	mode, enc := t.mouseMode, t.mouseEnc
	t.stateMu.Unlock()

	switch mode {
	case mouseOff:
		return false
	case mouseX10:
		// Presses only, without the modifiers
		if action != MousePress {
			return false
		}
		mods = 0
	case mouseNormal:
		if action == MouseMotion {
			return false
		}
	case mouseButton:
		if action == MouseMotion && button == MouseNone {
			return false
		}
	}

	code := button
	switch action {
	case MouseRelease:
		// Only SGR tells which button went up
		if enc != mouseEncSGR {
			code = MouseNone
		}
	case MouseMotion:
		code += 32
	case MouseWheel:
		code += 64
	}
	code |= mods

	x, y := col+1, row+1
	var seq string
	switch enc {
	case mouseEncSGR:
		final := 'M'
		if action == MouseRelease {
			final = 'm'
		}
		seq = fmt.Sprintf("\x1b[<%d;%d;%d%c", code, x, y, final)
	case mouseEncURXVT:
		seq = fmt.Sprintf("\x1b[%d;%d;%dM", code+32, x, y)
	case mouseEncUTF8:
		if x > 2015 || y > 2015 {
			return true // can not be told, dropped like xterm does
		}
		b := []byte("\x1b[M")
		for _, v := range []int{code + 32, x + 32, y + 32} {
			b = utf8.AppendRune(b, rune(v))
		}
		seq = string(b)
	default:
		if x > 223 || y > 223 {
			return true
		}
		seq = string([]byte{0x1b, '[', 'M', byte(code + 32), byte(x + 32), byte(y + 32)})
	}

	_, _ = t.Write([]byte(seq))
	return true
}
