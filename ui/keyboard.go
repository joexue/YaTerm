package ui

import (
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"runtime"
	"strings"
	"unicode/utf8"
)

// modifierOf returns the modifier a key is, or 0 when it is not a modifier key
func modifierOf(name fyne.KeyName) fyne.KeyModifier {
	switch name {
	case desktop.KeyShiftLeft, desktop.KeyShiftRight:
		return fyne.KeyModifierShift
	case desktop.KeyControlLeft, desktop.KeyControlRight:
		return fyne.KeyModifierControl
	case desktop.KeyAltLeft, desktop.KeyAltRight:
		return fyne.KeyModifierAlt
	case desktop.KeySuperLeft, desktop.KeySuperRight:
		return fyne.KeyModifierSuper
	}
	return 0
}

// xterm modifier parameter: 1 + shift(1) + alt(2) + ctrl(4)
func modifierParam(mods fyne.KeyModifier) int {
	m := 1
	if mods&fyne.KeyModifierShift != 0 {
		m += 1
	}
	if mods&fyne.KeyModifierAlt != 0 {
		m += 2
	}
	if mods&fyne.KeyModifierControl != 0 {
		m += 4
	}
	return m
}

// csiKey builds "ESC [ final" or with modifiers "ESC [ 1 ; m final", for arrows, Home, End
func csiKey(final byte, mods fyne.KeyModifier) []byte {
	m := modifierParam(mods)
	if m == 1 {
		return []byte{27, '[', final}
	}
	return append([]byte("\x1b[1;"+strconv.Itoa(m)), final)
}

// ss3Key builds "ESC O final" or with modifiers "ESC [ 1 ; m final", for F1-F4
func ss3Key(final byte, mods fyne.KeyModifier) []byte {
	m := modifierParam(mods)
	if m == 1 {
		return []byte{27, 'O', final}
	}
	return append([]byte("\x1b[1;"+strconv.Itoa(m)), final)
}

// tildeKey builds "ESC [ n ~" or with modifiers "ESC [ n ; m ~", for Insert, Delete, PageUp ...
func tildeKey(n int, mods fyne.KeyModifier) []byte {
	m := modifierParam(mods)
	if m == 1 {
		return []byte("\x1b[" + strconv.Itoa(n) + "~")
	}
	return []byte("\x1b[" + strconv.Itoa(n) + ";" + strconv.Itoa(m) + "~")
}

// keySequence returns what xterm sends for a non-printable key, nil if the key
// sends nothing. Printable keys come through TypedRune instead.
func keySequence(name fyne.KeyName, mods fyne.KeyModifier) []byte {
	alt := mods&fyne.KeyModifierAlt != 0
	ctrl := mods&fyne.KeyModifierControl != 0
	shift := mods&fyne.KeyModifierShift != 0

	// Alt as meta prefixes ESC to the simple keys
	withAlt := func(b ...byte) []byte {
		if alt {
			return append([]byte{27}, b...)
		}
		return b
	}

	switch name {
	case fyne.KeyReturn, fyne.KeyEnter:
		return withAlt('\r')

	case fyne.KeyTab:
		if shift {
			return []byte("\x1b[Z")
		}
		return withAlt('\t')

	// DEL, as xterm sends. ConPTY on Windows reads 0x08 as Ctrl+Backspace.
	case fyne.KeyBackspace:
		if ctrl {
			return withAlt(0x08)
		}
		return withAlt(0x7f)

	case fyne.KeyEscape:
		return withAlt(27)

	case fyne.KeySpace:
		if ctrl {
			return withAlt(0)
		}
		if alt {
			return []byte{27, ' '}
		}
		return nil // plain space comes as a rune

	case fyne.KeyUp:
		return csiKey('A', mods)
	case fyne.KeyDown:
		return csiKey('B', mods)
	case fyne.KeyRight:
		return csiKey('C', mods)
	case fyne.KeyLeft:
		return csiKey('D', mods)
	case fyne.KeyHome:
		return csiKey('H', mods)
	case fyne.KeyEnd:
		return csiKey('F', mods)

	case fyne.KeyInsert:
		return tildeKey(2, mods)
	case fyne.KeyDelete:
		return tildeKey(3, mods)
	case fyne.KeyPageUp:
		return tildeKey(5, mods)
	case fyne.KeyPageDown:
		return tildeKey(6, mods)

	case fyne.KeyF1:
		return ss3Key('P', mods)
	case fyne.KeyF2:
		return ss3Key('Q', mods)
	case fyne.KeyF3:
		return ss3Key('R', mods)
	case fyne.KeyF4:
		return ss3Key('S', mods)
	case fyne.KeyF5:
		return tildeKey(15, mods)
	case fyne.KeyF6:
		return tildeKey(17, mods)
	case fyne.KeyF7:
		return tildeKey(18, mods)
	case fyne.KeyF8:
		return tildeKey(19, mods)
	case fyne.KeyF9:
		return tildeKey(20, mods)
	case fyne.KeyF10:
		return tildeKey(21, mods)
	case fyne.KeyF11:
		return tildeKey(23, mods)
	case fyne.KeyF12:
		return tildeKey(24, mods)
	}

	return nil
}

// ctrlChar returns the control character for Ctrl+key, -1 if there is none
func ctrlChar(name fyne.KeyName) int {
	if len(name) == 1 {
		c := name[0]
		switch {
		case c >= 'A' && c <= 'Z':
			return int(c - 'A' + 1) // Ctrl+A = 0x01 ... Ctrl+Z = 0x1a
		case c == '2' || c == '@':
			return 0x00
		case c == '[' || c == '3':
			return 0x1b
		case c == '\\' || c == '4':
			return 0x1c
		case c == ']' || c == '5':
			return 0x1d
		case c == '6' || c == '^':
			return 0x1e
		case c == '/' || c == '-' || c == '7':
			return 0x1f
		case c == '8':
			return 0x7f
		}
	}
	return -1
}

// metaChar returns the character Alt+key prefixes with ESC, -1 if there is none
func metaChar(name fyne.KeyName, shift bool) int {
	if len(name) != 1 {
		return -1
	}

	c := name[0]
	if c >= 'A' && c <= 'Z' {
		if shift {
			return int(c)
		}
		return int(c - 'A' + 'a')
	}

	// Shifted punctuation depends on the keyboard layout, only plain ones
	if !shift && c > ' ' && c < 0x7f {
		return int(c)
	}
	return -1
}

func (p *Pane) TypedRune(r rune) {
	p.screen.ScrollToBottom()
	b := make([]byte, utf8.UTFMax)
	size := utf8.EncodeRune(b, r)
	_, _ = p.term.Write(b[:size])
}

// TypedKey gets keys without Ctrl, Alt or Super, those go to TypedShortcut.
// It is called for printable keys too, which send nothing here as their
// character comes through TypedRune.
func (p *Pane) TypedKey(ke *fyne.KeyEvent) {
	if seq := keySequence(ke.Name, p.mods); seq != nil {
		p.screen.ScrollToBottom()
		_, _ = p.term.Write(seq)
	}
}

// TypedShortcut gets every key with Ctrl, Alt or Super held, and the
// standard shortcuts like copy and paste.
func (p *Pane) TypedShortcut(sc fyne.Shortcut) {
	// On macOS the standard shortcuts are Cmd+key, so they really mean
	// copy/paste, elsewhere they are Ctrl+key which a terminal sends as is.
	mac := runtime.GOOS == "darwin"

	var (
		name fyne.KeyName
		mods = fyne.KeyModifierControl
	)

	switch sc := sc.(type) {
	case *fyne.ShortcutPaste:
		if mac || sc.Secondary { // Cmd+V, Shift+Insert
			p.paste(sc.Clipboard)
			return
		}
		name = fyne.KeyV

	case *fyne.ShortcutCopy:
		if mac || sc.Secondary { // Cmd+C, Ctrl+Insert, no selection to copy yet
			return
		}
		name = fyne.KeyC

	case *fyne.ShortcutCut:
		if sc.Secondary { // Shift+Delete
			name, mods = fyne.KeyDelete, fyne.KeyModifierShift
		} else {
			name = fyne.KeyX
		}

	case *fyne.ShortcutSelectAll:
		name = fyne.KeyA
	case *fyne.ShortcutUndo:
		name = fyne.KeyZ
	case *fyne.ShortcutRedo:
		name = fyne.KeyY

	case *desktop.CustomShortcut:
		name, mods = sc.KeyName, sc.Modifier

	default:
		return
	}

	// Ctrl+Shift+V pastes (Cmd+Shift+V on macOS), Ctrl+Shift+C would copy
	clipMods := fyne.KeyModifierControl | fyne.KeyModifierShift
	if mac {
		clipMods = fyne.KeyModifierSuper | fyne.KeyModifierShift
	}
	if mods == clipMods && (name == fyne.KeyV || name == fyne.KeyC) {
		if name == fyne.KeyV {
			p.paste(fyne.CurrentApp().Clipboard())
		}
		return
	}

	// Super is for the system or the app, not the terminal
	if mods&fyne.KeyModifierSuper != 0 {
		return
	}

	seq := keySequence(name, mods)
	if seq == nil {
		seq = shortcutChars(name, mods)
	}
	if seq != nil {
		p.screen.ScrollToBottom()
		_, _ = p.term.Write(seq)
	}
}

// shortcutChars returns what a printable key sends with Ctrl and/or Alt
func shortcutChars(name fyne.KeyName, mods fyne.KeyModifier) []byte {
	ctrl := mods&fyne.KeyModifierControl != 0
	alt := mods&fyne.KeyModifierAlt != 0
	shift := mods&fyne.KeyModifierShift != 0

	// AltGr on Windows is reported as Ctrl+Alt, the character it makes
	// (e.g. @ on a German keyboard) comes through TypedRune.
	if ctrl && alt && runtime.GOOS == "windows" {
		return nil
	}

	var c int
	if ctrl {
		c = ctrlChar(name)
	} else {
		c = metaChar(name, shift)
	}
	if c < 0 {
		return nil
	}

	if alt {
		return []byte{27, byte(c)}
	}
	return []byte{byte(c)}
}

// paste sends the clipboard text, new lines are sent as Enter (CR) like typing
func (p *Pane) paste(cb fyne.Clipboard) {
	if cb == nil {
		return
	}

	text := cb.Content()
	if text == "" {
		return
	}

	text = strings.ReplaceAll(text, "\r\n", "\r")
	text = strings.ReplaceAll(text, "\n", "\r")

	p.screen.ScrollToBottom()
	_, _ = p.term.Write([]byte(text))
}

// KeyDown and KeyUp track the modifier keys, TypedKey has no modifiers in its
// event, but needs Shift for keys like Shift+Tab and Shift+Arrow.
func (p *Pane) KeyDown(ke *fyne.KeyEvent) {
	if m := modifierOf(ke.Name); m != 0 {
		if p.modKeys == nil {
			p.modKeys = map[fyne.KeyName]bool{}
		}
		p.modKeys[ke.Name] = true
		p.updateMods()
	}
}

func (p *Pane) KeyUp(ke *fyne.KeyEvent) {
	if modifierOf(ke.Name) != 0 {
		delete(p.modKeys, ke.Name)
		p.updateMods()
	}
}

// updateMods works out the held modifiers, left and right keys are tracked
// apart so releasing one side while the other is held keeps the modifier.
func (p *Pane) updateMods() {
	p.mods = 0
	for name := range p.modKeys {
		p.mods |= modifierOf(name)
	}
}

func (p *Pane) AcceptsTab() bool {
	return true
}
