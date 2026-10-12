package ui

import (
	"encoding/json"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
)

/*
 * ================================
 *  Shortcuts: keys for YaTerm's own actions. Each action has a default key,
 *  which the user can change or set to none, then the key goes to the
 *  terminal as before.
 * ================================
 */

// action is something a shortcut does
type action string

const (
	actNextTab    action = "nextTab"
	actPrevTab    action = "prevTab"
	actPaneRight  action = "paneRight"
	actPaneLeft   action = "paneLeft"
	actPaneUp     action = "paneUp"
	actPaneDown   action = "paneDown"
	actSplitRight action = "splitRight"
	actSplitDown  action = "splitDown"
)

// actions are all the actions, in the order the settings show them, with
// their names there
var actions = []struct {
	id   action
	name string
}{
	{actNextTab, "Next tab"},
	{actPrevTab, "Previous tab"},
	{actPaneRight, "Right pane"},
	{actPaneLeft, "Left pane"},
	{actPaneUp, "Upper pane"},
	{actPaneDown, "Lower pane"},
	{actSplitRight, "Split right"},
	{actSplitDown, "Split down"},
}

// keyBinding is a key with its modifiers, the zero value is no key
type keyBinding struct {
	Key  fyne.KeyName
	Mods fyne.KeyModifier
}

func (b keyBinding) none() bool {
	return b.Key == ""
}

// defaultBindings are the keys when the user changed nothing. The split keys
// look like the divider they make: "\" for panes side by side, "-" for one
// over the other.
var defaultBindings = map[action]keyBinding{
	actNextTab:    {fyne.KeyRight, fyne.KeyModifierControl},
	actPrevTab:    {fyne.KeyLeft, fyne.KeyModifierControl},
	actPaneRight:  {fyne.KeyRight, fyne.KeyModifierAlt},
	actPaneLeft:   {fyne.KeyLeft, fyne.KeyModifierAlt},
	actPaneUp:     {fyne.KeyUp, fyne.KeyModifierAlt},
	actPaneDown:   {fyne.KeyDown, fyne.KeyModifierAlt},
	actSplitRight: {fyne.KeyBackslash, fyne.KeyModifierControl},
	actSplitDown:  {fyne.KeyMinus, fyne.KeyModifierControl},
}

// bindings are the keys in use, the defaults until LoadSettings loads the
// saved ones
var bindings = copyBindings(defaultBindings)

func copyBindings(b map[action]keyBinding) map[action]keyBinding {
	out := make(map[action]keyBinding, len(b))
	for k, v := range b {
		out[k] = v
	}
	return out
}

// prefShortcuts is the preference key of the shortcuts, a JSON object of the
// actions the user changed, "" for none. An action not in it has its default.
const prefShortcuts = "shortcuts"

// loadBindings reads the saved shortcuts over the defaults
func loadBindings(a fyne.App) map[action]keyBinding {
	b := copyBindings(defaultBindings)

	var saved map[action]string
	if data := a.Preferences().String(prefShortcuts); data != "" {
		if err := json.Unmarshal([]byte(data), &saved); err != nil {
			fyne.LogError("Shortcuts: can not read the saved shortcuts", err)
		}
	}
	for act, s := range saved {
		if _, known := defaultBindings[act]; known {
			b[act] = parseBinding(s)
		}
	}
	return b
}

// saveBindings keeps the shortcuts that differ from the defaults
func saveBindings(a fyne.App, b map[action]keyBinding) {
	saved := map[action]string{}
	for act, def := range defaultBindings {
		if b[act] != def {
			saved[act] = b[act].String()
		}
	}
	data, err := json.Marshal(saved)
	if err != nil {
		fyne.LogError("Shortcuts: can not save the shortcuts", err)
		return
	}
	a.Preferences().SetString(prefShortcuts, string(data))
}

// The modifiers in the order they are written
var modifierNames = []struct {
	mod  fyne.KeyModifier
	name string
}{
	{fyne.KeyModifierControl, "Ctrl"},
	{fyne.KeyModifierAlt, "Alt"},
	{fyne.KeyModifierShift, "Shift"},
	{fyne.KeyModifierSuper, "Super"},
}

// String writes a binding like "Ctrl+Shift+Right", "" for none
func (b keyBinding) String() string {
	if b.none() {
		return ""
	}
	var parts []string
	for _, m := range modifierNames {
		if b.Mods&m.mod != 0 {
			parts = append(parts, m.name)
		}
	}
	return strings.Join(append(parts, string(b.Key)), "+")
}

// parseBinding reads what String writes, "" or anything it can not read is
// no key
func parseBinding(s string) keyBinding {
	if s == "" {
		return keyBinding{}
	}

	// The key may be "+" itself
	key := s
	var mods fyne.KeyModifier
	if i := strings.LastIndex(s[:len(s)-1], "+"); i >= 0 {
		key = s[i+1:]
		for _, part := range strings.Split(s[:i], "+") {
			found := false
			for _, m := range modifierNames {
				if part == m.name {
					mods |= m.mod
					found = true
				}
			}
			if !found {
				return keyBinding{}
			}
		}
	}
	return keyBinding{Key: fyne.KeyName(key), Mods: mods}
}

// actionFor returns the action of a key, false when it is not a shortcut
func actionFor(key fyne.KeyName, mods fyne.KeyModifier) (action, bool) {
	if key == "" {
		return "", false
	}
	for act, b := range bindings {
		if b.Key == key && b.Mods == mods {
			return act, true
		}
	}
	return "", false
}

// runShortcut runs the action of a key pressed in the pane, false when the
// key is not a shortcut and goes to the terminal
func (p *Pane) runShortcut(key fyne.KeyName, mods fyne.KeyModifier) bool {
	act, ok := actionFor(key, mods)
	if !ok {
		return false
	}
	p.runAction(act)
	return true
}

// runAction runs an action from this pane. The tab actions work anywhere, the
// pane actions only on a pane of the tab that is shown, a pane of a hidden tab
// may still have the focus, e.g. after a switch to a markdown tab.
func (p *Pane) runAction(act action) {
	switch act {
	case actNextTab:
		switchTab(1)
		return
	case actPrevTab:
		switchTab(-1)
		return
	}

	if !p.inShownTab() {
		return
	}

	switch act {
	case actPaneRight:
		p.focusNeighbor(1, 0)
	case actPaneLeft:
		p.focusNeighbor(-1, 0)
	case actPaneUp:
		p.focusNeighbor(0, -1)
	case actPaneDown:
		p.focusNeighbor(0, 1)
	case actSplitRight:
		p.Split(splitRight, 0.5)
	case actSplitDown:
		p.Split(splitDown, 0.5)
	}
}

// inShownTab tells if the pane is in the tab that is shown
func (p *Pane) inShownTab() bool {
	if tabControl == nil {
		return true // e.g. a pane on its own in a test
	}
	i := tabControl.SelectedIndex
	return i >= 0 && i < len(tabControl.Items) && tabControl.Items[i].Content == p.root
}

// windowShortcuts are the shortcuts registered on the window's canvas
var windowShortcuts []fyne.Shortcut

// registerWindowShortcuts registers the shortcuts on the window's canvas too.
// A focused pane gets the keys itself, the canvas gets them when no pane has
// the focus, e.g. on a markdown tab or after a click outside the panes. Only
// keys with Ctrl, Alt or Super come to the canvas as shortcuts. Call it again
// when the shortcuts change.
func registerWindowShortcuts() {
	if win == nil {
		return
	}
	c := win.Canvas()
	for _, sc := range windowShortcuts {
		c.RemoveShortcut(sc)
	}
	windowShortcuts = nil

	for act, b := range bindings {
		if b.none() || b.Mods&(fyne.KeyModifierControl|fyne.KeyModifierAlt|fyne.KeyModifierSuper) == 0 {
			continue
		}
		sc := &desktop.CustomShortcut{KeyName: b.Key, Modifier: b.Mods}
		c.AddShortcut(sc, func(fyne.Shortcut) { runWindowAction(act) })
		windowShortcuts = append(windowShortcuts, sc)
	}
}

// runWindowAction runs an action that came to the window: the tab actions
// anywhere, the pane actions on the last focused pane of the shown tab
func runWindowAction(act action) {
	switch act {
	case actNextTab:
		switchTab(1)
		return
	case actPrevTab:
		switchTab(-1)
		return
	}

	if p := shownPane(); p != nil {
		p.runAction(act)
	}
}

// shownPane is the last focused pane of the shown tab, nil when the shown
// tab has no panes, e.g. a markdown tab
func shownPane() *Pane {
	if tabControl == nil {
		return nil
	}
	i := tabControl.SelectedIndex
	if i < 0 || i >= len(tabControl.Items) {
		return nil
	}
	root, ok := tabControl.Items[i].Content.(*Pane)
	if !ok {
		return nil
	}
	p := root.lastFocus
	if p == nil {
		p = root
	}
	for p.split != splitNone {
		p = p.first
	}
	return p
}

// switchTab selects the tab by steps from the selected one, round the ends
func switchTab(steps int) {
	if tabControl == nil || len(tabControl.Items) < 2 {
		return
	}
	n := len(tabControl.Items)
	tabControl.Select(((tabControl.SelectedIndex+steps)%n + n) % n)
}

// focusNeighbor moves the focus to the pane next to this one in the direction
// (dx, dy), e.g. (1, 0) is to the right. Of the panes on that side that line
// up with this one, the nearest gets it, of those the one most in line.
func (p *Pane) focusNeighbor(dx, dy int) {
	d := fyne.CurrentApp().Driver()
	at := func(q *Pane) (fyne.Position, fyne.Position) {
		pos := d.AbsolutePositionForObject(q)
		return pos, pos.Add(q.Size())
	}
	from, to := at(p)

	var best *Pane
	bestGap, bestOff := float32(0), float32(0)
	for _, q := range p.root.leaves() {
		if q == p {
			continue
		}
		qFrom, qTo := at(q)

		// The gap to it in the direction, and how far it is off the line,
		// panes that do not overlap this one sideways are not neighbors
		var gap, overlap, off float32
		switch {
		case dx > 0:
			gap, overlap = qFrom.X-to.X, min(to.Y, qTo.Y)-max(from.Y, qFrom.Y)
			off = abs32((qFrom.Y+qTo.Y)/2 - (from.Y+to.Y)/2)
		case dx < 0:
			gap, overlap = from.X-qTo.X, min(to.Y, qTo.Y)-max(from.Y, qFrom.Y)
			off = abs32((qFrom.Y+qTo.Y)/2 - (from.Y+to.Y)/2)
		case dy > 0:
			gap, overlap = qFrom.Y-to.Y, min(to.X, qTo.X)-max(from.X, qFrom.X)
			off = abs32((qFrom.X+qTo.X)/2 - (from.X+to.X)/2)
		default:
			gap, overlap = from.Y-qTo.Y, min(to.X, qTo.X)-max(from.X, qFrom.X)
			off = abs32((qFrom.X+qTo.X)/2 - (from.X+to.X)/2)
		}
		// A divider lies between neighbors, so the gap is a few units
		if gap < -1 || overlap <= 0 {
			continue
		}
		if best == nil || gap < bestGap || gap == bestGap && off < bestOff {
			best, bestGap, bestOff = q, gap, off
		}
	}

	if best != nil {
		if c := d.CanvasForObject(best); c != nil {
			c.Focus(best)
		}
	}
}

// leaves returns the panes under p that show a terminal
func (p *Pane) leaves() []*Pane {
	if p.split == splitNone {
		return []*Pane{p}
	}
	return append(p.first.leaves(), p.second.leaves()...)
}

func abs32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
