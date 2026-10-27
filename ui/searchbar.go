package ui

import (
	"regexp"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const searchEntryWidth = 200

// SearchBar is the find box shown at the top right of a pane:
// [Find      ] ↑ ↓ Aa ^$ ×
// ↑ and ↓ choose the direction and go to the next match in it, Enter goes on
// in that direction and Shift+Enter the other way, Aa matches case, ^$ takes
// the text as a regular expression, × and Esc close it.
type SearchBar struct {
	widget.BaseWidget

	screen  *Screen
	onClose func()

	Entry                     *searchEntry
	up, down, caseBtn, regBtn *widget.Button
	closeBtn                  *widget.Button
	bg                        *canvas.Rectangle

	upward, matchCase, regex bool
}

func NewSearchBar(screen *Screen, onClose func()) *SearchBar {
	b := &SearchBar{screen: screen, onClose: onClose, upward: true}

	b.Entry = newSearchEntry()
	b.Entry.SetPlaceHolder("Find")
	b.Entry.OnChanged = func(string) { b.Search() }
	b.Entry.OnSubmitted = func(string) { b.next(b.upward != b.Entry.shift) }
	b.Entry.onEscape = func() { b.onClose() }

	b.up = widget.NewButtonWithIcon("", theme.MoveUpIcon(), func() {
		b.upward = true
		b.updateButtons()
		b.next(true)
	})
	b.down = widget.NewButtonWithIcon("", theme.MoveDownIcon(), func() {
		b.upward = false
		b.updateButtons()
		b.next(false)
	})
	b.caseBtn = widget.NewButton("Aa", func() {
		b.matchCase = !b.matchCase
		b.updateButtons()
		b.Search()
	})
	b.regBtn = widget.NewButton("^$", func() {
		b.regex = !b.regex
		b.updateButtons()
		b.Search()
	})
	b.closeBtn = widget.NewButtonWithIcon("", theme.CancelIcon(), func() { b.onClose() })
	b.closeBtn.Importance = widget.LowImportance
	b.updateButtons()

	b.bg = canvas.NewRectangle(theme.Color(theme.ColorNameOverlayBackground))
	b.bg.CornerRadius = theme.InputRadiusSize() * 2
	b.bg.StrokeColor = theme.Color(theme.ColorNameSeparator)
	b.bg.StrokeWidth = 1

	b.ExtendBaseWidget(b)
	return b
}

// updateButtons shows the chosen direction and the options that are on as
// filled buttons, like a pressed toggle
func (b *SearchBar) updateButtons() {
	set := func(btn *widget.Button, on bool) {
		btn.Importance = widget.LowImportance
		if on {
			btn.Importance = widget.MediumImportance
		}
		btn.Refresh()
	}
	set(b.up, b.upward)
	set(b.down, !b.upward)
	set(b.caseBtn, b.matchCase)
	set(b.regBtn, b.regex)
}

// search runs the search for the text in the entry, an empty text or an
// invalid regular expression clears the matches
func (b *SearchBar) Search() {
	text := b.Entry.Text
	if text == "" {
		b.screen.Search(nil, b.upward)
		return
	}

	if !b.regex {
		text = regexp.QuoteMeta(text)
	}
	if !b.matchCase {
		text = "(?i)" + text
	}

	re, err := regexp.Compile(text)
	if err != nil {
		// Probably still being typed
		b.screen.Search(nil, b.upward)
		return
	}
	b.screen.Search(re, b.upward)
}

func (b *SearchBar) next(up bool) {
	b.screen.FindNext(up)
}

// focus puts the keyboard focus into the entry
func (b *SearchBar) Focus() {
	// A bar just made has no canvas until the next frame draws it, the
	// window's canvas finds it in the tree already
	c := fyne.CurrentApp().Driver().CanvasForObject(b)
	if c == nil && win != nil {
		c = win.Canvas()
	}
	if c != nil {
		c.Focus(b.Entry)
	}
}

// Tapped takes clicks on the bar between its widgets, so they do not reach
// the pane under it and start a selection there
func (b *SearchBar) Tapped(*fyne.PointEvent) {}

func (b *SearchBar) CreateRenderer() fyne.WidgetRenderer {
	entryBox := container.New(layout.NewGridWrapLayout(fyne.NewSize(searchEntryWidth, b.Entry.MinSize().Height)), b.Entry)
	row := container.NewHBox(entryBox, layout.NewSpacer(), b.up, b.down, b.caseBtn, b.regBtn, b.closeBtn)
	return widget.NewSimpleRenderer(container.NewStack(b.bg, container.NewPadded(row)))
}

// searchEntry is the find entry, it closes the bar on Esc and tracks Shift so
// Shift+Enter searches the other way
type searchEntry struct {
	widget.Entry

	shift    bool
	onEscape func()
}

var _ desktop.Keyable = (*searchEntry)(nil)

func newSearchEntry() *searchEntry {
	e := &searchEntry{}
	e.ExtendBaseWidget(e)
	return e
}

func (e *searchEntry) TypedKey(key *fyne.KeyEvent) {
	if key.Name == fyne.KeyEscape && e.onEscape != nil {
		e.onEscape()
		return
	}
	e.Entry.TypedKey(key)
}

func (e *searchEntry) KeyDown(key *fyne.KeyEvent) {
	if modifierOf(key.Name) == fyne.KeyModifierShift {
		e.shift = true
	}
	e.Entry.KeyDown(key)
}

func (e *searchEntry) KeyUp(key *fyne.KeyEvent) {
	if modifierOf(key.Name) == fyne.KeyModifierShift {
		e.shift = false
	}
	e.Entry.KeyUp(key)
}
