package ui

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"yaterm/assert"
	"yaterm/terminal"
)

/*
 * ================================
 *  tmux control mode (tmux -CC): every tmux window is a tab, every tmux pane
 *  a pane of it. The pane tmux runs in (the control pane) carries the
 *  commands to tmux and gets its replies and notifications, see
 *  https://github.com/tmux/tmux/wiki/Control-Mode
 * ================================
 */

// The session in use, only one at a time. Started on the reader goroutine of
// the control pane, used on the UI thread.
var (
	tmuxMu     sync.Mutex
	tmuxActive *tmuxSession
)

func activeTmux() *tmuxSession {
	tmuxMu.Lock()
	defer tmuxMu.Unlock()
	return tmuxActive
}

// tmuxSession is a tmux control mode session
type tmuxSession struct {
	control *Pane // the pane tmux -CC runs in

	// The callbacks of the commands sent, in order, each gets the lines
	// between "%begin" and "%end" (or "%error"). mu also keeps the order of
	// the commands written the same as of the callbacks.
	mu      sync.Mutex
	pending []func(lines []string, failed bool)

	// Work for the UI thread from the reader goroutine, see post
	qmu   sync.Mutex
	queue []func()
	wake  chan struct{}

	// UI thread only
	attached  bool
	gotFirst  bool // tmux starts with a reply to no command
	inBlock   bool
	block     []string
	windows   map[int]*tmuxWindow
	panes     map[int]*tmuxPane
	removing  bool // a tab is removed by the session, not closed by the user
	active    int  // the pane tmux has active
	cols, rws int  // the client size sent to tmux
}

type tmuxWindow struct {
	id   int
	tab  *container.TabItem
	root *Pane
}

// tmuxPane is a tmux pane, its terminal and screen stay while the panes
// showing them are made again on each layout change
type tmuxPane struct {
	id, window int
	term       *terminal.Terminal
	screen     *Screen
	leaf       *Pane
}

// startTmux is the control pane's OnTmuxControl: a session, or one that
// detaches right away when another is in use
func startTmux(control *Pane) terminal.TmuxHandler {
	tmuxMu.Lock()
	defer tmuxMu.Unlock()
	if tmuxActive != nil {
		control.term.Print("YaTerm runs one tmux control mode session at a time\n")
		control.term.Write([]byte("detach\n"))
		return &tmuxRefuser{term: control.term}
	}

	tmuxActive = &tmuxSession{
		control: control,
		windows: map[int]*tmuxWindow{},
		panes:   map[int]*tmuxPane{},
		active:  -1,
		wake:    make(chan struct{}, 1),
	}
	go tmuxActive.work()
	return tmuxActive
}

// tmuxRefuser detaches a second session, YaTerm runs one at a time
type tmuxRefuser struct {
	term *terminal.Terminal
	done bool
}

func (r *tmuxRefuser) TmuxLine(string) {
	// Do nothing, just consume the lines we don't want
}

func (r *tmuxRefuser) TmuxEnd() {
}

// TmuxLine and TmuxEnd come on the reader goroutine with the terminal's
// lock held, the session works on the UI thread
func (s *tmuxSession) TmuxLine(line string) {
	s.post(func() { s.line(line) })
}

func (s *tmuxSession) TmuxEnd() {
	s.post(nil) // nil ends the work
}

// post queues work for the UI thread. The queue has no limit, so the reader
// never waits here while it holds the terminal's lock, which the UI thread
// may need (e.g. to print a message), and the order stays.
func (s *tmuxSession) post(f func()) {
	s.qmu.Lock()
	s.queue = append(s.queue, f)
	s.qmu.Unlock()
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

// work runs the queued work on the UI thread, all that is queued at once,
// so a lot of output does not wait for the UI thread line by line
func (s *tmuxSession) work() {
	for range s.wake {
		s.qmu.Lock()
		batch := s.queue
		s.queue = nil
		s.qmu.Unlock()

		ended := false
		fyne.DoAndWait(func() {
			for _, f := range batch {
				if f == nil {
					s.end()
					ended = true
					return
				}
				f()
			}
		})
		if ended {
			return
		}
	}
}

// send writes a command to tmux, done gets its reply, done may be nil
func (s *tmuxSession) send(cmd string, done func(lines []string, failed bool)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pending = append(s.pending, done)
	_, _ = s.control.term.Write([]byte(cmd + "\n"))
}

// line handles a line from tmux: a reply to a command or a notification
func (s *tmuxSession) line(line string) {
	if s.inBlock {
		if strings.HasPrefix(line, "%end ") || strings.HasPrefix(line, "%error ") {
			lines, failed := s.block, strings.HasPrefix(line, "%error ")
			s.inBlock, s.block = false, nil
			s.reply(lines, failed)
			return
		}
		s.block = append(s.block, line)
		return
	}

	if strings.HasPrefix(line, "%begin ") {
		s.inBlock = true
		return
	}

	s.notification(line)
}

// reply hands a reply to its command, the first one is to no command
func (s *tmuxSession) reply(lines []string, failed bool) {
	if !s.gotFirst {
		s.gotFirst = true
		s.attach()
		return
	}

	s.mu.Lock()
	var done func([]string, bool)
	if len(s.pending) > 0 {
		done = s.pending[0]
		s.pending = s.pending[1:]
	}
	s.mu.Unlock()

	if failed {
		s.control.term.Print("\n[tmux] " + strings.Join(lines, "\n") + "\n")
	}
	if done != nil {
		done(lines, failed)
	}
}

// notification handles a line that is not in a reply
func (s *tmuxSession) notification(line string) {
	name, rest, _ := strings.Cut(line, " ")
	switch name {
	case "%output":
		pane, data, _ := strings.Cut(rest, " ")
		if id, ok := tmuxID(pane, '%'); ok {
			s.output(id, decodeTmux(data))
		}

	case "%layout-change":
		f := strings.Fields(rest)
		if len(f) >= 2 {
			if id, ok := tmuxID(f[0], '@'); ok {
				s.layoutChanged(id, f[1])
			}
		}

	case "%window-add":
		if id, ok := tmuxID(rest, '@'); ok {
			s.windowAdded(id)
		}

	case "%window-close", "%unlinked-window-close":
		if id, ok := tmuxID(rest, '@'); ok {
			s.closeWindow(id)
		}

	case "%window-renamed":
		w, name, _ := strings.Cut(rest, " ")
		if id, ok := tmuxID(w, '@'); ok {
			if win := s.windows[id]; win != nil {
				win.tab.Text = name
				tabControl.Refresh()
			}
		}

	case "%window-pane-changed":
		f := strings.Fields(rest)
		if len(f) == 2 {
			if id, ok := tmuxID(f[1], '%'); ok {
				s.paneActivated(id)
			}
		}

	case "%session-window-changed":
		f := strings.Fields(rest)
		if len(f) == 2 {
			if id, ok := tmuxID(f[1], '@'); ok {
				s.selectTab(id)
			}
		}
	}
}

// tmuxID reads an id like "%3" or "@1"
func tmuxID(s string, prefix byte) (int, bool) {
	if len(s) < 2 || s[0] != prefix {
		return 0, false
	}
	n, err := strconv.Atoi(s[1:])
	return n, err == nil
}

// attach starts using the session: the control pane only takes q to detach,
// tmux gets the client size, every window becomes a tab
func (s *tmuxSession) attach() {
	s.attached = true
	s.control.tmuxControl = s
	s.control.term.Print("Tmux control mode, press q to detach\n")

	s.sendClientSize(s.control.root.Size())
	s.send("list-windows -F '#{window_id} #{window_active} #{window_layout} #{window_name}'", func(lines []string, failed bool) {
		if failed {
			return
		}
		var activeTab *container.TabItem
		for _, l := range lines {
			if tab, active := s.windowFromLine(l); tab != nil && active {
				activeTab = tab
			}
		}
		if activeTab != nil {
			s.selectTabItem(activeTab)
		}
	})
}

// detach ends the session, tmux goes on
func (s *tmuxSession) detach() {
	s.send("detach", nil)
}

// end is the end of the session: its tabs close, the control pane is a
// terminal again
func (s *tmuxSession) end() {
	s.removing = true
	for id, w := range s.windows {
		tabControl.Remove(w.tab)
		delete(s.windows, id)
	}
	s.removing = false
	s.panes = map[int]*tmuxPane{}

	s.control.tmuxControl = nil
	if s.attached {
		s.control.term.Print("tmux control mode ended")
		s.control.term.Write([]byte{'\n'})
	}

	tmuxMu.Lock()
	if tmuxActive == s {
		tmuxActive = nil
	}
	tmuxMu.Unlock()
}

// sendClientSize tells tmux the size of the tab area in cells, the size of
// every tmux window
func (s *tmuxSession) sendClientSize(size fyne.Size) {
	cell := measureCell()
	cols := int((size.Width - 2*panePadding) / cell.Width)
	rows := int((size.Height - 2*panePadding) / cell.Height)
	if cols < 2 || rows < 2 || (cols == s.cols && rows == s.rws) {
		return
	}
	s.cols, s.rws = cols, rows
	s.send(fmt.Sprintf("refresh-client -C %dx%d", cols, rows), nil)
}

// windowFromLine makes the tab of a window from a line of
// "#{window_id} #{window_active} #{window_layout} #{window_name}"
func (s *tmuxSession) windowFromLine(line string) (*container.TabItem, bool) {
	f := strings.SplitN(line, " ", 4)
	if len(f) < 3 {
		return nil, false
	}
	id, ok := tmuxID(f[0], '@')
	if !ok {
		return nil, false
	}
	name := ""
	if len(f) == 4 {
		name = f[3]
	}
	return s.createWindow(id, name, f[2]), f[1] == "1"
}

// windowAdded asks for the layout of a new window, tmux does not send it
func (s *tmuxSession) windowAdded(id int) {
	if s.windows[id] != nil {
		return
	}
	s.send(fmt.Sprintf("display-message -p -t @%d '#{window_id} #{window_active} #{window_layout} #{window_name}'", id),
		func(lines []string, failed bool) {
			if !failed && len(lines) > 0 {
				if tab, _ := s.windowFromLine(lines[0]); tab != nil {
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
	root := s.buildWindow(id, layout)
	if root == nil {
		return nil
	}

	w := &tmuxWindow{id: id, root: root, tab: container.NewTabItemWithIcon(name, assert.TmuxIconRes, root)}
	s.windows[id] = w
	tabControl.Append(w.tab)
	return w.tab
}

// layoutChanged makes the panes of a window again after its layout, e.g. a
// split, a pane that ended or a new size. The terminals stay.
func (s *tmuxSession) layoutChanged(id int, layout string) {
	w := s.windows[id]
	if w == nil {
		return // not shown yet, windowAdded makes it
	}

	focused := -1
	if c := fyne.CurrentApp().Driver().CanvasForObject(w.root); c != nil {
		if p, ok := c.Focused().(*Pane); ok && p.tmux != nil {
			focused = p.tmux.id
		}
	}

	root := s.buildWindow(id, layout)
	if root == nil {
		return
	}
	w.root = root
	w.tab.Content = root
	tabControl.Refresh()

	if tp := s.panes[focused]; tp != nil && tp.leaf != nil {
		root.lastFocus = tp.leaf
		root.TryFocus()
	}
}

// buildWindow makes the pane tree of a window from its layout, the panes not
// in it any more are gone
func (s *tmuxSession) buildWindow(window int, layout string) *Pane {
	node, err := parseTmuxLayout(layout)
	if err != nil {
		fyne.LogError("tmux: can not read the layout "+layout, err)
		return nil
	}

	used := map[int]bool{}
	root := s.buildNode(node, nil, window, used)
	for id, tp := range s.panes {
		if tp.window == window && !used[id] {
			delete(s.panes, id)
		}
	}

	// The focus goes to the pane tmux has active, else to the first one. A
	// split root has none, NewPane only sets it for a single pane.
	root.lastFocus = nil
	if tp := s.panes[s.active]; tp != nil && tp.window == window && tp.leaf != nil {
		root.lastFocus = tp.leaf
	}
	if root.lastFocus == nil {
		root.lastFocus = root.leaves()[0]
	}
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

	direction := splitRight
	if n.kind == '[' {
		direction = splitDown
	}

	first := n.kids[0]
	rest := &tmuxLayout{kind: n.kind, kids: n.kids[1:]}
	if len(rest.kids) == 1 {
		rest = rest.kids[0]
	}

	// The part of the first, by the size in units the parts need to show
	// their cells, with the dividers and paddings of the panes
	a, b := first.outer(direction), rest.outer(direction)
	p := newSplitPane(parent, direction, a/(a+b))
	p.first = s.buildNode(first, p, window, used)
	p.second = s.buildNode(rest, p, window, used)
	return p
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
			term.SetColors(th.Scheme().ANSI)
			screen.SetCursorShape(th.CursorShape())
		}
		tp = &tmuxPane{id: n.pane, term: term, screen: screen}
		s.panes[n.pane] = tp
	}
	tp.window = window
	used[n.pane] = true

	leaf := NewPane(parent, tp.term, false)
	leaf.tmux = tp
	tp.leaf = leaf
	tp.term.Resize(n.h, n.w)

	if created {
		s.syncPane(tp)
	}
	return leaf
}

// syncPane fills a new pane with what tmux shows in it, with its history,
// and puts the cursor where it is
func (s *tmuxSession) syncPane(tp *tmuxPane) {
	var captured []string
	s.send(fmt.Sprintf("capture-pane -p -e -C -t %%%d -S -%d", tp.id, HistoryLines), func(lines []string, failed bool) {
		if !failed {
			captured = lines
		}
	})
	s.send(fmt.Sprintf("display-message -p -t %%%d '#{cursor_x} #{cursor_y}'", tp.id), func(lines []string, failed bool) {
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
func (s *tmuxSession) output(pane int, data []byte) {
	if tp := s.panes[pane]; tp != nil && len(data) > 0 {
		tp.term.ProcessOutput(data, len(data))
	}
}

// closeWindow removes the tab of a window that is gone in tmux
func (s *tmuxSession) closeWindow(id int) {
	w := s.windows[id]
	if w == nil {
		return
	}
	delete(s.windows, id)
	for pid, tp := range s.panes {
		if tp.window == id {
			delete(s.panes, pid)
		}
	}
	s.removing = true
	tabControl.Remove(w.tab)
	s.removing = false
}

// tabClosed is called when the user closed a tab, the window ends in tmux
func (s *tmuxSession) tabClosed(tab *container.TabItem) {
	if s.removing {
		return
	}
	for id, w := range s.windows {
		if w.tab == tab {
			delete(s.windows, id)
			s.send(fmt.Sprintf("kill-window -t @%d", id), nil)
			return
		}
	}
}

// ownsRoot tells if a pane is the root of a tmux window's tab
func (s *tmuxSession) ownsRoot(p *Pane) bool {
	for _, w := range s.windows {
		if w.root == p {
			return true
		}
	}
	return false
}

// ownsTab tells if a tab is a tmux window
func (s *tmuxSession) ownsTab(tab *container.TabItem) bool {
	for _, w := range s.windows {
		if w.tab == tab {
			return true
		}
	}
	return false
}

// paneFocused tells tmux the pane the user works in
func (s *tmuxSession) paneFocused(tp *tmuxPane) {
	if s.active == tp.id {
		return
	}
	s.active = tp.id
	s.send(fmt.Sprintf("select-pane -t %%%d", tp.id), nil)
}

// paneActivated focuses the pane tmux made active, if its tab is shown
func (s *tmuxSession) paneActivated(id int) {
	s.active = id
	tp := s.panes[id]
	if tp == nil || tp.leaf == nil || !tp.leaf.inShownTab() {
		return
	}
	tp.leaf.root.lastFocus = tp.leaf
	tp.leaf.TryFocus()
}

// selectTab shows the tab of a window tmux made current
func (s *tmuxSession) selectTab(window int) {
	if w := s.windows[window]; w != nil {
		s.selectTabItem(w.tab)
	}
}

func (s *tmuxSession) selectTabItem(tab *container.TabItem) {
	for i, t := range tabControl.Items {
		if t == tab && tabControl.SelectedIndex != i {
			tabControl.Select(i)
		}
	}
}

// split splits a tmux pane, the layout change makes the new pane
func (s *tmuxSession) split(tp *tmuxPane, direction int) {
	flag := "-h"
	if direction == splitDown {
		flag = "-v"
	}
	s.send(fmt.Sprintf("split-window %s -t %%%d", flag, tp.id), nil)
}

func (s *tmuxSession) killPane(tp *tmuxPane) {
	s.send(fmt.Sprintf("kill-pane -t %%%d", tp.id), nil)
}

func (s *tmuxSession) killWindow(window int) {
	s.send(fmt.Sprintf("kill-window -t @%d", window), nil)
}

func (s *tmuxSession) newWindow() {
	s.send("new-window", nil)
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
			s.send(fmt.Sprintf("send-keys -t %%%d -l %s", pane, tmuxQuote(string(data[:n]))), nil)
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
		s.send(fmt.Sprintf("send-keys -t %%%d -H%s", pane, hex.String()), nil)
		data = data[n:]
	}
}

// tmuxQuote quotes text for a tmux command, in double quotes \, " and $ need
// a backslash
func tmuxQuote(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, `$`, `\$`)
	return `"` + r.Replace(s) + `"`
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

func (p *tmuxPty) Resize(int, int)          {}
func (p *tmuxPty) RunCmd(string) error      { return nil }
func (p *tmuxPty) Read([]byte) (int, error) { return 0, io.EOF }
func (p *tmuxPty) Close()                   {}

// decodeTmux undoes the escapes of tmux: "\ooo" (octal) is a byte, "\\" a
// backslash. %output escapes the control characters and the backslash as
// octal, capture-pane -C escapes the backslash as "\\".
func decodeTmux(s string) []byte {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c != '\\' || i+1 >= len(s) {
			out = append(out, c)
			continue
		}
		if s[i+1] == '\\' {
			out = append(out, '\\')
			i++
			continue
		}
		if i+3 < len(s) && isOctal(s[i+1]) && isOctal(s[i+2]) && isOctal(s[i+3]) {
			out = append(out, (s[i+1]-'0')<<6|(s[i+2]-'0')<<3|(s[i+3]-'0'))
			i += 3
			continue
		}
		out = append(out, c)
	}
	return out
}

func isOctal(c byte) bool {
	return c >= '0' && c <= '7'
}

// tmuxLayout is a cell of a tmux window layout: a pane, or cells side by
// side ('{') or one over the other ('[')
type tmuxLayout struct {
	w, h, x, y int
	pane       int
	kind       byte
	kids       []*tmuxLayout
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
