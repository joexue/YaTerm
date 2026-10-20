package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"yaterm/assert"
	"yaterm/theme"
	"yaterm/ui"
)

// appID names YaTerm to Fyne and the system, e.g. its settings are kept in
// a folder of that name. Dots, not slashes: it is a folder name, a macOS
// bundle identifier and a Wayland app ID too.
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

	t := ui.NewTabControl(a, w)

	w.SetContent(t)
	w.ShowAndRun()
}
