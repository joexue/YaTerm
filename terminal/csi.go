package terminal

import (
	"fmt"
	"strconv"
	"strings"
)

// A CSI longer than this is not a real one, the rest is dropped
const csiMaxLen = 256

// parseParams splits "1;2" into numbers, an empty or invalid field becomes -1
func parseParams(message string) []int {
	if message == "" {
		return nil
	}

	fields := strings.Split(message, ";")
	params := make([]int, len(fields))
	for i, f := range fields {
		n, err := strconv.Atoi(f)
		if err != nil {
			n = -1
		}
		params[i] = n
	}

	return params
}

// eraseInLine: 0 cursor to end, 1 start to cursor, 2 whole line.
// Erasing to the end of a line also ends it, it no longer wraps to the next.
func (t *Terminal) eraseInLine(mode int) {
	switch mode {
	case 0:
		t.eraseCells(t.cursorY, t.cursorX, t.col)
		t.screen.SetWrapped(t.cursorY, false)
	case 1:
		t.eraseCells(t.cursorY, 0, t.cursorX+1)
	case 2:
		t.eraseCells(t.cursorY, 0, t.col)
		t.screen.SetWrapped(t.cursorY, false)
	}
}

// eraseInScreen: 0 cursor to end, 1 start to cursor, 2/3 whole screen
func (t *Terminal) eraseInScreen(mode int) {
	switch mode {
	case 0:
		t.eraseCells(t.cursorY, t.cursorX, t.col)
		t.screen.SetWrapped(t.cursorY, false)
		for row := t.cursorY + 1; row < t.row; row++ {
			t.eraseCells(row, 0, t.col)
			t.screen.SetWrapped(row, false)
		}
	case 1:
		for row := 0; row < t.cursorY; row++ {
			t.eraseCells(row, 0, t.col)
			t.screen.SetWrapped(row, false)
		}
		t.eraseCells(t.cursorY, 0, t.cursorX+1)
	case 2, 3:
		for row := 0; row < t.row; row++ {
			t.eraseCells(row, 0, t.col)
			t.screen.SetWrapped(row, false)
		}
	}
}

// eraseCells blanks columns [from, to) of a row with the current background
func (t *Terminal) eraseCells(row, from, to int) {
	if to > t.col {
		to = t.col
	}

	for col := from; col < to; col++ {
		t.screen.Project(' ', row, col, nil, t.bg, Style{})
	}
}

// handleCsiIntermediate runs a CSI with intermediate bytes, known by the
// intermediates and the final byte together. Unknown ones are ignored.
func (t *Terminal) handleCsiIntermediate(message, inter string, final rune) {
	switch inter + string(final) {
	// DECSTR, soft reset: colors and attributes, scroll region, character
	// sets and the saved cursor back to their defaults
	case "!p":
		t.resetGraphics()
		t.scrollTop, t.scrollBottom = 0, 0
		t.charsets, t.shifted = [2]rune{}, 0
		t.savedRow, t.savedCol = 0, 0
		t.savedCharsets, t.savedShifted = [2]rune{}, 0
		t.wrapPending = false

	// DECRQM, ask if a mode is set: "CSI ? 12 $ p". Modes are not reported
	// yet, the answer is 0, not recognized, so the program does not wait.
	case "$p":
		if mode := strings.TrimPrefix(message, "?"); mode != message {
			t.reply("\x1b[?" + mode + ";0$y")
		} else {
			t.reply("\x1b[" + message + ";0$y")
		}

	// DECSCUSR, cursor shape "CSI n SP q": the cursor is always a block.
	// "CSI 0 % m" and others: not supported, ignored.
	default:
	}
}

// deviceAttributes answers DA1 "CSI c" and DA2 "CSI > c". DA1: a VT220 (62)
// with ANSI color (22). DA2: the generic answer Windows Terminal gives too,
// claiming to be xterm (41) would make programs expect all of its features.
func (t *Terminal) deviceAttributes(message string) {
	switch message {
	case "", "0":
		t.reply("\x1b[?62;22c")
	case ">", ">0":
		t.reply("\x1b[>0;10;1c")
	}
}

// deviceStatus answers DSR "CSI n": 5 asks for the status, 6 for the cursor
// position (1 based), "?6" is the private form of that
func (t *Terminal) deviceStatus(message string) {
	switch message {
	case "5":
		t.reply("\x1b[0n")
	case "6":
		t.reply(fmt.Sprintf("\x1b[%d;%dR", t.cursorY+1, t.cursorX+1))
	case "?6":
		t.reply(fmt.Sprintf("\x1b[?%d;%dR", t.cursorY+1, t.cursorX+1))
	}
}

func (t *Terminal) handleCsi() {
	runes := t.param

	if len(runes) == 0 {
		return
	}

	final := runes[len(runes)-1]
	body := runes[:len(runes)-1]

	// The parameters come first, then the intermediate bytes
	n := 0
	for n < len(body) && !isIntermediate(body[n]) {
		n++
	}
	message, inter := string(body[:n]), body[n:]
	for _, r := range inter {
		if !isIntermediate(r) {
			return // a parameter after an intermediate: not a valid CSI
		}
	}
	if len(inter) > 0 {
		t.handleCsiIntermediate(message, string(inter), final)
		return
	}

	// Queries the program sends to learn about the terminal, most of them
	// with a private prefix, so they are answered before those are skipped
	switch final {
	case 'c':
		t.deviceAttributes(message)
		return
	case 'n':
		t.deviceStatus(message)
		return
	}

	// Private sequences such as "?25l" are not cursor or erase commands
	if message != "" && (message[0] == '?' || message[0] == '>' || message[0] == '=' || message[0] == '<') {
		return
	}

	params := parseParams(message)

	switch final {
	case 'm':
		t.handleColorMode(message)
	case '@':
		t.screen.InsertCells(t.cursorY, t.cursorX, param(params, 0, 1), t.bg)
		t.wrapPending = false
	case 'A':
		t.moveCursor(t.cursorY-param(params, 0, 1), t.cursorX)
	case 'B':
		t.moveCursor(t.cursorY+param(params, 0, 1), t.cursorX)
	case 'C':
		t.moveCursor(t.cursorY, t.cursorX+param(params, 0, 1))
	case 'D':
		t.moveCursor(t.cursorY, t.cursorX-param(params, 0, 1))
	case 'd':
		t.moveCursor(param(params, 0, 1)-1, t.cursorX)
	case 'H', 'f':
		t.moveCursor(param(params, 0, 1)-1, param(params, 1, 1)-1)
	case 'G':
		t.moveCursor(t.cursorY, param(params, 0, 1)-1)
	case 'h':
	//escapePrivateModeOn,
	case 'L':
		t.insertLines(param(params, 0, 1))
	case 'l':
	//escapePrivateModeOff,
	case 'J':
		t.eraseInScreen(param(params, 0, 0))
	case 'K':
		t.eraseInLine(param(params, 0, 0))
	case 'P':
		t.screen.DeleteCells(t.cursorY, t.cursorX, param(params, 0, 1), t.bg)
		t.wrapPending = false
	case 'r':
		t.setScrollRegion(param(params, 0, 1), param(params, 1, 0))
	case 's':
		t.saveCursor()
	case 'S':
		t.scrollRegion(param(params, 0, 1))
	case 'u':
		t.restoreCursor()
	case 'i':
	//escapePrinterMode,
	case 'X':
		n := param(params, 0, 1)
		t.eraseCells(t.cursorY, t.cursorX, t.cursorX+n)
	case 'M':
		t.deleteLines(param(params, 0, 1))
	case 'T':
		t.scrollRegion(-param(params, 0, 1))
	case 'b':
	//escapeRepeatChar,
	case 't':
		//escapeWindowOps,
	}
}

// processCsi collects a CSI: "ESC [", parameter bytes 0x30..0x3f (digits and
// ; : < = > ?), intermediate bytes 0x20..0x2f (space ! " # $ % & ' ( ) * + ,
// - . /), and a final byte 0x40..0x7e that ends it. vim sends e.g.
// "CSI 0 % m" and "CSI 2 SP q", their final byte must not be printed.
func (t *Terminal) processCsi(r rune) {
	switch {
	case r >= 0x20 && r <= 0x3f:
		if len(t.param) < csiMaxLen {
			t.param = append(t.param, r)
		}

	case r >= 0x40 && r <= 0x7e:
		t.param = append(t.param, r)
		t.handleCsi()
		t.param = nil
		t.state = stateGROUND

	case r < 0x20:
		// A C0 control inside a CSI is run right away, the CSI goes on
		t.processGround(r)

	case r == 0x7f:
		// DEL is ignored

	default:
		// Not part of a CSI, it is dropped
		t.param = nil
		t.state = stateGROUND
	}
}
