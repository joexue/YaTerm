package ui

import (
    "fmt"
    "fyne.io/fyne/v2"
    //"fyne.io/fyne/v2/app"
    //"fyne.io/fyne/v2/widget"
    "fyne.io/fyne/v2/container"
    "github.com/fyne-io/terminal"
)

type TabControl struct {
    container.DocTabs
}

func NewTabControl(app fyne.App, _ fyne.Window) fyne.CanvasObject {
    i := 0
    tabControl := &TabControl{}
    tabControl.ExtendBaseWidget(tabControl)

    tabControl.CreateTab = func() *container.TabItem {
        i++
        term := terminal.New()
        tab := container.NewTabItem(fmt.Sprintf("Tab %d", i), term)

        go func() {
            _ = term.RunLocalShell()
            fyne.Do(func() {
                tabControl.Remove(tab)
                tabControl.OnClosed(tab)
            })
        }()

        return tab
    }

    tabControl.OnClosed = func(tab *container.TabItem) {
        term := tab.Content.(*terminal.Terminal)
        term.Exit()
        if len(tabControl.Items) == 0 {
            app.Quit()
        }
    }

    tabControl.Append(tabControl.CreateTab())

    return tabControl
}
