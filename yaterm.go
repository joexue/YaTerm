package main

import (
	_ "embed"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"yaterm/assert"
	"yaterm/theme"
	"yaterm/ui"
)

func main() {
	a := app.New()
	a.SetIcon(assert.YatermIconRes)
	a.Settings().SetTheme(theme.NewYatermTheme())

	w := a.NewWindow("YaTerm")

	w.Resize(fyne.NewSize(800, 600))

	t := ui.NewTabControl(a, w)

	w.SetContent(t)
	w.ShowAndRun()
}
