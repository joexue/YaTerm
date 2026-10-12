package ui

import (
	"runtime"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

var (
	_ fyne.Focusable    = (*keyRecorder)(nil)
	_ fyne.Tabbable     = (*keyRecorder)(nil)
	_ fyne.Shortcutable = (*keyRecorder)(nil)
	_ desktop.Keyable   = (*keyRecorder)(nil)
)

// keyRecorder catches the next key with its modifiers, to set a shortcut.
// Keys with Ctrl, Alt or Super come as shortcuts, the others as typed keys,
// the modifiers held for those are tracked by KeyDown and KeyUp.
type keyRecorder struct {
	widget.BaseWidget

	modKeys map[fyne.KeyName]bool
	mods    fyne.KeyModifier
	focused bool

	onKey    func(keyBinding)
	onCancel func()
}

func newKeyRecorder() *keyRecorder {
	r := &keyRecorder{modKeys: map[fyne.KeyName]bool{}}
	r.ExtendBaseWidget(r)
	return r
}

func (r *keyRecorder) FocusGained() {
	r.focused = true
	r.Refresh()
}

func (r *keyRecorder) FocusLost() {
	r.focused = false
	clear(r.modKeys)
	r.mods = 0
	r.Refresh()
}

func (r *keyRecorder) TypedRune(rune) {}

// AcceptsTab lets Tab be a shortcut key too
func (r *keyRecorder) AcceptsTab() bool {
	return true
}

func (r *keyRecorder) KeyDown(ev *fyne.KeyEvent) {
	if m := modifierOf(ev.Name); m != 0 {
		r.modKeys[ev.Name] = true
		r.updateMods()
	}
}

func (r *keyRecorder) KeyUp(ev *fyne.KeyEvent) {
	if modifierOf(ev.Name) != 0 {
		delete(r.modKeys, ev.Name)
		r.updateMods()
	}
}

func (r *keyRecorder) updateMods() {
	r.mods = 0
	for name := range r.modKeys {
		r.mods |= modifierOf(name)
	}
}

// TypedKey gets keys without Ctrl, Alt or Super, e.g. F5 or Shift+Right
func (r *keyRecorder) TypedKey(ev *fyne.KeyEvent) {
	if modifierOf(ev.Name) != 0 {
		return
	}
	if ev.Name == fyne.KeyEscape && r.mods == 0 {
		if r.onCancel != nil {
			r.onCancel()
		}
		return
	}
	r.record(keyBinding{Key: ev.Name, Mods: r.mods & fyne.KeyModifierShift})
}

// TypedShortcut gets keys with Ctrl, Alt or Super. Fyne turns some into its
// standard shortcuts, they are the same keys again.
func (r *keyRecorder) TypedShortcut(sc fyne.Shortcut) {
	ctrl := fyne.KeyModifierControl
	if runtime.GOOS == "darwin" {
		ctrl = fyne.KeyModifierSuper
	}

	switch sc := sc.(type) {
	case *desktop.CustomShortcut:
		r.record(keyBinding{Key: sc.KeyName, Mods: sc.Modifier})
	case *fyne.ShortcutCopy:
		if sc.Secondary {
			r.record(keyBinding{Key: fyne.KeyInsert, Mods: ctrl})
		} else {
			r.record(keyBinding{Key: fyne.KeyC, Mods: ctrl})
		}
	case *fyne.ShortcutPaste:
		if sc.Secondary {
			r.record(keyBinding{Key: fyne.KeyInsert, Mods: fyne.KeyModifierShift})
		} else {
			r.record(keyBinding{Key: fyne.KeyV, Mods: ctrl})
		}
	case *fyne.ShortcutCut:
		if sc.Secondary {
			r.record(keyBinding{Key: fyne.KeyDelete, Mods: fyne.KeyModifierShift})
		} else {
			r.record(keyBinding{Key: fyne.KeyX, Mods: ctrl})
		}
	case *fyne.ShortcutSelectAll:
		r.record(keyBinding{Key: fyne.KeyA, Mods: ctrl})
	case *fyne.ShortcutUndo:
		r.record(keyBinding{Key: fyne.KeyZ, Mods: ctrl})
	case *fyne.ShortcutRedo:
		r.record(keyBinding{Key: fyne.KeyY, Mods: ctrl})
	}
}

func (r *keyRecorder) record(b keyBinding) {
	if r.onKey != nil {
		r.onKey(b)
	}
}

// Tapped takes the focus, e.g. after a click elsewhere in the dialog
func (r *keyRecorder) Tapped(*fyne.PointEvent) {
	if c := fyne.CurrentApp().Driver().CanvasForObject(r); c != nil {
		c.Focus(r)
	}
}

func (r *keyRecorder) CreateRenderer() fyne.WidgetRenderer {
	bg := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
	bg.CornerRadius = theme.InputRadiusSize()
	bg.StrokeWidth = 1
	label := widget.NewLabel("Waiting for keys...")
	label.Alignment = fyne.TextAlignCenter

	refresh := func() {
		bg.FillColor = theme.Color(theme.ColorNameInputBackground)
		bg.StrokeColor = theme.Color(theme.ColorNameInputBorder)
		if r.focused {
			bg.StrokeColor = theme.Color(theme.ColorNamePrimary)
		}
		bg.Refresh()
	}
	refresh()

	return &keyRecorderRenderer{
		WidgetRenderer: widget.NewSimpleRenderer(container.NewStack(bg, label)),
		refresh:        refresh,
	}
}

type keyRecorderRenderer struct {
	fyne.WidgetRenderer
	refresh func()
}

func (r *keyRecorderRenderer) Refresh() {
	r.refresh()
	r.WidgetRenderer.Refresh()
}
