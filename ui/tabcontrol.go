package ui

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	//"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"yaterm/assert"
)

var tabId int = 0
var markdownTabId = 0
var tabControl *Tabs
var win fyne.Window

func NewTabItem() *container.TabItem {
	tabId++
	pane := NewPane(nil, nil, true)
	tab := container.NewTabItem(fmt.Sprintf("Tab %d", tabId), pane)

	// The title the program sets (OSC 0/2) replaces "Tab n"
	pane.OnTitle = func(title string) {
		win.SetTitle("YaTerm: " + title)
	}

	// When pane is closed from pane internally (e.g. from popup menu)
	pane.OnTearDown = func() {
		tabControl.Remove(tab)

		if len(tabControl.Items) == 0 {
			win.Close()
		}
	}

	go func() {
		fyne.Do(func() {
			pane.TryFocus()
		})
	}()

	return tab
}

func NewTabControl(_ fyne.App, w fyne.Window) fyne.CanvasObject {
	win = w

	t := NewTabItem()
	tabControl = NewTabs(t)

	// On a tmux window the + makes a tmux window, its tab comes from tmux
	tabControl.CreateTab = func() *container.TabItem {
		if s := activeTmux(); s != nil && tabControl.SelectedIndex >= 0 && s.ownsTab(tabControl.Items[tabControl.SelectedIndex]) {
			s.newWindow()
			return nil
		}
		return NewTabItem()
	}

	tabControl.OnClosed = func(tab *container.TabItem) {
		// A tmux window ends in tmux when its tab is closed
		if s := activeTmux(); s != nil {
			s.tabClosed(tab)
		}
		switch tab.Content.(type) {
		case *Pane:
			tab.Content.(*Pane).TearDown()
		}
		if len(tabControl.Items) == 0 {
			win.Close()
		}
	}

	tabControl.OnSelected = func(tab *container.TabItem) {
		switch tab.Content.(type) {
		case *Pane:
			tab.Content.(*Pane).TryFocus()
		default:
			// A pane of the tab before would keep the focus while hidden and
			// get the keys typed on this tab, so nothing has the focus here,
			// the keys go to the window and its shortcuts
			w.Canvas().Unfocus()
		}
	}

	registerWindowShortcuts()

	tabControl.OnSettings = func() {
		ShowSettings(w)
	}

	return tabControl
}

func NewMarkdownTab() {
	markdownTabId++

	defaultText := `
# The content is empty
You need to select text from terminal then show them
`
	content := fyne.CurrentApp().Clipboard().Content()
	if len(content) == 0 {
		content = defaultText
	}
	rt := widget.NewRichTextFromMarkdown(content)
	// A scroll container, not rt.Scroll: Fyne drops the renderer of a widget
	// not drawn for a minute (e.g. in a tab in the background), and RichText
	// with its own scroll makes a new one that draws nothing, the tab would
	// be empty when shown again
	mtab := container.NewTabItemWithIcon(fmt.Sprintf("MD %d", markdownTabId), assert.MarkdownIconRes, container.NewScroll(rt))
	fyne.Do(func() {
		tabControl.Append(mtab)
	})
}
