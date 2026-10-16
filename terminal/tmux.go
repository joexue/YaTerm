package terminal

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
	t.tmuxLine.Reset()
	t.param = []rune{}
	t.state = stateDCSTmux
	return true
}

// processTmux collects the lines of the control mode. An ESC ends it, tmux
// escapes all of them in its text, so it is the start of the final ST.
func (t *Terminal) processTmux(r rune) {
	switch r {
	case asciiEscape, 0x9c: // ESC of "ESC \", or the C1 ST
		t.endTmux()
		/*
			if r == asciiEscape {
				// The "\" that follows ends the escape sequence
				t.escape = true
			}
		*/
	case '\n':
		line := t.tmuxLine.String()
		t.tmuxLine.Reset()
		t.tmux.TmuxLine(line)
	case '\r':
	default:
		if t.tmuxLine.Len() < tmuxMaxLine {
			t.tmuxLine.WriteRune(r)
		}
	}
}

// endTmux ends the control mode, called with the state lock held
func (t *Terminal) endTmux() {
	h := t.tmux
	t.tmux = nil
	t.tmuxLine.Reset()
	t.state = stateGROUND
	if h != nil {
		h.TmuxEnd()
	}
}
