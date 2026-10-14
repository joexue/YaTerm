package terminal

import (
	"encoding/base64"
	"fmt"
	"strconv"
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

// queryColors answers the "?" in "10;?;?", the n-th "?" asks for color
// code+n. Colors set this way ("10;#ff0000") are ignored.
func (t *Terminal) queryColors(code, arg, st string) {
	if t.OnQueryColor == nil {
		return
	}

	n, _ := strconv.Atoi(code)
	var reply strings.Builder
	for _, a := range strings.Split(arg, ";") {
		if a == "?" && n <= 12 {
			if c := t.OnQueryColor(n); c != nil {
				r, g, b, _ := c.RGBA()
				fmt.Fprintf(&reply, "\x1b]%d;rgb:%04x/%04x/%04x%s", n, r, g, b, st)
			}
		}
		n++
	}

	if reply.Len() > 0 {
		// Not while holding the state lock: if the program is busy writing
		// output, writing to it could block until that output is read, which
		// needs the lock
		go func() { _, _ = t.Write([]byte(reply.String())) }()
	}
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
func (t *Terminal) handleOsc() {
	data := string(t.param)

	code, arg, _ := strings.Cut(data, ";")
	switch code {
	// Window title, shown in the tab
	case "0", "2":
		t.setTitle(cleanTitle(arg))

	case "7":
		t.setCwd(arg)

	// Icon name, there are no icons
	case "1":
		/*
			// Query the text, background and cursor color, "10;?" or "10;?;?" for
			// 10 and 11. vim and others ask to know if the background is dark.
			case "10", "11", "12":
				t.queryColors(code, arg, st)
		*/
	// Set the clipboard: "52;c;<base64>"
	case "52":
		t.setClipboard(arg)

	// 7 working directory, 8 hyperlinks, 133 and 633 shell integration marks,
	// 4 and 104 palette, 9 notifications: not supported, ignored
	default:
	}
}

// processOsc collects an OSC ("ESC ] code ; text") until BEL or the C1 ST.
// The 7 bit ST "ESC \" ends it too, see the '\\' case of parseEscState.
// A '\' alone is text, e.g. ConPTY sends a title like "C:\WINDOWS\...".
func (t *Terminal) processOsc(r rune) {
	switch r {
	case asciiBell, asciiNull:
		t.handleOsc()
		t.param = nil
		t.state = stateGROUND

	case asciiEscape: // ESC ST
		t.state = stateOSCTermination

	case '\\':
		if t.state == stateOSCTermination {
			t.handleOsc()
			t.param = nil
			t.state = stateGROUND
		} else {
			t.param = append(t.param, r)
		}

	default:
		if len(t.param) < oscMaxLen {
			t.param = append(t.param, r)
		}
	}
}
