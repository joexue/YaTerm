package ui

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
)

var i int = 0
var tabControl *Tabs
var win fyne.Window

func NewTabItem() *container.TabItem {
	i++
	pane := NewPane(nil, nil, true)
	tab := container.NewTabItem(fmt.Sprintf("Tab %d", i), pane)

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
		tab.Content.(*Pane).TearDown()
		if len(tabControl.Items) == 0 {
			win.Close()
		}
	}

	tabControl.OnSelected = func(tab *container.TabItem) {
		tab.Content.(*Pane).TryFocus()
	}

	tabControl.OnSettings = func() {
		dialog.ShowInformation("Infos", "https://github.com/joexue/yaterm", w)
	}

	return tabControl
}
