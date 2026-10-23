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

	tabId, markdownTabId int
)

func ShowTitle(title string) {
	win.SetTitle("YaTerm: " + title)
}

func CloseTab(pane *Pane) {
	tabControl.CloseTabByObj(pane)
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

	// The long time hang caused by io i.e. remote image
	rt := widget.NewRichTextFromMarkdown(content)
	// A scroll container, not rt.Scroll: Fyne drops the renderer of a widget
	// not drawn for a minute (e.g. in a tab in the background), and RichText
	// with its own scroll makes a new one that draws nothing, the tab would
	// be empty when shown again

	mtab := container.NewTabItemWithIcon(fmt.Sprintf("MD %d", markdownTabId), assert.MarkdownIconColorRes, container.NewScroll(rt))
	fyne.Do(func() {
		tabControl.Append(mtab)
	})
}

func NewTab() *container.TabItem {
	tabId++
	pane := NewPane(nil, nil, true)
	tab := container.NewTabItem(fmt.Sprintf("Tab %d", tabId), pane)

	// The title the program sets (OSC 0/2) replaces "Tab n"
	pane.OnTitle = func(title string) {
		win.SetTitle("YaTerm: " + title)
	}

	return tab
}

func NewTabControl(a fyne.App, w fyne.Window) fyne.CanvasObject {
	win = w
	app = a

	t := NewTab()
	tabControl = NewTabs(t)

	// On a tmux window the + makes a tmux window, its tab comes from tmux
	tabControl.CreateTab = func() *container.TabItem {
		if s := activeTmux(); s != nil && tabControl.SelectedIndex >= 0 && s.ownsTab(tabControl.Focused()) {
			s.newWindow()
			return nil
		}
		return NewTab()
	}

	tabControl.OnClosed = func(tab *container.TabItem) {
		switch tab.Content.(type) {
		case *Pane:
			// A tmux window ends in tmux when its tab is closed, any other
			// tab ends its programs
			if s := activeTmux(); s != nil && s.ownsTab(tab) {
				s.tabClosed(tab)
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
			if s := activeTmux(); s != nil {
				s.tabFocus(tab)
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
