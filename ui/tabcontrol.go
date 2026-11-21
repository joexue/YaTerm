// Package ui is the window of YaTerm: its tabs, the panes they split into,
// the screens that draw the terminals, the keyboard and mouse, the settings,
// and the tabs and panes of tmux control mode.
package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"yaterm/assert"
)

var (
	win fyne.Window
	app fyne.App

	tabControl *Tabs

	tabID, markdownTabID int
)

// ShowTitle puts title in the title of the window
func ShowTitle(title string) {
	win.SetTitle("YaTerm: " + title)
}

// CloseTab closes the tab of the root pane, a tmux tab by killing its window
func CloseTab(pane *Pane) {
	if s := ActiveTmux(); s != nil && pane.isTmux {
		s.killWindow(pane.tabID)
		return
	}

	tabControl.CloseTabByObj(pane)
}

// NewMarkdownTab opens a tab showing text as Markdown
func NewMarkdownTab(text string) {
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

	mtab := container.NewTabItemWithIcon(fmt.Sprintf("MD %d", markdownTabID), assert.MarkdownIconColorRes, container.NewScroll(rt))
	fyne.Do(func() {
		tabControl.Append(mtab)
	})
}

// NewTab makes a local tab: a pane running the default program
func NewTab() *container.TabItem {
	tabID++
	pane := NewPane(nil, nil, tabID, true)
	tab := container.NewTabItemWithIcon(fmt.Sprintf("Tab %d", tabID), assert.LocalIconRes, pane)

	// The title the program sets (OSC 0/2) replaces "Tab n"
	pane.OnTitle = func(title string) {
		win.SetTitle("YaTerm: " + title)
	}

	return tab
}

// NewTabControl makes the tab bar of window w, with a first tab
func NewTabControl(a fyne.App, w fyne.Window) fyne.CanvasObject {
	win = w
	app = a

	t := NewTab()
	tabControl = NewTabs(t)

	// On a tmux window the + makes a tmux window, its tab comes from tmux
	tabControl.CreateTab = func() *container.TabItem {
		if s := ActiveTmux(); s != nil && tabControl.SelectedIndex >= 0 && s.OwnsTab(tabControl.Focused()) {
			s.NewWindow()
			return nil
		}
		return NewTab()
	}

	tabControl.OnClosed = func(tab *container.TabItem) {
		switch tab.Content.(type) {
		case *Pane:
			// A tmux window ends in tmux when its tab is closed, any other
			// tab ends its programs
			if s := ActiveTmux(); s != nil && s.OwnsTab(tab) {
				s.TabClosed(tab)
			} else {
				tab.Content.(*Pane).TearDown()
			}
		}
		if len(tabControl.Items) == 0 {
			win.Close()
		}
	}

	tabControl.OnSelected = func(tab *container.TabItem) {
		switch tab.Content.(type) {
		case *Pane:
			if s := ActiveTmux(); s != nil {
				s.TabFocus(tab)
			}
			tab.Content.(*Pane).TryFocus()

		default:
			// A pane of the tab before would keep the focus while hidden and
			// get the keys typed on this tab, so nothing has the focus here,
			// the keys go to the window and its shortcuts
			w.Canvas().Unfocus()
		}
	}

	tabControl.OnSettings = func() {
		ShowSettings(w)
	}

	registerWindowShortcuts()

	go func() {
		fyne.Do(func() {
			win.Canvas().Focus(t.Content.(*Pane))
		})
	}()

	return tabControl
}
