package terminal

import (
	"encoding/base64"
	"fmt"
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
func (t *Terminal) SetClipboard(arg string) {
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

// handleOsc runs a complete OSC, end is how it ended (BEL or ST), a reply
// ends the same way
// ref: windows terminal code src/terminal/parser/OutputStateMachineEngine.hpp
func (t *Terminal) handleOsc(end rune) {
	data := string(t.param)

	code, arg, _ := strings.Cut(data, ";")
	switch code {
	// Ask for the text (10), background (11) or cursor (12) color with "?",
	// e.g. vim and fish do, to know a light or a dark background. Setting
	// them is not supported.
	case "10", "11", "12":
		if arg == "?" {
			t.replyColor(code, end)
		}

	// Window title, shown in the tab
	case "0", "2":
		t.SetTitle(cleanTitle(arg))

	// Icon name, there are no icons
	case "1":

	// Set Color
	case "4":

	// 7 working directory
	case "7":
		t.SetCwd(arg)

	// 8 hyperlinks
	case "8":

	// Set the clipboard: "52;c;<base64>"
	case "52":
		t.SetClipboard(arg)

	default:
	}
}

// processOsc collects an OSC ("ESC ] PS ; PT BELL/ST")
// The 7 bit ST "ESC \" ends it too, see the '\\' case of ProcessRune.
// A '\' alone is text, e.g. ConPTY sends a title like "C:\WINDOWS\...".
func (t *Terminal) processOsc(r rune) {
	switch r {
	case asciiBell, asciiNull, StringTerminator:
		t.handleOsc(r)
		t.param = nil
		t.state = stateGROUND

	default:
		if len(t.param) < oscMaxLen {
			t.param = append(t.param, r)
		}
	}
}

// replyColor answers a color query with "OSC code ; rgb:rrrr/gggg/bbbb",
// ended as the query was
func (t *Terminal) replyColor(code string, end rune) {
	if t.OnQueryColor == nil {
		return
	}
	n := 10
	switch code {
	case "11":
		n = 11
	case "12":
		n = 12
	}
	c := t.OnQueryColor(n)
	if c == nil {
		return
	}
	r, g, b, _ := c.RGBA()
	st := "\x1b\\"
	if end == asciiBell {
		st = "\a"
	}
	t.reply(fmt.Sprintf("\x1b]%s;rgb:%04x/%04x/%04x%s", code, r, g, b, st))
}
