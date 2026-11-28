// Package ui is the window of YaTerm: its tabs, the panes they split into,
// the screens that draw the terminals, the keyboard and mouse, the settings,
// and the tabs and panes of tmux control mode.
package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"yaterm/assert"
	"yaterm/markdown"
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

	// The long time hang caused by io i.e. remote image
	page := markdown.New(win, text)
	mtab := container.NewTabItemWithIcon(fmt.Sprintf("MD %d", markdownTabID), assert.MarkdownIconRes, page)
	// As the close button of its tab does
	page.OnClose = func() { tabControl.closeTab(mtab) }
	// The name of the file opened or saved, as a note's
	page.OnTitle = func(title string) {
		mtab.Text = title
		tabControl.Refresh()
	}
	page.OnShortcut = noteShortcut
	tabControl.append(mtab)
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
	// As the close button of its tab does
	n.OnClose = func() { tabControl.closeTab(tab) }
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

// appender is what text can be added to: a note or a Markdown page
type appender interface {
	fyne.CanvasObject
	Append(text string)
}

// textTabs are the tabs of the notes and the Markdown pages, in their order,
// see Add to in the pane's menu
func textTabs() []*container.TabItem {
	var tabs []*container.TabItem
	for _, it := range tabControl.items {
		if _, ok := it.Content.(appender); ok {
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

		case *markdown.Page:
			// The editor gets the keys while editing, the preview none, the
			// window's shortcuts get them, see default
			if e := tab.Content.(*markdown.Page).Editor(); e != nil {
				w.Canvas().Focus(e)
			} else {
				w.Canvas().Unfocus()
			}

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
