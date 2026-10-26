package ui

import (
	"bytes"
	"fmt"
	"image/color"
	"io"
	"strconv"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"

	"yaterm/assert"
	"yaterm/terminal"
)

/*
 * ===========================================================================
 *  tmux control mode (tmux -CC): every tmux window is a tab, every tmux pane
 *  a pane of it. The pane tmux runs in (the control pane) carries the
 *  commands to tmux and gets its replies and notifications, see
 *  https://github.com/tmux/tmux/wiki/Control-Mode
 * ===========================================================================
 */

// The session in use, only one at a time. Started on the reader goroutine of
// the control pane, used on the UI thread.
var (
	tmuxMu     sync.Mutex
	tmuxActive *tmuxSession
)

// tmuxSession is a tmux control mode session
type tmuxSession struct {
	sessionId int

	// the pane tmux -CC runs in
	control *Pane

	// The callbacks of the commands sent, in order, each gets the lines
	// between "%begin" and "%end" (or "%error"). mu also keeps the order of
	// the commands written the same as of the callbacks.
	cmdMutex        sync.Mutex
	pendingHandlers []func(lines []string, failed bool)

	// UI thread only, the lines are handled there (see TmuxLine)
	inResponse bool
	response   []string
	windows    map[int]*tmuxWindow
	panes      map[int]*Pane
}

func ActiveTmux() *tmuxSession {
	tmuxMu.Lock()
	defer tmuxMu.Unlock()
	return tmuxActive
}

type tmuxWindow struct {
	id       int
	title    string
	tab      *container.TabItem
	root     *Pane
	checksum string // of the layout shown, the same one again is ignored
}

// attach starts using the session: the control pane only takes q to detach,
// tmux gets the client size, every window becomes a tab
func (s *tmuxSession) attach(lines []string, failed bool) {
	if failed {
		s.control.term.Print(strings.Join(lines, "\n") + "\n")
		return
	}

	s.control.tmuxControl = s
	s.control.term.Print("Tmux control mode, press q to detach\n")
}

// detach ends the session, tmux goes on
func (s *tmuxSession) detach() {
	s.sendCommand("detach", nil)
}

// end is the end of the session: its tabs close, the control pane is a
// terminal again. It comes by %exit and by the end of the control mode in
// the terminal (TmuxEnd), the second time does nothing.
func (s *tmuxSession) end() {
	tmuxMu.Lock()
	defer tmuxMu.Unlock()
	for id, w := range s.windows {
		delete(s.windows, id)
		fyne.Do(func() {
			tabControl.Remove(w.tab)
		})
	}
	s.panes = map[int]*Pane{}

	s.control.tmuxControl = nil

	tmuxActive = nil
}

// SendClientSize tells tmux the size of the tab area in cells, the size of
// every tmux window
func (s *tmuxSession) SendClientSize(row, col int) {
	s.sendCommand(fmt.Sprintf("refresh-client -C %dx%d", col, row), nil)
}

// moveTmuxControl moves the session to the pane the control terminal moved
// to: a split moves it to the first pane, a close of the pane next to it to
// the parent. Else q would go to tmux as a key, and the commands to a pane
// with no terminal.
func MoveTmuxControl(from, to *Pane) {
	s := ActiveTmux()
	if s == nil || s.control != from || from == to {
		return
	}
	s.control = to
	to.tmuxControl, from.tmuxControl = from.tmuxControl, nil
}

// windowFromLine makes the tab of a window from a line of
// "#{window_id} #{window_active} #{window_layout} #{window_name}"
func (s *tmuxSession) windowFromLine(line string) (*container.TabItem, bool, int) {
	f := strings.SplitN(line, " ", 4)
	if len(f) < 3 {
		return nil, false, 0
	}
	id, ok := tmuxID(f[0], '@')
	if !ok {
		return nil, false, 0
	}
	name := ""
	if len(f) == 4 {
		name = f[3]
	}
	return s.createWindow(id, name, f[2]), f[1] == "1", id
}

// WindowAdded asks for the layout of a new window, tmux does not send it
func (s *tmuxSession) WindowAdded(id int) {
	if s.windows[id] != nil {
		return
	}
	s.sendCommand(fmt.Sprintf("display-message -p -t @%d '#{window_id} #{window_active} #{window_layout} #{window_name}'", id),
		func(lines []string, failed bool) {
			if !failed && len(lines) > 0 {
				if tab, _, _ := s.windowFromLine(lines[0]); tab != nil {
					s.selectTabItem(tab)
				}
			}
		})
}

// createWindow makes the tab of a window, nil if it is there or the layout
// can not be read
func (s *tmuxSession) createWindow(id int, name, layout string) *container.TabItem {
	if s.windows[id] != nil {
		return nil
	}
	node := s.parseLayout(layout)
	if node == nil {
		return nil
	}
	root := s.buildWindow(id, node)

	w := &tmuxWindow{
		id:       id,
		root:     root,
		title:    name,
		tab:      container.NewTabItemWithIcon(fmt.Sprintf("Tmux %d:%d", s.sessionId, id), assert.TmuxIconRes, root),
		checksum: layoutChecksum(layout),
	}
	s.windows[id] = w
	fyne.Do(func() {
		tabControl.Append(w.tab)
	})
	return w.tab
}

// LayoutChanged shows the new layout of a window. The same one again (by its
// checksum) is ignored. New sizes only, e.g. a pane resized in tmux or a new
// client size, go to the panes there are. Else, e.g. after a split or a pane
// that ended, the panes are made again, the terminals stay.
func (s *tmuxSession) LayoutChanged(id int, layout string) {
	w := s.windows[id]
	if w == nil {
		return // not shown yet, windowAdded makes it
	}

	checksum := layoutChecksum(layout)
	if checksum != "" && checksum == w.checksum {
		return
	}
	node := s.parseLayout(layout)
	if node == nil {
		return
	}
	w.checksum = checksum

	if s.resizeNode(w.root, node) {
		return
	}

	// The tree is made on the UI thread, the panes kept from the tree before
	// are in use there. The new root is the tab's content, the old one may be
	// a pane in the new tree, so it can not take the new one.
	fyne.Do(func() {
		focused := -1
		if f := w.root.lastFocus; f != nil {
			focused = f.extId
		}

		root := s.buildWindow(id, node)
		if tp := s.panes[focused]; tp != nil && tp.root == root {
			root.lastFocus = tp
		}

		w.root = root
		w.tab.Content = root
		tabControl.Refresh()

		if root.inShownTab() {
			s.DiscovePane(id)
		}
	})
}

// parseLayout reads a layout, nil when it can not
func (s *tmuxSession) parseLayout(layout string) *tmuxLayout {
	node, err := parseTmuxLayout(layout)
	if err != nil {
		fyne.LogError("tmux: can not read the layout "+layout, err)
		return nil
	}
	return node
}

// buildWindow makes the pane tree of a window from its layout, the panes not
// in it any more are gone
func (s *tmuxSession) buildWindow(window int, node *tmuxLayout) *Pane {
	used := map[int]bool{}
	root := s.buildNode(node, nil, window, used)
	for id, tp := range s.panes {
		if tp.tabId == window && !used[id] {
			delete(s.panes, id)
		}
	}

	root.lastFocus = root.Leaves()[0]
	return root
}

// buildNode makes the pane of a layout node. tmux splits a cell in any
// number of parts, a pane in two, so more parts are split one by one.
func (s *tmuxSession) buildNode(n *tmuxLayout, parent *Pane, window int, used map[int]bool) *Pane {
	if n.kind == 0 {
		return s.leaf(n, parent, window, used)
	}
	if len(n.kids) == 1 {
		return s.buildNode(n.kids[0], parent, window, used)
	}

	direction, first, rest, ratio := n.split()
	p := NewSplitPane(parent, direction, window, ratio)
	p.first = s.buildNode(first, p, window, used)
	p.second = s.buildNode(rest, p, window, used)
	return p
}

// resizeNode gives the panes of a tree the sizes of a layout of the same
// shape: its ratios to the splits, its sizes to the terminals. False when the
// shape is another, e.g. a pane was added, then the tree is made again.
func (s *tmuxSession) resizeNode(p *Pane, n *tmuxLayout) bool {
	if n.kind == 0 {
		if p.split != splitNone || !p.isTmux || p.extId != n.pane {
			return false
		}
		p.term.Resize(n.h, n.w)
		return true
	}
	if len(n.kids) == 1 {
		return s.resizeNode(p, n.kids[0])
	}

	direction, first, rest, ratio := n.split()
	if p.split != direction || !s.resizeNode(p.first, first) || !s.resizeNode(p.second, rest) {
		return false
	}
	if p.ratio != ratio {
		fyne.Do(func() {
			p.SetRatio(ratio)
		})
	}
	return true
}

// leaf makes the pane of a tmux pane, with its terminal at the size tmux
// gives it
func (s *tmuxSession) leaf(n *tmuxLayout, parent *Pane, window int, used map[int]bool) *Pane {
	tp := s.panes[n.pane]
	created := tp == nil
	if created {
		screen := NewScreen()
		term := terminal.NewWithPty(screen, &tmuxPty{session: s, pane: n.pane})
		if th := currentTheme(); th != nil {
			term.SetColors(th.Scheme().ANSI, queryThemeColor())
			screen.SetCursorShape(th.CursorShape())
		}

		tp = NewPane(parent, term, window, false)
		s.panes[n.pane] = tp
	}

	//tp.tabId = window
	used[n.pane] = true

	// A pane kept from the tree before is in the new one now
	tp.parent = parent
	if parent == nil {
		tp.root = tp
	} else {
		tp.root = parent.root
	}

	tp.isTmux = true
	tp.extId = n.pane
	tp.term.Resize(n.h, n.w)

	if created {
		s.syncPane(tp)
	}

	return tp
}

// syncPane fills a new pane with what tmux shows in it, with its history,
// and puts the cursor where it is
func (s *tmuxSession) syncPane(tp *Pane) {
	var captured []string
	s.sendCommand(fmt.Sprintf("capture-pane -p -e -C -t %%%d -S -%d", tp.extId, HistoryLines), func(lines []string, failed bool) {
		if !failed {
			captured = lines
		}
	})
	s.sendCommand(fmt.Sprintf("display-message -p -t %%%d '#{cursor_x} #{cursor_y}'", tp.extId), func(lines []string, failed bool) {
		if failed || captured == nil {
			return
		}
		x, y := 0, 0
		if len(lines) > 0 {
			fmt.Sscanf(lines[0], "%d %d", &x, &y)
		}

		var b bytes.Buffer
		b.WriteString("\x1b[H\x1b[2J")
		for i, l := range captured {
			if i > 0 {
				b.WriteString("\r\n")
			}
			b.Write(decodeTmux(l))
		}
		fmt.Fprintf(&b, "\x1b[%d;%dH", y+1, x+1)
		tp.term.ProcessOutput(b.Bytes(), b.Len())
	})
}

// output is what a program in a tmux pane wrote
func (s *tmuxSession) Output(pane int, data []byte) {
	if tp := s.panes[pane]; tp != nil && len(data) > 0 {
		tp.term.ProcessOutput(data, len(data))
	}
}

// CloseWindow removes the tab of a window that is gone in tmux
func (s *tmuxSession) CloseWindow(id int) {
	w := s.windows[id]
	if w == nil {
		return
	}
	delete(s.windows, id)
	for pid, tp := range s.panes {
		if tp.tabId == id {
			delete(s.panes, pid)
		}
	}
	fyne.Do(func() {
		tabControl.Remove(w.tab)
	})
}

// TabClosed is called when the user closed a tab, the window ends in tmux
func (s *tmuxSession) TabClosed(tab *container.TabItem) {
	for id, w := range s.windows {
		if w.tab == tab {
			delete(s.windows, id)
			s.sendCommand(fmt.Sprintf("kill-window -t @%d", id), nil)
			return
		}
	}
}

// tabFocus is called when the user selected a tab, tmux gets the window
func (s *tmuxSession) tabFocus(tab *container.TabItem) {
	for id, w := range s.windows {
		if w.tab == tab {
			ShowTitle(w.title)
			s.sendCommand(fmt.Sprintf("select-window -t @%d", id), nil)
			return
		}
	}
}

// OwnsTab tells if a tab is a tmux window
func (s *tmuxSession) OwnsTab(tab *container.TabItem) bool {
	for _, w := range s.windows {
		if w.tab == tab {
			return true
		}
	}
	return false
}

// PaneFocused tells tmux the pane the user works in
func (s *tmuxSession) PaneFocused(paneId int) {
	s.sendCommand(fmt.Sprintf("select-pane -t %%%d", paneId), nil)
}

func (s *tmuxSession) selectTabItem(tab *container.TabItem) {
	fyne.Do(func() {
		for i, t := range tabControl.Items {
			if t == tab && tabControl.SelectedIndex != i {
				tabControl.Select(i)
			}
		}
	})
}

func (s *tmuxSession) SetRatio(p *Pane) {
	if p.split == splitNone {
		return
	}
	leaf := p.first
	for leaf.split != splitNone {
		leaf = leaf.first
	}
	if !leaf.isTmux {
		return
	}

	if p.split == splitRight {
		s.sendCommand(fmt.Sprintf("resize-pane -t %%%d -x %d", leaf.extId, leaf.col), nil)
	} else {
		s.sendCommand(fmt.Sprintf("resize-pane -t %%%d -y %d", leaf.extId, leaf.row), nil)
	}
}

// split splits a tmux pane, the layout change makes the new pane
func (s *tmuxSession) Split(paneId, direction int) {
	flag := "-h"
	if direction == splitDown {
		flag = "-v"
	}
	s.sendCommand(fmt.Sprintf("split-window %s -t %%%d", flag, paneId), nil)
}

func (s *tmuxSession) KillPane(paneId int) {
	s.sendCommand(fmt.Sprintf("kill-pane -t %%%d", paneId), nil)
}

func (s *tmuxSession) killWindow(window int) {
	s.sendCommand(fmt.Sprintf("kill-window -t @%d", window), nil)
}

func (s *tmuxSession) NewWindow() {
	s.sendCommand("new-window", nil)
}

// sendKeys types into a tmux pane: text as it is (-l), other bytes like ESC
// sequences as hex (-H), each command a line of its own
func (s *tmuxSession) sendKeys(pane int, data []byte) {
	for len(data) > 0 {
		n := 0
		for n < len(data) && data[n] >= 0x20 && data[n] != 0x7f {
			n++
		}
		if n > 0 {
			s.sendCommand(fmt.Sprintf("send-keys -t %%%d -l %s", pane, tmuxQuote(string(data[:n]))), nil)
			data = data[n:]
			continue
		}

		for n < len(data) && (data[n] < 0x20 || data[n] == 0x7f) {
			n++
		}
		var hex strings.Builder
		for _, c := range data[:n] {
			fmt.Fprintf(&hex, " %02x", c)
		}
		s.sendCommand(fmt.Sprintf("send-keys -t %%%d -H%s", pane, hex.String()), nil)
		data = data[n:]
	}
}

func (s *tmuxSession) DiscoveWindow() {
	s.sendCommand(fmt.Sprintf("list-windows -F '#{window_id} #{window_active} #{window_layout} #{window_name}' -t $%d", s.sessionId), func(lines []string, failed bool) {
		if failed {
			// Output the error message
			s.control.term.Print(strings.Join(lines, "\n") + "\n")
			return
		}
		var activeTab *container.TabItem
		var activeTabId int
		for _, l := range lines {
			if tab, active, id := s.windowFromLine(l); tab != nil && active {
				activeTab = tab
				activeTabId = id
			}
		}

		if activeTab != nil {
			s.selectTabItem(activeTab)
			s.DiscovePane(activeTabId)
		}
	})
}

func (s *tmuxSession) DiscovePane(window int) {
	s.sendCommand(fmt.Sprintf("list-panes -F '#{pane_id} #{pane_active}' -t @%d", window), func(lines []string, failed bool) {
		if failed {
			// Output the error message
			s.control.term.Print(strings.Join(lines, "\n") + "\n")
			return
		}

		var activePane *Pane
		for _, l := range lines {
			paneStr, active, _ := strings.Cut(l, " ")
			if active == "1" {
				if paneId, ok := tmuxID(paneStr, '%'); ok {
					activePane, _ = s.panes[paneId]
				}

				break
			}
		}

		if activePane != nil {
			fyne.Do(func() {
				activePane.Focus()
			})
		}
	})
}

// notification handles a line that is not in a reply
func (s *tmuxSession) notification(line string) {
	name, rest, _ := strings.Cut(line, " ")
	switch name {
	case "%session-changed":
		session, _, _ := strings.Cut(rest, " ")
		if id, ok := tmuxID(session, '$'); ok {
			s.sessionId = id
			s.SendClientSize(s.control.root.row, s.control.root.col)
			s.DiscoveWindow()
		}

	case "%output":
		pane, data, _ := strings.Cut(rest, " ")
		if id, ok := tmuxID(pane, '%'); ok {
			s.Output(id, decodeTmux(data))
		}

	// The window, its layout, then its visible layout and flags
	case "%layout-change":
		f := strings.Fields(rest)
		if len(f) >= 2 {
			if id, ok := tmuxID(f[0], '@'); ok {
				s.LayoutChanged(id, f[1])
			}
		}

	case "%window-add":
		if id, ok := tmuxID(rest, '@'); ok {
			s.WindowAdded(id)
		}

	case "%window-close", "%unlinked-window-close":
		if id, ok := tmuxID(rest, '@'); ok {
			s.CloseWindow(id)
		}

	case "%window-renamed":
		w, name, _ := strings.Cut(rest, " ")
		if id, ok := tmuxID(w, '@'); ok {
			if win := s.windows[id]; win != nil {
				win.title = name
				fyne.Do(func() {
					ShowTitle(name)
				})
			}
		}

	// Notifications not caused by YaTerm's own commands are ignored, e.g. a
	// pane or window another tmux client selected (%window-pane-changed,
	// %session-window-changed): YaTerm keeps its own focus and tab

	case "%exit":
		s.end()
	}
}

// reply hands a reply to its command, the first one is to no command
func (s *tmuxSession) handleCommandResponse(lines []string, failed bool) {
	s.cmdMutex.Lock()
	var done func([]string, bool)
	if len(s.pendingHandlers) > 0 {
		done = s.pendingHandlers[0]
		s.pendingHandlers = s.pendingHandlers[1:]
	}
	s.cmdMutex.Unlock()

	if done != nil {
		done(lines, failed)
	}
}

// send writes a command to tmux, handler gets its reply, handler may be nil
func (s *tmuxSession) sendCommand(cmd string, handler func(lines []string, failed bool)) {
	//s.control.term.Print("DEBUG COMMAND: " + cmd + "\n")
	s.cmdMutex.Lock()
	defer s.cmdMutex.Unlock()
	s.pendingHandlers = append(s.pendingHandlers, handler)
	// For the first imply command
	if len(cmd) > 0 {
		_, _ = s.control.term.Write([]byte(cmd + "\n"))
	}
}

// line handles a line from tmux: a reply to a command or a notification
func (s *tmuxSession) line(line string) {
	if s.inResponse {
		if strings.HasPrefix(line, "%end ") || strings.HasPrefix(line, "%error ") {
			lines, failed := s.response, strings.HasPrefix(line, "%error ")
			s.handleCommandResponse(lines, failed)
			s.inResponse, s.response = false, nil
			return
		}
		s.response = append(s.response, line)
		return
	}

	if strings.HasPrefix(line, "%begin ") {
		s.inResponse = true
		return
	}

	s.notification(line)
}

// TmuxLine and TmuxEnd come on the reader goroutine with the terminal's
// lock held. The session works on the UI thread: its windows and panes are
// tabs and widgets, and the UI uses its maps, which must not be used by two
// goroutines at once.
func (s *tmuxSession) TmuxLine(line string) {
	//s.control.term.Print("DEBUG: " + line)
	//s.control.term.Print("\n")
	s.line(line)
}

// TmuxEnd: the control mode ended in the terminal, by its ST, or by the end
// of the program, e.g. tmux went away without %exit
func (s *tmuxSession) TmuxEnd() {
	s.end()
}

// tmuxRefuser detaches a second session, YaTerm runs one at a time
type tmuxRefuser struct {
	term *terminal.Terminal
}

func (r *tmuxRefuser) TmuxLine(string) {
	// Do nothing, just consume the lines we don't want
}

func (r *tmuxRefuser) TmuxEnd() {
}

// tmuxPty is the pty of a tmux pane: what is typed goes to tmux as keys, the
// output comes from the session, the size from the tmux layout
type tmuxPty struct {
	session *tmuxSession
	pane    int
}

func (p *tmuxPty) Write(b []byte) (int, error) {
	// Replies of the terminal come from other goroutines, the keys go to
	// tmux in order from the UI thread
	data := append([]byte(nil), b...)
	fyne.Do(func() { p.session.sendKeys(p.pane, data) })
	return len(b), nil
}

// Resize does not tell tmux: the terminal of a tmux pane is only resized to
// the size a layout from tmux gives it, which tmux has. A size the user gives
// a pane by a divider goes to tmux by dividerMoved.
func (p *tmuxPty) Resize(int, int) {}

func (p *tmuxPty) RunCmd(string) error      { return nil }
func (p *tmuxPty) Read([]byte) (int, error) { return 0, io.EOF }
func (p *tmuxPty) Close()                   {}

// tmuxLayout is a cell of a tmux window layout: a pane, or cells side by
// side ('{') or one over the other ('[')
type tmuxLayout struct {
	w, h, x, y int
	pane       int
	kind       byte
	kids       []*tmuxLayout
}

// Split is how a cell of two or more cells is shown by panes, which split in
// two: the direction, the first cell, the rest as one cell, and the part of
// the first. The part is by the size in units the two need to show their
// cells, with the dividers and paddings of the panes.
func (n *tmuxLayout) split() (direction int, first, rest *tmuxLayout, ratio float32) {
	direction = splitRight
	if n.kind == '[' {
		direction = splitDown
	}

	first = n.kids[0]
	rest = &tmuxLayout{kind: n.kind, kids: n.kids[1:]}
	if len(rest.kids) == 1 {
		rest = rest.kids[0]
	}

	a, b := first.outer(direction), rest.outer(direction)
	return direction, first, rest, a / (a + b)
}

// outer is how many units the cell needs in a direction to show its cells:
// the cells, the paddings of the panes and the dividers between them
func (n *tmuxLayout) outer(direction int) float32 {
	cell := measureCell()
	side := n.kind == '{' && direction == splitRight || n.kind == '[' && direction == splitDown
	switch {
	case n.kind == 0 && direction == splitRight:
		return float32(n.w)*cell.Width + 2*panePadding
	case n.kind == 0:
		return float32(n.h)*cell.Height + 2*panePadding
	case side:
		sum := float32(0)
		for _, k := range n.kids {
			sum += k.outer(direction)
		}
		return sum + float32(len(n.kids)-1)*SplitDefaultThickness
	default:
		return n.kids[0].outer(direction)
	}
}

// parseTmuxLayout reads a layout like "a87d,100x30,0,0,0" or
// "6b8b,100x30,0,0{50x30,0,0,0,49x30,51,0,1}"
func parseTmuxLayout(s string) (*tmuxLayout, error) {
	// The checksum before the first ','
	if i := strings.IndexByte(s, ','); i == 4 {
		s = s[i+1:]
	}
	p := &layoutParser{s: s}
	n, err := p.cell()
	if err == nil && p.i != len(p.s) {
		err = fmt.Errorf("text after the layout at %d", p.i)
	}
	return n, err
}

type layoutParser struct {
	s string
	i int
}

func (p *layoutParser) num() (int, error) {
	start := p.i
	for p.i < len(p.s) && p.s[p.i] >= '0' && p.s[p.i] <= '9' {
		p.i++
	}
	if start == p.i {
		return 0, fmt.Errorf("a number is needed at %d", start)
	}
	return strconv.Atoi(p.s[start:p.i])
}

func (p *layoutParser) want(c byte) error {
	if p.i >= len(p.s) || p.s[p.i] != c {
		return fmt.Errorf("%q is needed at %d", c, p.i)
	}
	p.i++
	return nil
}

// cell reads "WxH,X,Y" then ",ID", "{cells}" or "[cells]"
func (p *layoutParser) cell() (*tmuxLayout, error) {
	n := &tmuxLayout{}
	var err error
	if n.w, err = p.num(); err != nil {
		return nil, err
	}
	if err = p.want('x'); err != nil {
		return nil, err
	}
	if n.h, err = p.num(); err != nil {
		return nil, err
	}
	if err = p.want(','); err != nil {
		return nil, err
	}
	if n.x, err = p.num(); err != nil {
		return nil, err
	}
	if err = p.want(','); err != nil {
		return nil, err
	}
	if n.y, err = p.num(); err != nil {
		return nil, err
	}

	if p.i >= len(p.s) {
		return nil, fmt.Errorf("the layout ends at %d", p.i)
	}
	switch c := p.s[p.i]; c {
	case ',':
		p.i++
		if n.pane, err = p.num(); err != nil {
			return nil, err
		}
		return n, nil
	case '{', '[':
		end := byte('}')
		if c == '[' {
			end = ']'
		}
		n.kind = c
		p.i++
		for {
			kid, err := p.cell()
			if err != nil {
				return nil, err
			}
			n.kids = append(n.kids, kid)
			if p.i < len(p.s) && p.s[p.i] == ',' {
				p.i++
				continue
			}
			if err := p.want(end); err != nil {
				return nil, err
			}
			return n, nil
		}
	default:
		return nil, fmt.Errorf("unexpected %q at %d", c, p.i)
	}
}

// newSplitPane makes a pane split in two, e.g. for a tmux layout, the caller
// sets its first and second pane
func NewSplitPane(parent *Pane, direction int, window int, ratio float32) *Pane {
	b := canvas.NewRectangle(color.Transparent)
	b.StrokeWidth = paneBorderWidth
	b.StrokeColor = color.Transparent

	p := &Pane{
		id:       paneId,
		extId:    -1,
		tabId:    window,
		border:   b,
		split:    direction,
		ratio:    ratio,
		parent:   parent,
		cellSize: measureCell(),
		isTmux:   true,
	}
	if parent == nil {
		p.root = p
	} else {
		p.root = parent.root
	}
	p.divider = NewDivider(p)
	p.ExtendBaseWidget(p)

	paneId++
	return p
}

// StartTmux is the control pane's OnTmuxControl: a session, or one that
// detaches right away when another is in use
func StartTmux(control *Pane) terminal.TmuxHandler {
	tmuxMu.Lock()
	defer tmuxMu.Unlock()
	if tmuxActive != nil {
		// Print takes the lock this (OnTmuxControl) holds
		control.term.Print("YaTerm runs one tmux control mode session at a time\n")
		control.term.Write([]byte("detach\n"))
		return &tmuxRefuser{term: control.term}
	}

	tmuxActive = &tmuxSession{
		control: control,
		windows: map[int]*tmuxWindow{},
		panes:   map[int]*Pane{},
	}

	// the fist response from the outside command "attach", but we need to handle it
	tmuxActive.sendCommand("", tmuxActive.attach)
	return tmuxActive
}
