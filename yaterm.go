package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"yaterm/assert"
	"yaterm/theme"
	"yaterm/ui"
)

const appID = "com.github.joexue.yaterm"

func main() {
	// The ID lets Fyne keep the settings (preferences) between runs
	a := app.NewWithID(appID)
	a.SetIcon(assert.YatermIconRes)

	th := theme.NewYatermTheme()
	ui.LoadSettings(a, th)
	a.Settings().SetTheme(th)

	w := a.NewWindow("YaTerm")

	w.Resize(fyne.NewSize(800, 600))

	t := ui.NewTabControl(w)

	w.SetContent(t)
	ui.KeepSizeWhenMinimized(w)
	w.ShowAndRun()
}
