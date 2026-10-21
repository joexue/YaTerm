package ui

import (
	//"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"image/color"
	"strings"
	"time"
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
	_ fyne.CanvasObject      = (*Pane)(nil)
	_ fyne.Widget            = (*Pane)(nil)
	_ desktop.Keyable        = (*Pane)(nil)
	_ fyne.Shortcutable      = (*Pane)(nil)
	_ fyne.Tabbable          = (*Pane)(nil)
	_ fyne.Tappable          = (*Pane)(nil)
	_ fyne.SecondaryTappable = (*Pane)(nil)
	_ desktop.Mouseable      = (*Pane)(nil)
	_ fyne.Draggable         = (*Pane)(nil)
)

const (
	splitNone  = iota
	splitRight // the new pane on the right, side by side
	splitDown  // the new pane below, one over the other
)

var (
	paneId          = 0
	paneBorderWidth = float32(1)
	panePadding     = (paneBorderWidth * 3)
)

type Pane struct {
	widget.BaseWidget

	id     int
	size   fyne.Size
	border *canvas.Rectangle

	// Rows and colums, the only source for terminal and screen
	row, col int
	cellSize fyne.Size
	term     *terminal.Terminal
	screen   *Screen

	split   int
	ratio   float32
	first   *Pane
	second  *Pane
	divider *Divider

	root      *Pane
	parent    *Pane
	lastFocus *Pane

	// Mouse selection: the last click, to find double clicks, and a drag
	lastClick    time.Time
	lastClickPos fyne.Position
	dragging     bool
	pressed      bool // the primary button went down on the pane itself

	// The find box at the top right, nil until first opened
	search *searchBar

	// tmux control mode: the tmux pane this pane shows, or the session this
	// pane runs (tmux -CC), nil otherwise
	tmux        *tmuxPane
	tmuxControl *tmuxSession

	// Where the drag is, in screen coordinates, and the stop for the timer
	// that scrolls while it is past the top or bottom
	dragPos        fyne.Position
	dragScrollStop chan struct{}

	// Modifier keys held down, tracked by KeyDown/KeyUp as TypedKey has no modifiers
	modKeys map[fyne.KeyName]bool
	mods    fyne.KeyModifier

	// Callback from creater, called on the root with the title of the
	// focused pane's program (OSC 0/2), e.g. to show it in the tab
	OnTitle func(title string)
}

func NewPane(parent *Pane, term *terminal.Terminal, run bool) *Pane {
	var screen *Screen

	if term == nil {
		screen = NewScreen()
		term = terminal.New(screen)
		if th := currentTheme(); th != nil {
			term.SetColors(th.Scheme().ANSI, queryThemeColor())
			screen.SetCursorShape(th.CursorShape())
		}
	} else {
		screen = term.GetScreen().(*Screen)
	}

	b := canvas.NewRectangle(color.Transparent)
	b.StrokeWidth = paneBorderWidth
	b.StrokeColor = color.Transparent

	p := &Pane{
		id:       paneId,
		border:   b,
		split:    splitNone,
		first:    nil,
		second:   nil,
		parent:   parent,
		term:     term,
		screen:   screen,
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

	paneId++

	return p
}

// Called by TabControl
func (p *Pane) TearDown() {
	if p.split == splitNone {
		t := p.term
		t.OnExit = nil
		t.Exit()
	} else {
		p.first.TearDown()
		p.second.TearDown()
	}
}

// Called by TabControl
func (p *Pane) TryFocus() {
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
		// A pane not drawn yet, e.g. of a tmux tab made or laid out just now,
		// has no canvas known, the window's finds it in its content
		c := fyne.CurrentApp().Driver().CanvasForObject(focusPane)
		if c == nil && win != nil {
			c = win.Canvas()
		}
		if c != nil {
			c.Focus(focusPane)
		}
	}
}

func (p *Pane) Close() {
	if p == p.root {
		CloseTab(p)
		return
	}

	// Closed already, e.g. its program ended while the user closed it: its
	// parent holds other panes now, taking one of them would break the tree
	pa := p.parent
	if pa == nil || (pa.first != p && pa.second != p) {
		return
	}

	var take, quit *Pane
	if pa.first == p {
		take = pa.second
		quit = pa.first
	} else {
		take = pa.first
		quit = pa.second
	}

	pa.split = take.split
	pa.first = take.first
	pa.second = take.second
	pa.ratio = take.ratio

	if pa.split == splitNone {
		// The screen must go with its terminal, pa still holds the screen of
		// the pane it was split from, which is the closed one when the first
		// pane is closed.
		pa.term = take.term
		pa.screen = take.screen
		moveTmuxControl(take, pa)
		// take's search bar goes away with take, so do its highlights
		if take.search != nil {
			take.screen.Search(nil, true)
		}
		pa.connectTerm()

		pa.root.lastFocus = pa
		// pa keeps the same size, so force to resize the terminal
		pa.term.Resize(pa.row, pa.col)
	} else {
		pa.first.parent = pa
		pa.second.parent = pa
		pa.root.lastFocus = pa.first
	}

	quit.TearDown()

	pa.Refresh()
	pa.TryFocus()
}

// Called by terminal exit
func (p *Pane) TryClose() {
	fyne.Do(func() {
		p.Close()
	})
}

func (p *Pane) Split(direction int, ratio float32) {
	// tmux splits its pane, the layout change makes the new one here
	if p.tmux != nil {
		if s := activeTmux(); s != nil {
			s.split(p.tmux, direction)
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
	p.first = NewPane(p, p.term, false)
	moveTmuxControl(p, p.first)
	p.second = NewPane(p, nil, true)
	p.term = nil

	p.divider = NewDivider(p)
	p.ratio = ratio
	p.root.lastFocus = p.second

	p.Refresh()

	p.second.TryFocus()
}

func (p *Pane) SetRatio(ratio float32) {
	p.ratio = ratio
	p.Refresh()
}

func (p *Pane) Resize(size fyne.Size) {
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
	if p.term != nil && p.tmux == nil {
		p.term.Resize(p.row, p.col)
	}
	if s := activeTmux(); s != nil && p.parent == nil {
		s.sendClientSize(p.row, p.col)
	}

	// Cann't skip it, This will triage the renderer to call Layout
	p.BaseWidget.Resize(size)
}

/*
 * ================================
 *  Focusable
 * ================================
 */
func (p *Pane) FocusGained() {
	p.root.lastFocus = p
	if s := activeTmux(); s != nil && p.tmux != nil {
		s.paneFocused(p.tmux)
	}
	p.screen.SetFocused(true)
	if p.term != nil {
		p.showTitle(p.term.Title())
	}

	if p.parent != nil {
		p.border.StrokeColor = theme.Color(theme.ColorNamePrimary)
		p.border.Refresh()
	}
}

func (p *Pane) FocusLost() {
	p.screen.SetFocused(false)

	// Key ups go to the new focus, so forget the modifiers held here
	clear(p.modKeys)
	p.mods = 0

	p.border.StrokeColor = color.Transparent
	p.border.Refresh()
}

// applyTheme gives the theme's colors to the terminals of the pane and its
// children, and fits them again to their size, as a new font size changes how
// many rows and columns fit
func (p *Pane) applyTheme(th *yatheme.YatermTheme) {
	if p.split != splitNone {
		p.first.applyTheme(th)
		p.second.applyTheme(th)
		return
	}

	if p.term != nil {
		p.term.SetColors(th.Scheme().ANSI, queryThemeColor())
	}
	p.screen.SetCursorShape(th.CursorShape())

	// Resize returns early for the same size, so forget it first
	p.size = fyne.Size{}
	p.Resize(p.Size())
	p.screen.Refresh()
}

/*
 * ================================
 *  Renderer
 * ================================
 */
func (p *Pane) CreateRenderer() fyne.WidgetRenderer {
	r := &paneRenderer{
		pane: p,
	}

	return r
}

var _ fyne.WidgetRenderer = (*paneRenderer)(nil)

type paneRenderer struct {
	pane *Pane
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
		w1 := (w - SplitDefaultThickness) * r.pane.ratio
		w2 := w - SplitDefaultThickness - w1

		r.pane.first.Resize(fyne.NewSize(w1, h))
		r.pane.first.Move(fyne.NewPos(0, 0))

		r.pane.divider.Resize(fyne.NewSize(SplitDefaultThickness, h))
		r.pane.divider.Move(fyne.NewPos(w1, 0))

		r.pane.second.Resize(fyne.NewSize(w2, h))
		r.pane.second.Move(fyne.NewPos(w1+SplitDefaultThickness, 0))

	case splitDown:
		h1 := (h - SplitDefaultThickness) * r.pane.ratio
		h2 := h - SplitDefaultThickness - h1

		r.pane.first.Resize(fyne.NewSize(w, h1))
		r.pane.first.Move(fyne.NewPos(0, 0))

		r.pane.divider.Resize(fyne.NewSize(w, SplitDefaultThickness))
		r.pane.divider.Move(fyne.NewPos(0, h1))

		r.pane.second.Resize(fyne.NewSize(w, h2))
		r.pane.second.Move(fyne.NewPos(0, h1+SplitDefaultThickness))
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
func (p *Pane) showSearch() {
	if p.split != splitNone {
		return
	}

	if p.search == nil {
		p.search = newSearchBar(p.screen, p.hideSearch)
		p.Refresh()
	}

	if text := p.screen.SelectedText(); text != "" && !strings.Contains(text, "\n") {
		p.search.entry.SetText(text)
	}

	p.search.Show()
	p.Refresh()
	p.search.focus()
	p.search.search()
}

// hideSearch closes the search bar, clears the highlights and gives the
// keyboard back to the terminal
func (p *Pane) hideSearch() {
	if p.search == nil {
		return
	}

	p.search.Hide()
	p.screen.Search(nil, true)
	if c := fyne.CurrentApp().Driver().CanvasForObject(p); c != nil {
		c.Focus(p)
	}
}

/*
 * ================================
 *  Program requests (OSC)
 * ================================
 */

// connectTerm points the terminal's callbacks to this pane, done again when
// the terminal moves to another pane on a split or a close
func (p *Pane) connectTerm() {
	p.term.OnExit = p.TryClose
	p.term.OnTitleChange = p.titleChanged
	p.term.OnTmuxControl = func() terminal.TmuxHandler { return startTmux(p) }
	p.term.OnClipboard = func(text string) {
		fyne.Do(func() { fyne.CurrentApp().Clipboard().SetContent(text) })
	}
}

// titleChanged shows a new title from the program, if this pane is the
// focused one. Called on the terminal's reader goroutine.
func (p *Pane) titleChanged(title string) {
	fyne.Do(func() {
		if p.root.lastFocus == p {
			p.root.showTitle(title)
		}
	})
}

// showTitle hands a title to the root's OnTitle, an empty one keeps the old
func (p *Pane) showTitle(title string) {
	if f := p.root.OnTitle; f != nil && title != "" {
		f(title)
	}
}

// newSplitPane makes a pane split in two, e.g. for a tmux layout, the caller
// sets its first and second pane
func newSplitPane(parent *Pane, direction int, ratio float32) *Pane {
	b := canvas.NewRectangle(color.Transparent)
	b.StrokeWidth = paneBorderWidth
	b.StrokeColor = color.Transparent

	p := &Pane{
		id:       paneId,
		border:   b,
		split:    direction,
		ratio:    ratio,
		parent:   parent,
		cellSize: measureCell(),
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
