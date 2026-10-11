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
	splitNone = iota
	splitHorizontal
	splitVertical
)

var paneBorderWidth = float32(1)
var panePadding = (paneBorderWidth * 2)

var id int = 0

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

	// Where the drag is, in screen coordinates, and the stop for the timer
	// that scrolls while it is past the top or bottom
	dragPos        fyne.Position
	dragScrollStop chan struct{}

	// Modifier keys held down, tracked by KeyDown/KeyUp as TypedKey has no modifiers
	modKeys map[fyne.KeyName]bool
	mods    fyne.KeyModifier

	// Callback from creater, called when the whole pane tree is finished
	OnTearDown func()

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
			term.SetColors(th.Scheme().ANSI)
		}
	} else {
		screen = term.GetScreen().(*Screen)
	}

	b := canvas.NewRectangle(color.Transparent)
	b.StrokeWidth = paneBorderWidth
	b.StrokeColor = color.Transparent

	p := &Pane{
		id:       id,
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

	term.OnExit = p.TryClose
	p.connectTerm()
	if run {
		go func() {
			term.RunCmd("xxx")
			//fmt.Println("xxxxxxxxx exit", p.id)
		}()
	}

	id++

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
	fyne.Do(func() {
		focusPane := p.root.lastFocus
		for {
			if focusPane.split == splitNone {
				break
			} else {
				focusPane = focusPane.first
			}
		}

		if focusPane.split == splitNone {
			if c := fyne.CurrentApp().Driver().CanvasForObject(focusPane); c != nil {
				c.Focus(focusPane)
			}
		}
	})
}

func (p *Pane) Close() {
	if p == p.root {
		if f := p.OnTearDown; f != nil {
			f()
		}
		return
	}

	pa := p.parent
	var take *Pane
	if pa.first == p {
		take = pa.second
	} else {
		take = pa.first
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
		// take's search bar goes away with take, so do its highlights
		if take.search != nil {
			take.screen.Search(nil, true)
		}
		pa.term.OnExit = pa.TryClose
		pa.connectTerm()

		pa.root.lastFocus = pa
		// pa keeps the same size, so force to resize the terminal
		pa.term.Resize(pa.row, pa.col)
	} else {
		pa.first.parent = pa
		pa.second.parent = pa
		pa.root.lastFocus = pa.first
	}

	pa.Refresh()
	pa.TryFocus()
}

func (p *Pane) TryClose() {
	fyne.Do(func() {
		p.Close()
	})
}

func (p *Pane) Split(direction int, ratio float32) {
	// The screen moves to the first child, the search bar does not follow
	if p.search != nil {
		p.hideSearch()
		p.search = nil
	}

	if direction == splitHorizontal && p.col < 3 {
		return
	}

	if direction == splitVertical && p.row < 3 {
		return
	}

	p.split = direction
	p.first = NewPane(p, p.term, false)
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

	// The screen Resize() will be called by UI frame, just handle terminal here
	if p.term != nil {
		p.term.Resize(p.row, p.col)
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
	if p.term != nil {
		p.root.showTitle(p.term.Title())
	}

	if p.parent != nil {
		p.border.StrokeColor = theme.Color(theme.ColorNamePrimary)
		p.border.Refresh()
	}
}

func (p *Pane) FocusLost() {
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

	case splitHorizontal:
		w1 := (w - SplitDefaultThickness) * r.pane.ratio
		w2 := w - SplitDefaultThickness - w1

		r.pane.first.Resize(fyne.NewSize(w1, h))
		r.pane.first.Move(fyne.NewPos(0, 0))

		r.pane.divider.Resize(fyne.NewSize(SplitDefaultThickness, h))
		r.pane.divider.Move(fyne.NewPos(w1, 0))

		r.pane.second.Resize(fyne.NewSize(w2, h))
		r.pane.second.Move(fyne.NewPos(w1+SplitDefaultThickness, 0))

	case splitVertical:
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
	p.term.OnTitleChange = p.titleChanged
	p.term.OnClipboard = func(text string) {
		fyne.Do(func() { fyne.CurrentApp().Clipboard().SetContent(text) })
	}
	p.term.OnQueryColor = queryThemeColor
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
	if f := p.OnTitle; f != nil && title != "" {
		f(title)
	}
}

// queryThemeColor answers OSC 10, 11 and 12 with the theme's text,
// background and cursor color
func queryThemeColor(code int) color.Color {
	switch code {
	case 10:
		return theme.Color(theme.ColorNameForeground)
	case 11:
		return theme.Color(theme.ColorNameBackground)
	case 12:
		return theme.Color(theme.ColorNamePrimary)
	}
	return nil
}

// currentTheme is the app's YatermTheme, nil when the app uses another theme
// (e.g. in tests, or inside a container.ThemeOverride)
func currentTheme() *yatheme.YatermTheme {
	th, _ := theme.Current().(*yatheme.YatermTheme)
	return th
}
