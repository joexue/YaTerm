package theme

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"image/color"
	"yaterm/assert"
)

type YatermTheme struct {
	fyne.Theme

	variant fyne.ThemeVariant
}

func (f *YatermTheme) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	return f.Theme.Color(name, f.variant)
}

func (f *YatermTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNameSplitThickness:
		return 4
	}

	return f.Theme.Size(name)
}

func (t *YatermTheme) Font(s fyne.TextStyle) fyne.Resource {
	if !s.Monospace {
		return t.Theme.Font(s)
	}

	if s.Bold {
		return assert.ResourceJetBrainsMonoBoldTtf
	} else if s.Italic {
		return assert.ResourceJetBrainsMonoItalicTtf
	}

	return assert.ResourceJetBrainsMonoRegularTtf
}

func NewYatermTheme() *YatermTheme {
	return &YatermTheme{Theme: theme.DarkTheme(), variant: theme.VariantDark}
}
