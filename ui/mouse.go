package ui

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"yaterm/assert"
	"yaterm/terminal"
)

func (p *Pane) Tapped(pe *fyne.PointEvent) {
	p.Focus()
}

// TappedSecondary is the right click: the menu, or a paste when the buttons
// are swapped in the settings
func (p *Pane) TappedSecondary(pe *fyne.PointEvent) {
	p.Focus()

	// The right click went to the program
	if p.skipSecondaryTap {
		p.skipSecondaryTap = false
		return
	}

	if rightClickPastes {
		if p.split == splitNone {
			p.paste(fyne.CurrentApp().Clipboard())
		}
		return
	}
	p.showMenu(pe.Position)
}

// showMenu opens the pane's menu at pos, a position in the pane
func (p *Pane) showMenu(pos fyne.Position) {
	if c := fyne.CurrentApp().Driver().CanvasForObject(p); c != nil {
		splitRightItem := fyne.NewMenuItemWithIcon("Split Right", theme.NewThemedResource(assert.HSplitIconRes), func() {
			p.Split(splitRight, 0.5)
		})
		splitDownItem := fyne.NewMenuItemWithIcon("Split Down", theme.NewThemedResource(assert.VSplitIconRes), func() {
			p.Split(splitDown, 0.5)
		})

		separatorItem := fyne.NewMenuItemSeparator()

		closePaneItem := fyne.NewMenuItemWithIcon("Close Pane", theme.Icon(theme.IconNameWindowClose), func() {
			p.Close()
		})

		closeTabItem := fyne.NewMenuItemWithIcon("Close Tab", theme.Icon(theme.IconNameWindowClose), func() {
			CloseTab(p.root)
		})

		showMarkDownItem := fyne.NewMenuItemWithIcon("Show as Markdown", theme.NewThemedResource(assert.MarkdownIconRes), func() {
			NewMarkdownTab()
		})

		searchItem := fyne.NewMenuItemWithIcon("Search", theme.Icon(theme.IconNameSearch), func() {
			p.ShowSearch()
		})

		quickCommandsItem := fyne.NewMenuItemWithIcon("Quick Commands", quickCommandIcon(), nil)
		quickCommandsItem.ChildMenu = p.quickCommandsMenu()

		if p.tmuxControl != nil {
			quickCommandsItem.Disabled = true
		}
		menuItems := []*fyne.MenuItem{splitRightItem, splitDownItem, separatorItem, showMarkDownItem, separatorItem, searchItem, quickCommandsItem, separatorItem, closePaneItem, closeTabItem}
		popUpMenu := widget.NewPopUpMenu(fyne.NewMenu("", menuItems...), c)
		popUpMenu.ShowAtRelativePosition(pos, p)
	}
}

/*
 * ================================
 *  Selection: drag or double click to select and copy, middle click pastes
 *  (or right click, see rightClickPastes)
 * ================================
 */

// Two clicks closer than this in time and distance make a double click
const (
	doubleClickTime     = 400 * time.Millisecond
	doubleClickDistance = 4
)

// While a selection is dragged past the top or bottom, the view scrolls this
// often, by one line plus one for each line height the mouse is away, so it
// goes faster the further away it is, up to dragScrollMax lines a step.
const (
	dragScrollInterval = 50 * time.Millisecond
	dragScrollMax      = 10
)

// screenPos converts a position in the pane to one in its screen
func (p *Pane) screenPos(pos fyne.Position) fyne.Position {
	return pos.Subtract(fyne.NewPos(panePadding, panePadding))
}

func (p *Pane) MouseDown(ev *desktop.MouseEvent) {
	if p.split != splitNone {
		return
	}

	// A program that gets the mouse gets the press, not the selection
	if p.mousePress(ev) {
		return
	}

	switch ev.Button {
	case desktop.MouseButtonPrimary:
		p.pressed = true

		// A double click is detected here, not with fyne.DoubleTappable,
		// which would delay every single click and so the focus
		now := time.Now()
		d := ev.Position.Subtract(p.lastClickPos)
		double := now.Sub(p.lastClick) < doubleClickTime &&
			abs(int(d.X)) <= doubleClickDistance && abs(int(d.Y)) <= doubleClickDistance
		p.lastClick, p.lastClickPos = now, ev.Position

		if double {
			p.screen.SelectWord(p.screenPos(ev.Position))
			p.copySelection()
			p.lastClick = time.Time{} // a third click starts again
		} else {
			p.screen.SelectStart(p.screenPos(ev.Position))
		}

	// Middle click pastes, or opens the menu when the buttons are swapped
	case desktop.MouseButtonTertiary:
		if rightClickPastes {
			p.Focus()
			p.showMenu(ev.Position)
		} else {
			p.paste(fyne.CurrentApp().Clipboard())
		}
	}
}

func (p *Pane) MouseUp(ev *desktop.MouseEvent) {
	p.pressed = false
	p.mouseRelease(p.screenPos(ev.Position), ev.Modifier)
}

// Dragged selects from where the mouse went down to where it is now
func (p *Pane) Dragged(ev *fyne.DragEvent) {
	if p.split != splitNone {
		return
	}

	// A button held for the program moves there (modes 1002 and 1003)
	if p.mouseHeld {
		p.sendMouse(terminal.MouseMotion, p.mouseBtn, p.screenPos(ev.Position), p.mods)
		return
	}

	// Only a drag that started with a press on the pane selects. Fyne sends
	// a drag that starts on a button over the pane (e.g. in the search bar)
	// to the pane too, as the button can not be dragged, but the press went
	// to the button.
	if !p.pressed && !p.dragging {
		return
	}

	p.dragging = true
	p.dragPos = p.screenPos(ev.Position)
	p.screen.SelectTo(p.dragPos)

	// Past the top or bottom, keep scrolling even when the mouse stops, as
	// Dragged only comes when it moves
	if p.dragScrollLines() != 0 {
		p.startDragScroll()
	} else {
		p.stopDragScroll()
	}
}

// dragScrollLines returns how many lines to scroll for the drag position,
// positive to go back into the history, 0 when it is inside the screen
func (p *Pane) dragScrollLines() int {
	h := p.screen.cellSize.Height
	switch {
	case p.dragPos.Y < 0:
		return min(1+int(-p.dragPos.Y/h), dragScrollMax)
	case p.dragPos.Y >= p.screen.Size().Height:
		return -min(1+int((p.dragPos.Y-p.screen.Size().Height)/h), dragScrollMax)
	}
	return 0
}

// startDragScroll scrolls the view and extends the selection every
// dragScrollInterval until stopDragScroll
func (p *Pane) startDragScroll() {
	if p.dragScrollStop != nil {
		return
	}

	stop := make(chan struct{})
	p.dragScrollStop = stop
	go func() {
		ticker := time.NewTicker(dragScrollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				fyne.Do(func() {
					// Stopped while this was queued
					if p.dragScrollStop != stop {
						return
					}
					if n := p.dragScrollLines(); n != 0 {
						p.screen.ScrollLines(n)
						// The row is clamped to the screen, so the selection
						// reaches the line that just scrolled into view
						p.screen.SelectTo(p.dragPos)
					}
				})
			}
		}
	}()
}

func (p *Pane) stopDragScroll() {
	if p.dragScrollStop != nil {
		close(p.dragScrollStop)
		p.dragScrollStop = nil
	}
}

// DragEnd copies the selection, like xterm there is no separate copy step
func (p *Pane) DragEnd() {
	// The drag of a button held for the program ends with its release, in
	// case MouseUp does not come, at the last cell reported
	if p.mouseHeld {
		p.mouseHeld = false
		if p.term != nil {
			p.term.SendMouse(terminal.MouseRelease, p.mouseBtn, p.mouseRow, p.mouseCol, mouseMods(p.mods))
		}
		p.Focus()
		return
	}

	p.stopDragScroll()
	if !p.dragging {
		// Not our drag, e.g. it started on a search bar button
		return
	}
	p.dragging = false

	p.Focus()

	p.copySelection()
}

// copySelection puts the selected text into the clipboard
func (p *Pane) copySelection() {
	if text := p.screen.SelectedText(); text != "" {
		fyne.CurrentApp().Clipboard().SetContent(text)
	}
}

/*
 * ================================
 *  Mouse reporting: a program that asked for the mouse (e.g. vim with
 *  "set mouse=a", htop, tmux) gets the clicks, drags, wheel and moves, see
 *  terminal/mouse.go. Shift held keeps the mouse for YaTerm, e.g. to select.
 * ================================
 */

// wantsMouse tells the mouse goes to the program of this pane, with the
// modifier keys held
func (p *Pane) wantsMouse(mods fyne.KeyModifier) bool {
	return p.split == splitNone && p.term != nil && p.tmuxControl == nil &&
		mods&fyne.KeyModifierShift == 0 && p.term.MouseTracking()
}

// mouseButton is the button of the program for a Fyne button, false for
// others
func mouseButton(b desktop.MouseButton) (int, bool) {
	switch b {
	case desktop.MouseButtonPrimary:
		return terminal.MouseLeft, true
	case desktop.MouseButtonTertiary:
		return terminal.MouseMiddle, true
	case desktop.MouseButtonSecondary:
		return terminal.MouseRight, true
	}
	return 0, false
}

// mouseMods are the modifier keys as the program gets them
func mouseMods(m fyne.KeyModifier) int {
	mods := 0
	if m&fyne.KeyModifierShift != 0 {
		mods |= terminal.MouseShift
	}
	if m&fyne.KeyModifierAlt != 0 {
		mods |= terminal.MouseAlt
	}
	if m&fyne.KeyModifierControl != 0 {
		mods |= terminal.MouseCtrl
	}
	return mods
}

// sendMouse reports the mouse at a position in the screen, false when it is
// not reported, e.g. the view is scrolled back or the mode does not report it
func (p *Pane) sendMouse(action terminal.MouseAction, button int, pos fyne.Position, mods fyne.KeyModifier) bool {
	row, col, ok := p.screen.LiveCell(pos)
	if !ok {
		return false
	}
	// A move in the same cell tells nothing new
	if action == terminal.MouseMotion && row == p.mouseRow && col == p.mouseCol {
		return true
	}
	p.mouseRow, p.mouseCol = row, col
	return p.term.SendMouse(action, button, row, col, mouseMods(mods))
}

// mousePress sends a press to the program, false when the mouse stays with
// YaTerm
func (p *Pane) mousePress(ev *desktop.MouseEvent) bool {
	p.skipSecondaryTap = false
	btn, ok := mouseButton(ev.Button)
	if !ok || !p.wantsMouse(ev.Modifier) {
		return false
	}
	if !p.sendMouse(terminal.MousePress, btn, p.screenPos(ev.Position), ev.Modifier) {
		return false
	}
	p.mouseHeld, p.mouseBtn = true, btn
	p.skipSecondaryTap = btn == terminal.MouseRight
	p.Focus()
	return true
}

// mouseRelease sends the release of the button pressed, at a position in
// the screen
func (p *Pane) mouseRelease(pos fyne.Position, mods fyne.KeyModifier) {
	if !p.mouseHeld {
		return
	}
	p.mouseHeld = false
	if p.term != nil {
		p.sendMouse(terminal.MouseRelease, p.mouseBtn, pos, mods)
	}
}

// mouseWheel sends the wheel to the program, one report a notch, false when
// the view is to scroll instead
func (p *Pane) mouseWheel(pos fyne.Position, dy float32) bool {
	if !p.wantsMouse(p.mods) {
		return false
	}
	if _, _, ok := p.screen.LiveCell(pos); !ok {
		return false
	}

	p.wheelAcc += dy
	n := int(p.wheelAcc / scrollNotchUnits)
	p.wheelAcc -= float32(n) * scrollNotchUnits

	dir := terminal.MouseWheelUp
	if n < 0 {
		dir, n = terminal.MouseWheelDown, -n
	}
	for range n {
		if !p.sendMouse(terminal.MouseWheel, dir, pos, p.mods) {
			// e.g. mode 9 has no wheel, the view scrolls
			return false
		}
	}
	return true
}

// mouseHover sends a move with no button down (mode 1003), a move with one
// down comes by Dragged
func (p *Pane) mouseHover(pos fyne.Position) {
	if p.mouseHeld || !p.wantsMouse(p.mods) {
		return
	}
	p.sendMouse(terminal.MouseMotion, terminal.MouseNone, pos, p.mods)
}
