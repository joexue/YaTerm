package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"yaterm/assert"
	"yaterm/theme"
	"yaterm/ui"
)

func main() {
	// The ID lets Fyne keep the settings (preferences) between runs
	a := app.NewWithID("github.com/joexue/yaterm")
	a.SetIcon(assert.YatermIconRes)

	th := theme.NewYatermTheme()
	ui.LoadSettings(a, th)
	a.Settings().SetTheme(th)

	w := a.NewWindow("YaTerm")

	w.Resize(fyne.NewSize(800, 600))

	t := ui.NewTabControl(a, w)

	w.SetContent(t)
	w.ShowAndRun()
}
