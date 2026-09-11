package main

import (
	_ "embed"
    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/app"
    "fyne.io/fyne/v2/theme"
    "image/color"
    "yaterm/assert"
    "yaterm/ui"
)

type YatermTheme struct {
	fyne.Theme

	variant fyne.ThemeVariant
}

func (f *YatermTheme) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	return f.Theme.Color(name, f.variant)
}

func main() {
    a := app.New()
    a.SetIcon(assert.YatermIconRes)
    a.Settings().SetTheme(&YatermTheme{Theme: theme.DefaultTheme(), variant: theme.VariantDark})

    w := a.NewWindow("YaTerm")

    w.Resize(fyne.NewSize(800, 600))

    t := ui.NewTabControl(a, w)

    w.SetContent(t)
    w.ShowAndRun()
}
