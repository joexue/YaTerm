package terminal

import (
	"encoding/base64"
	"strings"
	"unicode"
)

// An OSC that never ends would grow without limit, the rest is dropped.
// Large enough for OSC 52 clipboard text.
const oscMaxLen = 1 << 20

// cleanTitle drops control characters a title must not have
func cleanTitle(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
}

// setClipboard handles "52;<targets>;<base64>". Reading the clipboard with
// "?" is not allowed, any program in the terminal could read it then.
func (t *Terminal) setClipboard(arg string) {
	_, data, ok := strings.Cut(arg, ";")
	if !ok || data == "?" || t.OnClipboard == nil {
		return
	}

	text, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return
	}
	t.OnClipboard(string(text))
}

// handleOsc runs a complete OSC, st is how it ended, a reply ends the same way
// ref: windows terminal code src/terminal/parser/OutputStateMachineEngine.hpp
func (t *Terminal) handleOsc() {
	data := string(t.param)

	code, arg, _ := strings.Cut(data, ";")
	switch code {
	// Window title, shown in the tab
	case "0", "2":
		t.setTitle(cleanTitle(arg))

	// Icon name, there are no icons
	case "1":

	// Set Color
	case "4":

	// 7 working directory
	case "7":
		t.setCwd(arg)

	// 8 hyperlinks
	case "8":

	// Set the clipboard: "52;c;<base64>"
	case "52":
		t.setClipboard(arg)

	default:
	}
}

// processOsc collects an OSC ("ESC ] PS ; PT BELL/ST")
// The 7 bit ST "ESC \" ends it too, see the '\\' case of ProcessRune.
// A '\' alone is text, e.g. ConPTY sends a title like "C:\WINDOWS\...".
func (t *Terminal) processOsc(r rune) {
	switch r {
	case asciiBell, asciiNull:
		t.handleOsc()
		t.param = nil
		t.state = stateGROUND

	case StringTerminator:
		t.handleOsc()
		t.param = nil
		t.state = stateGROUND

	default:
		if len(t.param) < oscMaxLen {
			t.param = append(t.param, r)
		}
	}
}
