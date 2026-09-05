package ui

import (
    "fmt"
    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/container"
)

type TabControl struct {
    container.DocTabs
    panes map[*container.TabItem]*Pane
    app fyne.App
}

func NewTabControl(app fyne.App, win fyne.Window) fyne.CanvasObject {
    i := 0
    tabControl := &TabControl{}
    tabControl.ExtendBaseWidget(tabControl)
    tabControl.panes = make(map[*container.TabItem]*Pane)
    tabControl.app = app

    tabControl.CreateTab = func() *container.TabItem {
        i++
        pane := NewPane(app, win, tabControl)
        tab := container.NewTabItem(fmt.Sprintf("Tab %d", i), pane.GetContainer())

        tabControl.panes[tab] = pane
        pane.OnClose = func () {
            fyne.Do(func() {
                delete(tabControl.panes, tab)
                tabControl.Remove(tab)

                if len(tabControl.Items) == 0 {
                    app.Quit()
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
        p, found := tabControl.panes[tab]

        if found {
            p.Close()
        }

        if len(tabControl.Items) == 0 {
            app.Quit()
        }
    }

    tabControl.OnSelected = func(tab *container.TabItem) {
        p, found := tabControl.panes[tab]

        if found {
            p.Focus()
        }
    }

    tabControl.Append(tabControl.CreateTab())

    return tabControl
}
