package main

import (
	_ "embed"
    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/app"
    "yaterm/ui"
)

//go:embed Resource/icons/yaterm.png
var yatermIcon []byte

var yatermIconRes = &fyne.StaticResource{
	StaticName:    "yaterm.png",
	StaticContent: yatermIcon,
}

func main() {
    a := app.New()
    a.SetIcon(yatermIconRes)

    w := a.NewWindow("YaTerm")

    w.Resize(fyne.NewSize(800, 600))

    t := ui.NewTabControl(a, w)

    w.SetContent(t)
    w.ShowAndRun()
}
