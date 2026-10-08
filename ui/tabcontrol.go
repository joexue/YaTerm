package ui

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
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

	tabControl.CreateTab = NewTabItem

	tabControl.OnClosed = func(tab *container.TabItem) {
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
		}
	}

	tabControl.OnSettings = func() {
		dialog.ShowInformation("Infos", "https://github.com/joexue/yaterm", w)
	}

	return tabControl
}

func NewMarkdownTab(text []rune) {
	markdownTabId++

	defaultText := `
# The content is empty
You need to select text from terminal then show them
`
	rt := widget.NewRichTextFromMarkdown(defaultText)
	mtab := container.NewTabItemWithIcon(fmt.Sprintf("MD %d", markdownTabId), theme.NewThemedResource(assert.MarkdownIconRes), rt)
	fyne.Do(func() {
		tabControl.Append(mtab)
	})
}
