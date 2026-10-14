package terminal

import (
	"encoding/hex"
	"fmt"
	"strings"
)

// A DCS can carry a lot (e.g. sixel images, which are not supported), more
// than this is dropped, the sequence is still read to its end
const dcsMaxLen = 1 << 16

const (
	asciiCancel     = 0x18 // CAN
	asciiSubstitute = 0x1a // SUB
)

// reply sends an answer to the program. Not while holding the state lock: if
// the program is busy writing output, writing to it could block until that
// output is read, which needs the lock.
func (t *Terminal) reply(s string) {
	if s == "" {
		return
	}
	go func() { _, _ = t.Write([]byte(s)) }()
}

// sgrString is the SGR parameters that set the current colors and attributes
func (t *Terminal) sgrString() string {
	params := []string{"0"}
	flags := []struct {
		on   bool
		code string
	}{
		{t.style.Bold, "1"}, {t.style.Faint, "2"}, {t.style.Italic, "3"},
		{t.style.Underline, "4"}, {t.style.Blinking, "5"}, {t.style.Inverse, "7"},
		{t.style.Hidden, "8"}, {t.style.Strikethrough, "9"},
	}
	for _, f := range flags {
		if f.on {
			params = append(params, f.code)
		}
	}

	// Colors are kept as RGB, so they are given back as RGB
	if t.fg != nil {
		r, g, b, _ := t.fg.RGBA()
		params = append(params, fmt.Sprintf("38;2;%d;%d;%d", r>>8, g>>8, b>>8))
	}
	if t.bg != nil {
		r, g, b, _ := t.bg.RGBA()
		params = append(params, fmt.Sprintf("48;2;%d;%d;%d", r>>8, g>>8, b>>8))
	}

	return strings.Join(params, ";")
}

// capabilities XTGETTCAP knows, an empty value is a boolean capability.
// The terminal is xterm-256color with true color (38;2 / 48;2).
var capabilities = map[string]string{
	"TN":      "xterm-256color",
	"name":    "xterm-256color",
	"Co":      "256",
	"colors":  "256",
	"RGB":     "",
	"Tc":      "",
	"setrgbf": "\x1b[38;2;%p1%d;%p2%d;%p3%dm",
	"setrgbb": "\x1b[48;2;%p1%d;%p2%d;%p3%dm",
}

// requestCapabilities answers XTGETTCAP, names are hex encoded and separated
// by ';'. Each one gets "DCS 1 + r <name>=<value> ST" (no "=<value>" for a
// boolean), or "DCS 0 + r <name> ST" when it is not known.
func (t *Terminal) requestCapabilities(names string) {
	var reply strings.Builder
	for _, hexName := range strings.Split(names, ";") {
		hexName = strings.ToUpper(hexName)

		name, err := hex.DecodeString(hexName)
		value, ok := capabilities[string(name)]
		if err != nil || !ok {
			fmt.Fprintf(&reply, "\x1bP0+r%s\x1b\\", hexName)
			continue
		}

		if value == "" {
			fmt.Fprintf(&reply, "\x1bP1+r%s\x1b\\", hexName)
		} else {
			fmt.Fprintf(&reply, "\x1bP1+r%s=%s\x1b\\", hexName, strings.ToUpper(hex.EncodeToString([]byte(value))))
		}
	}

	t.reply(reply.String())
}

// requestSetting answers DECRQSS with "DCS 1 $ r <setting> ST", or with
// "DCS 0 $ r ST" for a setting it does not know
func (t *Terminal) requestSetting(setting string) {
	var value string
	switch setting {
	// The current colors and attributes (SGR)
	case "m":
		value = t.sgrString() + "m"

	// The scroll region (DECSTBM)
	case "r":
		value = fmt.Sprintf("%d;%dr", t.scrollTop+1, t.regionBottom()+1)

	// The cursor style (DECSCUSR): 2 is a steady block, the only one drawn
	case " q":
		value = "2 q"

	default:
		t.reply("\x1bP0$r\x1b\\")
		return
	}

	t.reply("\x1bP1$r" + value + "\x1b\\")
}

// handleDcs answers the requests programs send to learn about the terminal,
// anything else (sixel, tmux passthrough, ...) is dropped without showing
func (t *Terminal) handleDcs() {
	data := string(t.param)

	switch {
	// DECRQSS: ask for a setting, "$q" and the setting's final characters
	case strings.HasPrefix(data, "$q"):
		t.requestSetting(data[2:])

	// XTGETTCAP: ask for terminfo capabilities, "+q" and hex names
	case strings.HasPrefix(data, "+q"):
		t.requestCapabilities(data[2:])
	}
}

// processDcs collects a DCS ("ESC P ... ESC \") until the ST. The 7 bit
// "ESC \" comes through the '\\' case of parseEscState, an ESC with anything
// else inside a DCS is kept as text (see parseEscState). CAN or SUB cancel it.
func (t *Terminal) processDcs(r rune) {
	switch r {
	case StringTerminator:
		t.handleDcs()
		t.param = nil
		t.state = stateGROUND

	case asciiCancel, asciiSubstitute:
		t.state = stateGROUND
		t.param = nil

	// Tmux control mode
	case 'p':
		if len(t.param) == 4 && string(t.param) == "1000" {
			t.param = nil
			t.state = stateDCSTmux
		} else {
			t.param = append(t.param, r)
		}

	default:
		if len(t.param) < dcsMaxLen {
			t.param = append(t.param, r)
		}
	}
}
