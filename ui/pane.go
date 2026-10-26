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
	_ DividerObject          = (*Pane)(nil)
)

const (
	splitNone  = iota
	splitRight // the new pane on the right, side by side
	splitDown  // the new pane below, one over the other
)

var (
	paneId          int
	paneBorderWidth = float32(1)
	panePadding     = (paneBorderWidth * 3)
)

type Pane struct {
	widget.BaseWidget

	id    int
	extId int
	tabId int

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
	tmuxControl *tmuxSession

	// Where the drag is, in screen coordinates, and the stop for the timer
	// that scrolls while it is past the top or bottom
	dragPos        fyne.Position
	dragScrollStop chan struct{}

	// Modifier keys held down, tracked by KeyDown/KeyUp as TypedKey has no modifiers
	modKeys map[fyne.KeyName]bool
	mods    fyne.KeyModifier

	isTmux bool
	// Callback from creater, called on the root with the title of the
	// focused pane's program (OSC 0/2), e.g. to show it in the tab
	OnTitle func(title string)
}

func NewPane(parent *Pane, term *terminal.Terminal, tabId int, run bool) *Pane {
	var screen *Screen

	paneId++

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
		tabId:    tabId,
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

	p.ConnectTerm()
	if run {
		go func() {
			term.RunCmd(program)
		}()
	}

	return p
}

func (p *Pane) Take(sub *Pane) {
	p.split = sub.split
	p.first = sub.first
	p.second = sub.second
	p.ratio = sub.ratio
	p.divider = sub.divider
	if p.split == splitNone {
		p.term = sub.term
		p.screen = sub.screen
		MoveTmuxControl(p, sub)
		if sub.search != nil {
			sub.screen.Search(nil, true)
		}
		p.ConnectTerm()

		p.root.lastFocus = p
		p.term.Resize(p.row, p.col)
	} else {
		p.first.parent = p
		p.second.parent = p
		p.root.lastFocus = p.first
	}
}

// Called by TabControl or complete close
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
		focusPane.Focus()
	}
}

func (p *Pane) Focus() {
	c := fyne.CurrentApp().Driver().CanvasForObject(p)
	if c == nil && win != nil {
		c = win.Canvas()
	}
	if c != nil {
		c.Focus(p)
	}
}

func (p *Pane) Close() {
	if s := ActiveTmux(); s != nil && p.isTmux {
		s.KillPane(p.extId)
		return
	}

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

	var stay, quit *Pane
	if pa.first == p {
		stay = pa.second
		quit = pa.first
	} else {
		stay = pa.first
		quit = pa.second
	}

	pa.Take(stay)

	quit.TearDown()

	pa.Refresh()
	pa.TryFocus()
}

func (p *Pane) Split(direction int, ratio float32) {
	// tmux splits its pane, the layout change makes the new one here
	if p.isTmux {
		if s := ActiveTmux(); s != nil {
			s.Split(p.extId, direction)
		}
		return
	}

	// The screen moves to the first child, the search bar does not follow
	if p.search != nil {
		p.HideSearch()
		p.search = nil
	}

	if direction == splitRight && p.col < 3 {
		return
	}

	if direction == splitDown && p.row < 3 {
		return
	}

	p.split = direction
	p.first = NewPane(p, p.term, p.tabId, false)
	MoveTmuxControl(p, p.first)
	p.second = NewPane(p, nil, p.tabId, true)
	p.term = nil
	p.screen = nil

	p.divider = NewDivider(p)
	p.ratio = ratio
	p.root.lastFocus = p.second

	p.Refresh()

	p.second.TryFocus()
}

func (p *Pane) SetRatio(ratio float32) {
	p.ratio = ratio
	p.Refresh()

	if s := ActiveTmux(); s != nil && p.isTmux {
		s.SetRatio(p)
	}
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
	if p.term != nil && !p.isTmux {
		p.term.Resize(p.row, p.col)
	}
	if s := ActiveTmux(); s != nil && p.parent == nil {
		s.SendClientSize(p.row, p.col)
	}

	// Cann't skip it, This will triage the renderer to call Layout
	p.BaseWidget.Resize(size)
}

func (p *Pane) SplitDirection() int {
	return p.split
}

// ConnectTerm points the terminal's callbacks to this pane, done again when
// the terminal moves to another pane on a split or a close
func (p *Pane) ConnectTerm() {
	p.term.OnExit = func() {
		fyne.Do(func() {
			p.Close()
		})
	}

	p.term.OnTitleChange = func(title string) {
		fyne.Do(func() {
			if p.root.lastFocus == p {
				p.root.ShowTitle(title)
			}
		})
	}

	p.term.OnClipboard = func(text string) {
		fyne.Do(func() {
			fyne.CurrentApp().Clipboard().SetContent(text)
		})
	}

	p.term.OnTmuxControl = func() terminal.TmuxHandler {
		return StartTmux(p)
	}
}

// inShownTab tells if the pane is in the tab that is shown
func (p *Pane) InFocusTab() bool {
	if tabControl == nil {
		return true // e.g. a pane on its own in a test
	}
	i := tabControl.SelectedIndex
	return i >= 0 && i < len(tabControl.Items) && tabControl.Items[i].Content == p.root
}

// ApplyTheme gives the theme's colors to the terminals of the pane and its
// children, and fits them again to their size, as a new font size changes how
// many rows and columns fit
func (p *Pane) ApplyTheme(th *yatheme.YatermTheme) {
	if p.split != splitNone {
		p.first.ApplyTheme(th)
		p.second.ApplyTheme(th)
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

// Leaves returns the panes under p that show a terminal
func (p *Pane) Leaves() []*Pane {
	if p.split == splitNone {
		return []*Pane{p}
	}
	return append(p.first.Leaves(), p.second.Leaves()...)
}

/*
 * ================================
 *  Focusable
 * ================================
 */
func (p *Pane) FocusGained() {
	p.root.lastFocus = p
	if s := ActiveTmux(); s != nil && p.isTmux {
		s.PaneFocused(p.extId)
	}
	p.screen.SetFocused(true)
	if p.term != nil {
		p.ShowTitle(p.term.Title())
	}

	if p.parent != nil {
		p.border.StrokeColor = theme.Color(theme.ColorNamePrimary)
		p.border.Refresh()
	}
}

func (p *Pane) FocusLost() {
	// The lost pane may be a parent pane which doesn't have screen
	if p.screen != nil {
		p.screen.SetFocused(false)
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

// ShowSearch opens the search bar at the top right and focuses its entry.
// A selection on one line is put in it as the text to find.
func (p *Pane) ShowSearch() {
	if p.split != splitNone {
		return
	}

	if p.search == nil {
		p.search = newSearchBar(p.screen, p.HideSearch)
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

// HideSearch closes the search bar, clears the highlights and gives the
// keyboard back to the terminal
func (p *Pane) HideSearch() {
	if p.search == nil {
		return
	}

	p.search.Hide()
	p.screen.Search(nil, true)
	p.Focus()
}

/*
 * ================================
 *  Program requests (OSC)
 * ================================
 */

// ShowTitle hands a title to the root's OnTitle, an empty one keeps the old
func (p *Pane) ShowTitle(title string) {
	if f := p.root.OnTitle; f != nil && title != "" {
		f(title)
	}
}
