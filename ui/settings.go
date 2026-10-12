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
	prefScheme      = "scheme"
	prefFontSize    = "fontSize"
	prefCursorShape = "cursorShape"
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
	th.SetCursorShape(yatheme.CursorShape(p.StringWithFallback(prefCursorShape, string(yatheme.DefaultCursorShape))))
	bindings = loadBindings(a)
}

// settings is what the settings dialog edits, applied only on Save
type settings struct {
	scheme      string
	fontSize    float32
	cursorShape yatheme.CursorShape
	bindings    map[action]keyBinding
}

// cursorShapeNames are the cursor shapes as the settings show them
var cursorShapeNames = map[yatheme.CursorShape]string{
	yatheme.CursorBlock:     "Block",
	yatheme.CursorBar:       "Bar",
	yatheme.CursorUnderline: "Underline",
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

	s := &settings{
		scheme:      th.Scheme().Name,
		fontSize:    th.FontSize(),
		cursorShape: th.CursorShape(),
		bindings:    copyBindings(bindings),
	}

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
		container.NewTabItem("Shortcuts", page(shortcutsPage(s, w))),
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
	show := func() { preview.show(yatheme.Schemes[s.scheme], s.fontSize, s.cursorShape) }
	show()

	names := slices.Sorted(maps.Keys(yatheme.Schemes))
	scheme := widget.NewSelect(names, func(name string) {
		s.scheme = name
		show()
	})
	scheme.SetSelected(s.scheme)

	sizeLabel := widget.NewLabel("")
	size := widget.NewSlider(minFontSize, maxFontSize)
	size.Step = 1
	size.OnChanged = func(v float64) {
		s.fontSize = float32(v)
		sizeLabel.SetText(fmt.Sprintf("%.0f", v))
		show()
	}
	size.SetValue(float64(s.fontSize))

	var shapes []string
	for _, c := range yatheme.CursorShapes {
		shapes = append(shapes, cursorShapeNames[c])
	}
	cursor := widget.NewRadioGroup(shapes, func(name string) {
		for c, n := range cursorShapeNames {
			if n == name {
				s.cursorShape = c
			}
		}
		show()
	})
	cursor.Horizontal = true
	cursor.Required = true
	cursor.SetSelected(cursorShapeNames[s.cursorShape])

	form := widget.NewForm(
		widget.NewFormItem("Color scheme", scheme),
		widget.NewFormItem("Font size", container.NewBorder(nil, nil, nil, sizeLabel, size)),
		widget.NewFormItem("Cursor shape", cursor),
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

	// The cursor after the prompt: cell sizes the place, cursor is drawn in it
	cell   *canvas.Rectangle
	cursor *canvas.Rectangle
}

func newSchemePreview() *schemePreview {
	p := &schemePreview{bg: canvas.NewRectangle(color.Black)}
	p.bg.CornerRadius = 4

	p.cell = canvas.NewRectangle(color.Transparent)
	p.cursor = canvas.NewRectangle(color.White)
	cursorCell := container.NewStack(p.cell, container.NewWithoutLayout(p.cursor))

	text := container.NewVBox()
	for i, t := range []string{"user@yaterm:~$ ls -la", "drwxr-xr-x  joe  joe  4096  src", "-rw-r--r--  joe  joe   210  README.md"} {
		line := canvas.NewText(t, color.White)
		line.TextStyle = fyne.TextStyle{Monospace: true}
		p.lines = append(p.lines, line)
		if i == 0 {
			// The prompt line ends with the cursor
			text.Add(container.NewHBox(line, cursorCell))
		} else {
			text.Add(line)
		}
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

// show draws scheme in the preview, with the text at size and the cursor in
// shape
func (p *schemePreview) show(scheme *yatheme.Scheme, size float32, shape yatheme.CursorShape) {
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

	cell := fyne.MeasureText("M", size, fyne.TextStyle{Monospace: true})
	p.cell.SetMinSize(cell)
	pos, csize := fyne.NewPos(0, 0), cell
	switch shape {
	case yatheme.CursorBar:
		csize.Width = cursorThickness
	case yatheme.CursorUnderline:
		pos.Y = cell.Height - cursorThickness
		csize.Height = cursorThickness
	}
	p.cursor.Move(pos)
	p.cursor.Resize(csize)
	p.cursor.FillColor = scheme.Colors[yatheme.ColorCursor]
	p.cursor.Refresh()
	p.box.Refresh()
}

// shortcutsPage has YaTerm's own shortcuts, each can be changed, set to
// none or back to its default, and below them the keys that are built in
func shortcutsPage(s *settings, w fyne.Window) fyne.CanvasObject {
	note := widget.NewLabel("")
	note.Importance = widget.LowImportance
	note.Wrapping = fyne.TextWrapWord

	keyLabels := map[action]*widget.Label{}
	refresh := func() {
		for act, l := range keyLabels {
			if b := s.bindings[act]; b.none() {
				l.SetText("None")
			} else {
				l.SetText(b.String())
			}
		}
	}

	// set gives act the key, a key another action has moves over
	set := func(act action, b keyBinding) {
		note.SetText("")
		if !b.none() {
			for other, ob := range s.bindings {
				if other != act && ob == b {
					s.bindings[other] = keyBinding{}
					note.SetText(b.String() + " was the key of " + actionName(other) + ", which has none now.")
				}
			}
		}
		s.bindings[act] = b
		refresh()
	}

	actionsForm := widget.NewForm()
	for _, a := range actions {
		act := a.id
		key := widget.NewLabel("")
		key.TextStyle = fyne.TextStyle{Monospace: true}
		keyLabels[act] = key

		change := widget.NewButton("Change", func() {
			recordKey(w, a.name, func(b keyBinding) { set(act, b) })
		})
		clear := widget.NewButton("None", func() { set(act, keyBinding{}) })
		reset := widget.NewButton("Default", func() { set(act, defaultBindings[act]) })
		for _, b := range []*widget.Button{change, clear, reset} {
			b.Importance = widget.LowImportance
		}

		actionsForm.Append(a.name, container.NewBorder(nil, nil, nil, container.NewHBox(change, clear, reset), key))
	}
	refresh()

	builtIn := widget.NewForm()
	for _, k := range []struct{ action, keys string }{
		{"Copy", "Ctrl+Shift+C, Ctrl+Insert, or select with the mouse"},
		{"Paste", "Ctrl+Shift+V, Shift+Insert, middle click"},
		{"Select a word", "Double click"},
		{"Search", "Right click -> Search"},
		{"Scroll back", "Mouse wheel"},
	} {
		builtIn.Append(k.action, widget.NewLabel(k.keys))
	}

	title := func(t string) fyne.CanvasObject {
		l := widget.NewLabel(t)
		l.TextStyle = fyne.TextStyle{Bold: true}
		return l
	}
	content := container.NewVBox(title("Actions:"), actionsForm, note, widget.NewSeparator(), title("Built in:"), builtIn)
	return container.NewVScroll(content)
}

// actionName is the name the settings show for an action
func actionName(act action) string {
	for _, a := range actions {
		if a.id == act {
			return a.name
		}
	}
	return string(act)
}

// recordKey asks for the keys of a shortcut, done gets them. Esc cancels.
func recordKey(w fyne.Window, name string, done func(keyBinding)) {
	var d dialog.Dialog
	hint := widget.NewLabel("Press the keys for \"" + name + "\", or Esc to cancel.")
	problem := widget.NewLabel("")
	problem.Importance = widget.DangerImportance

	rec := newKeyRecorder()
	rec.onCancel = func() { d.Hide() }
	rec.onKey = func(b keyBinding) {
		if reason := unusableKey(b); reason != "" {
			problem.SetText(reason)
			return
		}
		d.Hide()
		done(b)
	}

	cancel := widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), func() { d.Hide() })
	content := container.NewVBox(hint, rec, problem, container.NewHBox(layout.NewSpacer(), cancel))
	d = dialog.NewCustomWithoutButtons("Set shortcut", content, w)
	d.Show()
	w.Canvas().Focus(rec)
}

// unusableKey tells why a key can not be a shortcut, "" when it can: a key
// that types something, like a letter or Shift+letter, is needed for typing
func unusableKey(b keyBinding) string {
	if modifierOf(b.Key) != 0 {
		return ""
	}
	typing := b.Mods&(fyne.KeyModifierControl|fyne.KeyModifierAlt|fyne.KeyModifierSuper) == 0
	if typing && len([]rune(string(b.Key))) == 1 {
		return "A letter, number or sign needs Ctrl or Alt, it is needed for typing."
	}
	if typing && (b.Key == fyne.KeySpace || b.Key == fyne.KeyReturn || b.Key == fyne.KeyEnter || b.Key == fyne.KeyBackspace || b.Key == fyne.KeyTab) {
		return string(b.Key) + " needs Ctrl or Alt, it is needed for typing."
	}
	return ""
}

// applySettings keeps the settings and applies them to the theme and to every
// pane of every tab
func applySettings(s *settings, th *yatheme.YatermTheme) {
	a := fyne.CurrentApp()
	a.Preferences().SetString(prefScheme, s.scheme)
	a.Preferences().SetFloat(prefFontSize, float64(s.fontSize))
	a.Preferences().SetString(prefCursorShape, string(s.cursorShape))
	if s.bindings != nil {
		bindings = copyBindings(s.bindings)
		saveBindings(a, bindings)
	}

	th.SetScheme(s.scheme)
	th.SetFontSize(s.fontSize)
	th.SetCursorShape(s.cursorShape)

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
	p.screen.SetCursorShape(th.CursorShape())

	// Resize returns early for the same size, so forget it first
	p.size = fyne.Size{}
	p.Resize(p.Size())
	p.screen.Refresh()
}
