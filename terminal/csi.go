package terminal

import (
	"strconv"
	"strings"
)

func (t *Terminal) processCsi(r rune) {
	t.controlSequence += string(r)
	if (r < '0' || r > '9') && r != ';' && r != ':' && r != '=' && r != '?' && r != '>' {
		t.handleCsi()
		t.state = stateGROUND
	}
}

func (t *Terminal) handleCsi() {
	runes := []rune(t.controlSequence)
	t.controlSequence = ""

	if len(runes) == 0 {
		return
	}

	final := runes[len(runes)-1]
	message := string(runes[0 : len(runes)-1])

	// Private sequences such as "?25l" are not cursor or erase commands
	if message != "" && (message[0] == '?' || message[0] == '>' || message[0] == '=') {
		return
	}

	params := parseParams(message)

	switch final {
	case 'm':
		t.handleColorMode(message)
	case '@':
	//escapeInsertChars,
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
	//escapeInsertLines,
	case 'l':
	//escapePrivateModeOff,
	case 'J':
		t.eraseInScreen(param(params, 0, 0))
	case 'K':
		t.eraseInLine(param(params, 0, 0))
	case 'P':
	//escapeDeleteChars,
	case 'r':
	//escapeSetScrollArea,
	case 's':
		t.saveCursor()
	case 'S':
		for i := 0; i < param(params, 0, 1); i++ {
			t.screen.ScrollUp()
		}
	case 'u':
		t.restoreCursor()
	case 'i':
	//escapePrinterMode,
	case 'c':
	//escapeDeviceAttribute,
	case 'X':
		n := param(params, 0, 1)
		t.eraseCells(t.cursorY, t.cursorX, t.cursorX+n)
	case 'M':
	//escapeDeleteLines,
	case 'T':
	//escapeScrollDown,
	case 'b':
	//escapeRepeatChar,
	case 't':
		//escapeWindowOps,
	}
}

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

// param returns the i-th parameter, or def when it is missing or zero for counts
func param(params []int, i, def int) int {
	if i >= len(params) || params[i] < 0 {
		return def
	}

	if def > 0 && params[i] == 0 {
		return def
	}

	return params[i]
}

// eraseInLine: 0 cursor to end, 1 start to cursor, 2 whole line
func (t *Terminal) eraseInLine(mode int) {
	switch mode {
	case 0:
		t.eraseCells(t.cursorY, t.cursorX, t.col)
	case 1:
		t.eraseCells(t.cursorY, 0, t.cursorX+1)
	case 2:
		t.eraseCells(t.cursorY, 0, t.col)
	}
}

// eraseInScreen: 0 cursor to end, 1 start to cursor, 2/3 whole screen
func (t *Terminal) eraseInScreen(mode int) {
	switch mode {
	case 0:
		t.eraseCells(t.cursorY, t.cursorX, t.col)
		for row := t.cursorY + 1; row < t.row; row++ {
			t.eraseCells(row, 0, t.col)
		}
	case 1:
		for row := 0; row < t.cursorY; row++ {
			t.eraseCells(row, 0, t.col)
		}
		t.eraseCells(t.cursorY, 0, t.cursorX+1)
	case 2, 3:
		for row := 0; row < t.row; row++ {
			t.eraseCells(row, 0, t.col)
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
