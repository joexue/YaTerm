package ui

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"yaterm/assert"
)

func (p *Pane) Tapped(pe *fyne.PointEvent) {
	p.Focus()
}

func (p *Pane) TappedSecondary(pe *fyne.PointEvent) {
	p.Focus()

	if c := fyne.CurrentApp().Driver().CanvasForObject(p); c != nil {
		splitRightItem := fyne.NewMenuItemWithIcon("Split Right", theme.NewThemedResource(assert.HSplitIconRes), func() {
			p.Split(splitRight, 0.5)
		})
		splitDownItem := fyne.NewMenuItemWithIcon("Split Down", theme.NewThemedResource(assert.VSplitIconRes), func() {
			p.Split(splitDown, 0.5)
		})

		separatorItem := fyne.NewMenuItemSeparator()

		closePaneItem := fyne.NewMenuItemWithIcon("Close Pane", theme.Icon(theme.IconNameWindowClose), func() {
			// tmux closes its pane, the layout change removes it here
			if s := activeTmux(); s != nil && p.isTmux {
				s.killPane(p.id)
				return
			}

			p.Close()
		})

		closeTabItem := fyne.NewMenuItemWithIcon("Close Tab", theme.Icon(theme.IconNameWindowClose), func() {
			if s := activeTmux(); s != nil && p.isTmux {
				s.killWindow(p.extId)
				return
			}

			// Close the root pane, same as close tab
			p.root.Close()
		})

		showMarkDownItem := fyne.NewMenuItemWithIcon("Show as Markdown", theme.NewThemedResource(assert.MarkdownIconRes), func() {
			NewMarkdownTab()
		})

		searchItem := fyne.NewMenuItemWithIcon("Search", theme.Icon(theme.IconNameSearch), func() {
			p.showSearch()
		})

		quickCommandsItem := fyne.NewMenuItemWithIcon("Quick Commands", quickCommandIcon(), nil)
		quickCommandsItem.ChildMenu = p.quickCommandsMenu()

		menuItems := []*fyne.MenuItem{splitRightItem, splitDownItem, separatorItem, showMarkDownItem, separatorItem, searchItem, quickCommandsItem, separatorItem, closePaneItem, closeTabItem}
		popUpMenu := widget.NewPopUpMenu(fyne.NewMenu("", menuItems...), c)
		popUpMenu.ShowAtRelativePosition(pe.Position, p)
	}
}

/*
 * ================================
 *  Selection: drag or double click to select and copy, middle click pastes
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

	case desktop.MouseButtonTertiary:
		p.paste(fyne.CurrentApp().Clipboard())
	}
}

func (p *Pane) MouseUp(*desktop.MouseEvent) {
	p.pressed = false
}

// Dragged selects from where the mouse went down to where it is now
func (p *Pane) Dragged(ev *fyne.DragEvent) {
	if p.split != splitNone {
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
