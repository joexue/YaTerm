// Package ui is the window of YaTerm: its tabs, the panes they split into,
// the screens that draw the terminals, the keyboard and mouse, the settings,
// and the tabs and panes of tmux control mode.
package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
	"yaterm/assert"
	"yaterm/note"
)

var (
	win fyne.Window

	tabControl *tabs

	tabID, markdownTabID, noteTabID int
)

// showTitle puts title in the title of the window
func showTitle(title string) {
	win.SetTitle("YaTerm: " + title)
}

// closeTab closes the tab of the root pane, a tmux tab by killing its window
func closeTab(root *pane) {
	if s := activeTmux(); s != nil && root.isTmux {
		s.killWindow(root.tabID)
		return
	}

	tabControl.closeTabByObj(root)
}

// newMarkdownTab opens a tab showing text as Markdown
func newMarkdownTab(text string) {
	markdownTabID++

	defaultText := `
# The selection is empty
You need to select text from terminal then show it as markdown style
`
	content := text
	if len(text) == 0 {
		content = defaultText
	}

	// The long time hang caused by io i.e. remote image
	rt := widget.NewRichTextFromMarkdown(content)
	// A scroll container, not rt.Scroll: Fyne drops the renderer of a widget
	// not drawn for a minute (e.g. in a tab in the background), and RichText
	// with its own scroll makes a new one that draws nothing, the tab would
	// be empty when shown again

	mtab := container.NewTabItemWithIcon(fmt.Sprintf("MD %d", markdownTabID), assert.MarkdownIconRes, container.NewScroll(rt))
	fyne.Do(func() {
		tabControl.append(mtab)
	})
}

// newNoteTab opens a tab with an empty note, "Note n" until it is opened from
// or saved to a file, then the file's name
func newNoteTab() *container.TabItem {
	noteTabID++
	n := note.New(win)
	tab := container.NewTabItemWithIcon(fmt.Sprintf("Note %d", noteTabID), assert.NoteIconRes, n)
	n.OnTitle = func(title string) {
		tab.Text = title
		tabControl.Refresh()
	}
	n.OnShortcut = noteShortcut
	tabControl.append(tab)
	return tab
}

// noteShortcut runs the tab shortcuts (new, next and previous tab) typed in a
// note, the note has the focus and would get them. The others are the note's,
// e.g. Copy, the pane shortcuts do nothing on a note tab.
func noteShortcut(sc fyne.Shortcut) bool {
	cs, ok := sc.(*desktop.CustomShortcut)
	if !ok {
		return false
	}
	act, ok := actionFor(cs.KeyName, cs.Modifier)
	if !ok || (act != actNewTab && act != actNextTab && act != actPrevTab) {
		return false
	}
	runWindowAction(act)
	return true
}

// notes are the tabs of the notes, in their order
func notes() []*container.TabItem {
	var tabs []*container.TabItem
	for _, it := range tabControl.items {
		if _, ok := it.Content.(*note.Note); ok {
			tabs = append(tabs, it)
		}
	}
	return tabs
}

// newTab makes a local tab: a pane running the default program
func newTab() *container.TabItem {
	tabID++
	p := newPane(nil, nil, tabID, true)
	tab := container.NewTabItemWithIcon(fmt.Sprintf("Tab %d", tabID), assert.LocalIconRes, p)

	// The title the program sets (OSC 0/2) replaces "Tab n"
	p.onTitle = func(title string) {
		win.SetTitle("YaTerm: " + title)
	}

	return tab
}

// NewTabControl makes the tab bar of window w, with a first tab
func NewTabControl(w fyne.Window) fyne.CanvasObject {
	win = w

	t := newTab()
	tabControl = newTabs(t)

	// On a tmux window the + makes a tmux window, its tab comes from tmux
	tabControl.createTab = func() *container.TabItem {
		if s := activeTmux(); s != nil && tabControl.selectedIndex >= 0 && s.ownsTab(tabControl.focused()) {
			s.newWindow()
			return nil
		}
		return newTab()
	}

	tabControl.onClosed = func(tab *container.TabItem) {
		switch tab.Content.(type) {
		case *pane:
			// A tmux window ends in tmux when its tab is closed, any other
			// tab ends its programs
			if s := activeTmux(); s != nil && s.ownsTab(tab) {
				s.tabClosed(tab)
			} else {
				tab.Content.(*pane).tearDown()
			}
		}
		if len(tabControl.items) == 0 {
			win.Close()
		}
	}

	tabControl.onSelected = func(tab *container.TabItem) {
		switch tab.Content.(type) {
		case *pane:
			if s := activeTmux(); s != nil {
				s.tabFocus(tab)
			}
			tab.Content.(*pane).tryFocus()

		case *note.Note:
			// The note gets the keys, to type in it
			w.Canvas().Focus(tab.Content.(*note.Note).Editor())

		default:
			// A pane of the tab before would keep the focus while hidden and
			// get the keys typed on this tab, so nothing has the focus here,
			// the keys go to the window and its shortcuts
			w.Canvas().Unfocus()
		}
	}

	tabControl.onSettings = func() {
		showSettings(w)
	}

	registerWindowShortcuts()

	go func() {
		fyne.Do(func() {
			win.Canvas().Focus(t.Content.(*pane))
		})
	}()

	return tabControl
}
