package terminal

import (
	"encoding/hex"
	"fmt"
	"strings"
)

// A DCS can carry a lot (e.g. sixel images, which are not supported), more
// than this is dropped, the sequence is still read to its end
const dcsMaxLen = 1 << 16

/*
 * ================================
 *  tmux control mode (tmux -CC). tmux starts it with the DCS "ESC P 1000 p",
 *  then sends lines of text, notifications like "%output %1 ..." and the
 *  replies to commands between "%begin" and "%end", and ends it with ST.
 *  The terminal hands each line to a TmuxHandler, the UI does the rest.
 * ================================
 */

// TmuxHandler gets the lines of a tmux control mode session. Both are
// called on the reader goroutine with the state lock held, so they must not
// call back into the terminal, e.g. hand the work to the UI thread.
type TmuxHandler interface {
	TmuxLine(line string)
	TmuxEnd()
}

// A line longer than this is not tmux, the rest is dropped
const tmuxMaxLine = 1 << 20

// startTmux starts the control mode if the UI allows it, called when the DCS
// "1000p" is read. With no handler the DCS goes on as any other, its text is
// dropped.
func (t *Terminal) startTmux() bool {
	if t.OnTmuxControl == nil {
		return false
	}
	h := t.OnTmuxControl()
	if h == nil {
		return false
	}

	t.tmux = h
	return true
}

// endTmux ends the control mode, called with the state lock held: at the ST,
// or when the program ended inside it
func (t *Terminal) endTmux() {
	h := t.tmux
	if h != nil {
		h.TmuxEnd()
	}
	t.tmux = nil

	// The pty kept its size during the mode, see Resize. Not here: this holds
	// the lock, and is the reader of what ConPTY writes on a resize.
	if t.pty != nil && t.row > 0 && t.col > 0 {
		row, col := t.row, t.col
		go t.pty.Resize(row, col)
	}
}

// processDcsTmux collects the lines of the control mode. An ESC ends it, tmux
// escapes all of them in its text, so it is the start of the final ST.
func (t *Terminal) processDcsTmux(r rune) {
	switch r {
	case '\n':
		line := string(t.param)
		t.param = []rune{}
		t.tmux.TmuxLine(line)

	// Ignore
	case '\r':

	default:
		if len(t.param) < tmuxMaxLine {
			t.param = append(t.param, r)
		}
	}
}

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
		// Not by the state: the "ESC \" of tmux comes in stateDCSTermination
		if t.tmux != nil {
			t.endTmux()
		} else {
			t.handleDcs()
		}
		t.param = nil
		t.state = stateGROUND

	case asciiCancel, asciiSub:
		t.state = stateGROUND
		t.param = nil

	// Tmux control mode
	case 'p':
		if len(t.param) == 4 && string(t.param) == "1000" && t.startTmux() {
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
