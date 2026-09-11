package ui

import (
    "fmt"
    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/container"
)

type TabControl struct {
    container.DocTabs
    app fyne.App
}

func NewTabControl(app fyne.App, win fyne.Window) fyne.CanvasObject {
    i := 0
    tabControl := &TabControl{}
    tabControl.ExtendBaseWidget(tabControl)
    tabControl.app = app

    tabControl.CreateTab = func() *container.TabItem {
        i++
        pane := NewPane()
        tab := container.NewTabItem(fmt.Sprintf("Tab %d", i), pane)

        pane.OnClose = func () {
            fyne.Do(func() {
                tabControl.Remove(tab)

                if len(tabControl.Items) == 0 {
                    win.Close()
                }
            })
        }

        go func() {
            fyne.Do(func() {
                pane.Focus()
            })
        }()

        return tab
    }

    tabControl.OnClosed = func(tab *container.TabItem) {
        tab.Content.(*Pane).Close()

        if len(tabControl.Items) == 0 {
            win.Close()
        }
    }

    tabControl.OnSelected = func(tab *container.TabItem) {
        tab.Content.(*Pane).Focus()
    }

    tabControl.Append(tabControl.CreateTab())

    return tabControl
}
