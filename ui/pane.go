package ui

import (
	"image/color"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"yaterm/terminal"
	yatheme "yaterm/theme"
)

/*
 *   Keyboard ----> Pane ----> Terminal State <----> Pty <----> Program
 *                                    |
 *                                    |
 *                                    |
 *                                    V
 *                                  Screen
 */

var (
	_ fyne.CanvasObject      = (*pane)(nil)
	_ fyne.Widget            = (*pane)(nil)
	_ desktop.Keyable        = (*pane)(nil)
	_ fyne.Shortcutable      = (*pane)(nil)
	_ fyne.Tabbable          = (*pane)(nil)
	_ fyne.Tappable          = (*pane)(nil)
	_ fyne.SecondaryTappable = (*pane)(nil)
	_ desktop.Mouseable      = (*pane)(nil)
	_ fyne.Draggable         = (*pane)(nil)
	_ dividerObject          = (*pane)(nil)
)

const (
	splitNone  = iota
	splitRight // the new pane on the right, side by side
	splitDown  // the new pane below, one over the other
)

var (
	paneID          int
	paneBorderWidth = float32(1)
	panePadding     = (paneBorderWidth * 3)
)

var (
	tmuxColor  = color.RGBA{R: 0x1b, G: 0xb9, B: 0x1f, A: 0xff}
	localColor = color.RGBA{R: 0x02, G: 0x77, B: 0xBC, A: 0xff}
)

// pane is a node of a tab's split tree: a terminal with its screen, or a
// split into a first and a second pane with a divider between them
type pane struct {
	widget.BaseWidget

	id    int
	extID int
	tabID int

	size   fyne.Size
	border *canvas.Rectangle

	// Rows and colums, the only source for terminal and screen
	row, col int
	cellSize fyne.Size
	term     *terminal.Terminal
	screen   *screen

	split   int
	ratio   float32
	first   *pane
	second  *pane
	divider *divider

	root      *pane
	parent    *pane
	lastFocus *pane

	// Mouse selection: the last click, to find double clicks, and a drag
	lastClick    time.Time
	lastClickPos fyne.Position
	dragging     bool
	pressed      bool // the primary button went down on the pane itself

	// The find box at the top right, nil until first opened
	search *searchBar

	// tmux control mode: the tmux pane this pane shows, or the session this
	// pane runs (tmux -CC), nil otherwise
	tmuxControl *tmuxSession

	// Where the drag is, in screen coordinates, and the stop for the timer
	// that scrolls while it is past the top or bottom
	dragPos        fyne.Position
	dragScrollStop chan struct{}

	// Modifier keys held down, tracked by KeyDown/KeyUp as TypedKey has no modifiers
	modKeys map[fyne.KeyName]bool
	mods    fyne.KeyModifier

	// The mouse of a program that gets it (mouse tracking): a button is held
	// whose press went to the program, and which one, the cell of the last
	// report, so a move in the same cell is not sent again, the wheel turned
	// less than a notch, and a right click that went to the program, so its
	// tap does not open the menu
	mouseHeld        bool
	mouseBtn         int
	mouseRow         int
	mouseCol         int
	wheelAcc         float32
	skipSecondaryTap bool

	isTmux bool
	// Callback from creater, called on the root with the title of the
	// focused pane's program (OSC 0/2), e.g. to show it in the tab
	onTitle func(title string)
}

// newPane makes a pane in parent (nil for the root of a tab) showing term, or
// a new terminal when term is nil; run starts its program
func newPane(parent *pane, term *terminal.Terminal, tabID int, run bool) *pane {
	var scr *screen

	paneID++

	if term == nil {
		scr = newScreen()
		term = terminal.New(scr)
		if th := currentTheme(); th != nil {
			term.SetColors(th.Scheme().ANSI, queryThemeColor())
			scr.setCursorShape(th.CursorShape())
		}
	} else {
		scr = term.Screen().(*screen)
	}

	b := canvas.NewRectangle(color.Transparent)
	b.StrokeWidth = paneBorderWidth
	b.StrokeColor = color.Transparent

	p := &pane{
		id:       paneID,
		tabID:    tabID,
		border:   b,
		split:    splitNone,
		first:    nil,
		second:   nil,
		parent:   parent,
		term:     term,
		screen:   scr,
		cellSize: measureCell(),
	}

	if p.parent == nil {
		p.lastFocus = p
		p.root = p
	} else {
		p.root = p.parent.root
	}

	p.ExtendBaseWidget(p)

	p.connectTerm()
	if run {
		go func() {
			term.RunCmd(program)
		}()
	}

	return p
}

// newSplitPane makes a pane split in two, e.g. for a tmux layout, the caller
// sets its first and second pane
func newSplitPane(parent *pane, direction int, window int, ratio float32) *pane {
	b := canvas.NewRectangle(color.Transparent)
	b.StrokeWidth = paneBorderWidth
	b.StrokeColor = color.Transparent

	p := &pane{
		id:       paneID,
		extID:    -1,
		tabID:    window,
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
	p.divider = newDivider(p)
	p.ExtendBaseWidget(p)

	paneID++
	return p
}

// take makes p what its child sub is: a terminal, or a split with sub's
// panes, when the other child of p closed
func (p *pane) take(sub *pane) {
	p.split = sub.split
	p.first = sub.first
	p.second = sub.second
	p.ratio = sub.ratio
	// The divider of sub would still set sub's ratio, which is out of the tree
	p.divider = newDivider(p)
	if p.split == splitNone {
		p.term = sub.term
		p.screen = sub.screen
		moveTmuxControl(p, sub)
		if sub.search != nil {
			sub.screen.search(nil, true)
		}
		p.connectTerm()

		p.root.lastFocus = p
		p.term.Resize(p.row, p.col)
	} else {
		p.first.parent = p
		p.second.parent = p
		p.root.lastFocus = p.first
	}
}

// tearDown exits all the terminals in the pane
func (p *pane) tearDown() {
	if p.split == splitNone {
		t := p.term
		t.OnExit = nil
		t.Exit()
	} else {
		p.first.tearDown()
		p.second.tearDown()
	}
}

// tryFocus focuses the pane of the tab that had the focus last, or its first
// one, called by TabControl
func (p *pane) tryFocus() {
	// A tree built from outside, e.g. a tmux layout, may have none yet
	focusPane := p.root.lastFocus
	if focusPane == nil {
		focusPane = p.root
	}

	for {
		if focusPane.split == splitNone {
			break
		} else {
			focusPane = focusPane.first
		}
	}

	if focusPane.split == splitNone {
		focusPane.focus()
	}
}

// focus gives the pane the keyboard focus
func (p *pane) focus() {
	c := fyne.CurrentApp().Driver().CanvasForObject(p)
	if c == nil && win != nil {
		c = win.Canvas()
	}
	if c != nil {
		c.Focus(p)
	}
}

// close closes the pane: its program ends and the other pane of the split
// takes its place, or the tab closes with its last pane
func (p *pane) close() {
	if s := activeTmux(); s != nil && p.isTmux {
		s.killPane(p.extID)
		return
	}

	if p == p.root {
		p.tearDown()
		closeTab(p)
		return
	}

	// Closed already, e.g. its program ended while the user closed it: its
	// parent holds other panes now, taking one of them would break the tree
	pa := p.parent
	if pa == nil || (pa.first != p && pa.second != p) {
		return
	}

	var stay, quit *pane
	if pa.first == p {
		stay = pa.second
		quit = pa.first
	} else {
		stay = pa.first
		quit = pa.second
	}

	pa.take(stay)

	quit.tearDown()

	pa.Refresh()
	// stay may be split: only a pane with a screen takes the focus
	pa.firstLeaf().focus()
}

// firstLeaf is the first pane in p that has a terminal, p itself if it is not
// split
func (p *pane) firstLeaf() *pane {
	for p.split != splitNone {
		p = p.first
	}
	return p
}

// splitPane splits the pane in direction (splitRight or splitDown): its terminal
// goes to the first pane, a new one to the second, which gets the focus
func (p *pane) splitPane(direction int, ratio float32) {
	// tmux splits its pane, the layout change makes the new one here
	if p.isTmux {
		if s := activeTmux(); s != nil {
			s.split(p.extID, direction)
		}
		return
	}

	// The screen moves to the first child, the search bar does not follow
	if p.search != nil {
		p.hideSearch()
		p.search = nil
	}

	if direction == splitRight && p.col < 3 {
		return
	}

	if direction == splitDown && p.row < 3 {
		return
	}

	p.split = direction
	p.first = newPane(p, p.term, p.tabID, false)
	moveTmuxControl(p, p.first)
	p.second = newPane(p, nil, p.tabID, true)

	p.term = nil
	p.screen = nil
	p.divider = newDivider(p)
	p.ratio = ratio
	p.root.lastFocus = p.second
	p.Refresh()
	p.second.focus()
}

// SetRatio sets the share of the first pane, see dividerObject. A tmux pane
// tells tmux the new sizes once the drag ended.
func (p *pane) SetRatio(ratio float32, changing bool) {
	p.ratio = ratio
	p.Refresh()

	if changing {
		return
	}

	if s := activeTmux(); s != nil && p.isTmux {
		s.setRatio(p)
	}
}

// Resize sizes the pane, and its terminal to the rows and columns that fit
func (p *pane) Resize(size fyne.Size) {
	if size.Width <= 0 || size.Height <= 0 || size == p.size {
		return
	}

	p.size = size

	cellSize := measureCell()
	screenSize := fyne.NewSize(size.Width-2*panePadding, size.Height-2*panePadding)
	row, col := int(screenSize.Height/cellSize.Height), int(screenSize.Width/cellSize.Width)
	p.row, p.col = row, col

	// The screen Resize() will be called by UI frame, just handle terminal
	// here. A tmux pane has the size tmux gives it, the tab area of tmux is
	// told to tmux instead.
	if p.term != nil && !p.isTmux {
		p.term.Resize(p.row, p.col)
	}
	if s := activeTmux(); s != nil && p.parent == nil {
		s.sendClientSize(p.row, p.col)
	}

	// Cann't skip it, This will triage the renderer to call Layout
	p.BaseWidget.Resize(size)
}

// SplitDirection is how the pane is split, splitNone when it is a terminal,
// see dividerObject
func (p *pane) SplitDirection() int {
	return p.split
}

// connectTerm points the terminal's callbacks to this pane, done again when
// the terminal moves to another pane on a split or a close
func (p *pane) connectTerm() {
	p.term.OnExit = func() {
		fyne.Do(func() {
			p.close()
		})
	}

	p.term.OnTitleChange = func(title string) {
		fyne.Do(func() {
			if p.root.lastFocus == p {
				p.root.showTitle(title)
			}
		})
	}

	p.term.OnClipboard = func(text string) {
		fyne.Do(func() {
			fyne.CurrentApp().Clipboard().SetContent(text)
		})
	}

	p.term.OnTmuxControl = func() terminal.TmuxHandler {
		return startTmux(p)
	}

	// The screen moves with the terminal, its mouse comes to this pane now
	p.screen.onWheel = p.mouseWheel
	p.screen.onHover = p.mouseHover
}

// inFocusTab tells if the pane is in the tab that is shown
func (p *pane) inFocusTab() bool {
	if tabControl == nil {
		return true // e.g. a pane on its own in a test
	}
	i := tabControl.selectedIndex
	return i >= 0 && i < len(tabControl.items) && tabControl.items[i].Content == p.root
}

// applyTheme gives the theme's colors to the terminals of the pane and its
// children, and fits them again to their size, as a new font size changes how
// many rows and columns fit
func (p *pane) applyTheme(th *yatheme.YatermTheme) {
	if p.split != splitNone {
		p.first.applyTheme(th)
		p.second.applyTheme(th)
		return
	}

	if p.term != nil {
		p.term.SetColors(th.Scheme().ANSI, queryThemeColor())
	}
	p.screen.setCursorShape(th.CursorShape())

	// Resize returns early for the same size, so forget it first
	p.size = fyne.Size{}
	p.Resize(p.Size())
	p.screen.Refresh()
}

// leaves returns the panes under p that show a terminal
func (p *pane) leaves() []*pane {
	if p.split == splitNone {
		return []*pane{p}
	}
	return append(p.first.leaves(), p.second.leaves()...)
}

/*
 * ================================
 *  Focusable
 * ================================
 */

// FocusGained shows the pane focused: its cursor and border, the title of its
// terminal, see fyne.Focusable
func (p *pane) FocusGained() {
	p.root.lastFocus = p
	if s := activeTmux(); s != nil && p.isTmux {
		s.paneFocused(p.extID)
	}
	p.screen.setFocused(true)
	if p.term != nil {
		p.showTitle(p.term.Title())
	}

	if p.parent != nil {
		if p.isTmux {
			// The color theme is from TMUX icon
			p.border.StrokeColor = tmuxColor
		} else {
			p.border.StrokeColor = localColor
		}
		p.border.Refresh()
	}
}

// FocusLost shows the pane unfocused and forgets the modifier keys held, see
// fyne.Focusable
func (p *pane) FocusLost() {
	// The lost pane may be a parent pane which doesn't have screen
	if p.screen != nil {
		p.screen.setFocused(false)
	}

	// Key ups go to the new focus, so forget the modifiers held here
	clear(p.modKeys)
	p.mods = 0

	p.border.StrokeColor = color.Transparent
	p.border.Refresh()
}

/*
 * ================================
 *  Renderer
 * ================================
 */

// CreateRenderer is a private method to Fyne which links this widget to its
// renderer
func (p *pane) CreateRenderer() fyne.WidgetRenderer {
	r := &paneRenderer{
		pane: p,
	}

	return r
}

var _ fyne.WidgetRenderer = (*paneRenderer)(nil)

type paneRenderer struct {
	pane *pane
}

func (r *paneRenderer) Destroy() {
}

func (r *paneRenderer) Layout(size fyne.Size) {
	w := size.Width
	h := size.Height
	switch r.pane.split {
	case splitNone:
		pad := fyne.NewSize(panePadding, panePadding)
		offset := fyne.NewPos(panePadding, panePadding)

		r.pane.border.Resize(size)
		r.pane.border.Move(fyne.NewPos(0, 0))

		r.pane.screen.Resize(size.Subtract(pad).Subtract(pad))
		r.pane.screen.Move(offset)

		// The search bar sits in the top right corner, over the text
		if s := r.pane.search; s != nil {
			margin := theme.Padding() * 2
			min := s.MinSize()
			width := fyne.Min(min.Width, size.Width-2*margin)
			s.Resize(fyne.NewSize(width, min.Height))
			s.Move(fyne.NewPos(size.Width-width-margin, margin))
		}

	case splitRight:
		w1 := (w - splitDefaultThickness) * r.pane.ratio
		w2 := w - splitDefaultThickness - w1

		r.pane.first.Resize(fyne.NewSize(w1, h))
		r.pane.first.Move(fyne.NewPos(0, 0))

		r.pane.divider.Resize(fyne.NewSize(splitDefaultThickness, h))
		r.pane.divider.Move(fyne.NewPos(w1, 0))

		r.pane.second.Resize(fyne.NewSize(w2, h))
		r.pane.second.Move(fyne.NewPos(w1+splitDefaultThickness, 0))

	case splitDown:
		h1 := (h - splitDefaultThickness) * r.pane.ratio
		h2 := h - splitDefaultThickness - h1

		r.pane.first.Resize(fyne.NewSize(w, h1))
		r.pane.first.Move(fyne.NewPos(0, 0))

		r.pane.divider.Resize(fyne.NewSize(w, splitDefaultThickness))
		r.pane.divider.Move(fyne.NewPos(0, h1))

		r.pane.second.Resize(fyne.NewSize(w, h2))
		r.pane.second.Move(fyne.NewPos(0, h1+splitDefaultThickness))
	}
}

func (r *paneRenderer) MinSize() fyne.Size {
	p := panePadding * 2
	return fyne.NewSize(p, p)
}

func (r *paneRenderer) Objects() []fyne.CanvasObject {
	if r.pane.split == splitNone {
		if r.pane.search != nil {
			return []fyne.CanvasObject{r.pane.border, r.pane.screen, r.pane.search}
		}
		return []fyne.CanvasObject{r.pane.border, r.pane.screen}
	} else {
		return []fyne.CanvasObject{r.pane.first, r.pane.divider, r.pane.second}
	}
}

func (r *paneRenderer) Refresh() {
	r.Layout(r.pane.Size())
}

/*
 * ================================
 *  Search
 * ================================
 */

// showSearch opens the search bar at the top right and focuses its entry.
// A selection on one line is put in it as the text to find.
func (p *pane) showSearch() {
	if p.split != splitNone {
		return
	}

	if p.search == nil {
		p.search = newSearchBar(p.screen, p.hideSearch)
		p.Refresh()
	}

	if text := p.screen.selectedText(); text != "" && !strings.Contains(text, "\n") {
		p.search.entry.SetText(text)
	}

	p.search.Show()
	p.Refresh()
	p.search.focus()
	p.search.search()
}

// hideSearch closes the search bar, clears the highlights and gives the
// keyboard back to the terminal
func (p *pane) hideSearch() {
	if p.search == nil {
		return
	}

	p.search.Hide()
	p.screen.search(nil, true)
	p.focus()
}

/*
 * ================================
 *  Program requests (OSC)
 * ================================
 */

// showTitle hands a title to the root's onTitle, an empty one keeps the old
func (p *pane) showTitle(title string) {
	if f := p.root.onTitle; f != nil && title != "" {
		f(title)
	}
}
