package ui

import (
	"fmt"
	"image/color"
	"maps"
	"net/url"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	yatheme "yaterm/theme"
)

// Preference keys of the settings, kept by Fyne between runs
const (
	prefScheme   = "scheme"
	prefFontSize = "fontSize"
)

// The font sizes the setting offers
const (
	minFontSize = 10
	maxFontSize = 32
)

// LoadSettings sets the saved settings into the theme, call it before the
// theme is set to the app
func LoadSettings(a fyne.App, th *yatheme.YatermTheme) {
	p := a.Preferences()
	th.SetScheme(p.StringWithFallback(prefScheme, yatheme.DefaultScheme))
	// A size saved before the limits changed is brought into them
	size := p.FloatWithFallback(prefFontSize, float64(yatheme.DefaultFontSize))
	th.SetFontSize(float32(max(minFontSize, min(size, maxFontSize))))
}

// settings is what the settings dialog edits, applied only on Save
type settings struct {
	scheme   string
	fontSize float32
}

// ShowSettings opens the settings dialog: the kinds of settings in tabs on
// the left, like in Windows Terminal. Save applies them to all panes and
// keeps them, Cancel drops them.
func ShowSettings(w fyne.Window) {
	th := currentTheme()
	if th == nil {
		dialog.ShowInformation("Settings", "The settings need the YaTerm theme.", w)
		return
	}

	s := &settings{scheme: th.Scheme().Name, fontSize: th.FontSize()}

	var d dialog.Dialog
	save := func() {
		d.Hide()
		applySettings(s, th)
	}
	cancel := func() {
		d.Hide()
	}

	// Each page ends with the project's page and the buttons, so the link
	// lines up with the page and not with the tabs on its left. A widget can
	// be in one place only, so every page gets its own row.
	page := func(body fyne.CanvasObject) fyne.CanvasObject {
		return container.NewBorder(nil, bottomRow(save, cancel), nil, nil, body)
	}

	tabs := container.NewAppTabs(
		container.NewTabItem("Appearance", page(appearancePage(s))),
		container.NewTabItem("Shortcuts", page(shortcutsPage())),
	)
	tabs.SetTabLocation(container.TabLocationLeading)

	d = dialog.NewCustomWithoutButtons("Settings", tabs, w)
	d.Resize(fyne.NewSize(720, 480))
	d.Show()
}

// bottomRow is the row at the end of a settings page: the project's page on
// the left, Cancel and Save on the right with some room between them. The
// buttons are made here, not by a confirm dialog, so the link can be in the
// same row.
func bottomRow(save, cancel func()) fyne.CanvasObject {
	buttons, _ := saveCancelButtons(save, cancel)
	return container.NewBorder(nil, nil, projectLink(), buttons)
}

// saveCancelButtons are the Cancel and Save buttons of YaTerm's dialogs, with
// some room between them, to put on the right. The Save button is returned
// too, e.g. to disable it while a form is not valid.
func saveCancelButtons(save, cancel func()) (fyne.CanvasObject, *widget.Button) {
	cancelBtn := widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), cancel)
	saveBtn := widget.NewButtonWithIcon("Save", theme.ConfirmIcon(), save)
	saveBtn.Importance = widget.HighImportance

	gap := canvas.NewRectangle(color.Transparent)
	gap.SetMinSize(fyne.NewSize(theme.Padding()*3, 0))

	return container.NewHBox(cancelBtn, gap, saveBtn), saveBtn
}

// projectURL is the project's page
const projectURL = "https://github.com/joexue/yaterm"

// projectLink is a link to the project's page, opened in the browser
func projectLink() fyne.CanvasObject {
	u, err := url.Parse(projectURL)
	if err != nil {
		return widget.NewLabel(projectURL)
	}
	return widget.NewHyperlink("github.com/joexue/yaterm", u)
}

// appearancePage has the color scheme with a preview and the font size
func appearancePage(s *settings) fyne.CanvasObject {
	preview := newSchemePreview()
	preview.show(yatheme.Schemes[s.scheme], s.fontSize)

	names := slices.Sorted(maps.Keys(yatheme.Schemes))
	scheme := widget.NewSelect(names, func(name string) {
		s.scheme = name
		preview.show(yatheme.Schemes[name], s.fontSize)
	})
	scheme.SetSelected(s.scheme)

	sizeLabel := widget.NewLabel("")
	size := widget.NewSlider(minFontSize, maxFontSize)
	size.Step = 1
	size.OnChanged = func(v float64) {
		s.fontSize = float32(v)
		sizeLabel.SetText(fmt.Sprintf("%.0f", v))
		preview.show(yatheme.Schemes[s.scheme], s.fontSize)
	}
	size.SetValue(float64(s.fontSize))

	form := widget.NewForm(
		widget.NewFormItem("Color scheme", scheme),
		widget.NewFormItem("Font size", container.NewBorder(nil, nil, nil, sizeLabel, size)),
	)

	return container.NewBorder(form, nil, nil, nil, preview.box)
}

// schemePreview shows a scheme: some text in its colors on its background,
// and its 16 colors
type schemePreview struct {
	box    *fyne.Container
	bg     *canvas.Rectangle
	lines  []*canvas.Text
	colors [16]*canvas.Rectangle
}

func newSchemePreview() *schemePreview {
	p := &schemePreview{bg: canvas.NewRectangle(color.Black)}
	p.bg.CornerRadius = 4

	text := container.NewVBox()
	for _, t := range []string{"user@yaterm:~$ ls -la", "drwxr-xr-x  joe  joe  4096  src", "-rw-r--r--  joe  joe   210  README.md"} {
		line := canvas.NewText(t, color.White)
		line.TextStyle = fyne.TextStyle{Monospace: true}
		p.lines = append(p.lines, line)
		text.Add(line)
	}

	normal, bright := container.NewHBox(), container.NewHBox()
	for i := range p.colors {
		r := canvas.NewRectangle(color.Black)
		r.SetMinSize(fyne.NewSize(28, 18))
		p.colors[i] = r
		if i < 8 {
			normal.Add(r)
		} else {
			bright.Add(r)
		}
	}

	content := container.NewVBox(text, layout.NewSpacer(), normal, bright)
	p.box = container.NewStack(p.bg, container.NewPadded(content))
	return p
}

// show draws scheme in the preview, with the text at size
func (p *schemePreview) show(scheme *yatheme.Scheme, size float32) {
	if scheme == nil {
		return
	}

	p.bg.FillColor = scheme.Colors[yatheme.ColorBackground]
	p.bg.Refresh()
	for i, line := range p.lines {
		line.Color = scheme.Colors[yatheme.ColorForeground]
		if i == 0 {
			line.Color = scheme.ANSI[2] // the prompt in green, like many shells
		}
		line.TextSize = size
		line.Refresh()
	}
	for i, r := range p.colors {
		r.FillColor = scheme.ANSI[i]
		r.Refresh()
	}
}

// shortcutsPage lists the shortcuts. They can not be changed yet.
func shortcutsPage() fyne.CanvasObject {
	form := widget.NewForm()
	for _, s := range []struct{ action, keys string }{
		{"Copy", "Ctrl+Shift+C, Ctrl+Insert, or select with the mouse"},
		{"Paste", "Ctrl+Shift+V, Shift+Insert, middle click"},
		{"Select a word", "Double click"},
		{"Search", "Right click > Search"},
		{"Split pane", "Right click > Horizontal / Vertical Split"},
		{"Scroll back", "Mouse wheel"},
	} {
		form.Append(s.action, widget.NewLabel(s.keys))
	}

	note := widget.NewLabel("Changing the shortcuts will come later.")
	note.Importance = widget.LowImportance
	return container.NewBorder(nil, note, nil, nil, form)
}

// applySettings keeps the settings and applies them to the theme and to every
// pane of every tab
func applySettings(s *settings, th *yatheme.YatermTheme) {
	a := fyne.CurrentApp()
	a.Preferences().SetString(prefScheme, s.scheme)
	a.Preferences().SetFloat(prefFontSize, float64(s.fontSize))

	th.SetScheme(s.scheme)
	th.SetFontSize(s.fontSize)

	// Setting the theme again redraws everything in its new colors and size
	a.Settings().SetTheme(th)

	if tabControl == nil {
		return
	}
	for _, tab := range tabControl.Items {
		if p, ok := tab.Content.(*Pane); ok {
			p.applyTheme(th)
		}
	}
}

// applyTheme gives the theme's colors to the terminals of the pane and its
// children, and fits them again to their size, as a new font size changes how
// many rows and columns fit
func (p *Pane) applyTheme(th *yatheme.YatermTheme) {
	if p.split != splitNone {
		p.first.applyTheme(th)
		p.second.applyTheme(th)
		return
	}

	if p.term != nil {
		p.term.SetColors(th.Scheme().ANSI)
	}

	// Resize returns early for the same size, so forget it first
	p.size = fyne.Size{}
	p.Resize(p.Size())
	p.screen.Refresh()
}
