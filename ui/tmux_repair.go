package ui

import (
	"bytes"
	"fmt"
	"runtime"
	"sync"
	"time"
)

/*
 * ================================
 *  Repair of characters ConPTY broke (Windows): ConPTY decodes the output of
 *  the program in it (e.g. wsl.exe running tmux -CC) as UTF-8 text. tmux
 *  splits the output of a pane between its %output lines at any byte, also
 *  inside a character, then ConPTY turns both parts into "�" (U+FFFD), and
 *  the character can not be told back. Such a line starts or ends with "�",
 *  then the pane is drawn again from what tmux shows in it (capture-pane),
 *  once its output calmed down: whole lines, no character is split there.
 * ================================
 */

// repairDelay is how long a pane's output has to be quiet after a broken
// character before it is drawn again, so a burst (e.g. a full redraw) is
// repaired once
const repairDelay = 100 * time.Millisecond

var replacementChar = []byte("�")

// tmuxRepair is the repair state of a session
type tmuxRepair struct {
	mu      sync.Mutex
	damaged map[int]bool // tmux pane ids to draw again
	timer   *time.Timer
	stopped bool
}

// cutByConpty tells %output data of a pane may have a character ConPTY
// broke: only on Windows, at the start or the end of the line
func cutByConpty(data []byte) bool {
	return runtime.GOOS == "windows" &&
		(bytes.HasPrefix(data, replacementChar) || bytes.HasSuffix(data, replacementChar))
}

// damage marks a tmux pane to be drawn again, after repairDelay without
// another broken character
func (s *TmuxSession) damage(pane int) {
	r := &s.repair
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.stopped {
		return
	}
	if r.damaged == nil {
		r.damaged = map[int]bool{}
	}
	r.damaged[pane] = true
	if r.timer == nil {
		r.timer = time.AfterFunc(repairDelay, s.repairDamaged)
	} else {
		r.timer.Reset(repairDelay)
	}
}

// stopRepair stops the repairs when the session ends, a command sent after
// it would go to the shell of the control pane
func (s *TmuxSession) stopRepair() {
	r := &s.repair
	r.mu.Lock()
	defer r.mu.Unlock()

	r.stopped = true
	r.damaged = nil
	if r.timer != nil {
		r.timer.Stop()
	}
}

// repairDamaged draws the damaged panes again, on the timer's goroutine
func (s *TmuxSession) repairDamaged() {
	r := &s.repair
	r.mu.Lock()
	panes := r.damaged
	r.damaged = nil
	stopped := r.stopped
	r.mu.Unlock()

	if stopped || ActiveTmux() != s {
		return
	}
	for id := range panes {
		s.repairPane(id)
	}
}

// repairPane draws a pane again from what tmux shows in it, with the cursor
// where tmux has it. Each line goes to its row by a cursor move, so a scroll
// region the program set does not scroll them. The replies are handled on
// the control pane's goroutine, like %output.
func (s *TmuxSession) repairPane(id int) {
	var captured []string
	s.sendCommand(fmt.Sprintf("capture-pane -p -e -C -t %%%d", id), func(lines []string, failed bool) {
		if !failed {
			captured = lines
		}
	})
	s.sendCommand(fmt.Sprintf("display-message -p -t %%%d '#{cursor_x} #{cursor_y}'", id), func(lines []string, failed bool) {
		tp := s.panes[id]
		if failed || captured == nil || tp == nil || tp.term == nil {
			return
		}
		x, y := 0, 0
		if len(lines) > 0 {
			fmt.Sscanf(lines[0], "%d %d", &x, &y)
		}

		var b bytes.Buffer
		b.WriteString("\x1b[2J")
		for i, l := range captured {
			fmt.Fprintf(&b, "\x1b[%d;1H", i+1)
			b.Write(decodeTmux(l))
		}
		fmt.Fprintf(&b, "\x1b[%d;%dH", y+1, x+1)
		tp.term.ProcessOutput(b.Bytes(), b.Len())
	})
}
