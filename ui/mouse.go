package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"yaterm/assert"
)

func (p *Pane) Tapped(pe *fyne.PointEvent) {
	if c := fyne.CurrentApp().Driver().CanvasForObject(p); c != nil {
		c.Focus(p)
	}
}

func (p *Pane) TappedSecondary(pe *fyne.PointEvent) {
	if c := fyne.CurrentApp().Driver().CanvasForObject(p); c != nil {
		c.Focus(p)
	}

	if c := fyne.CurrentApp().Driver().CanvasForObject(p); c != nil {
		hsplitItem := fyne.NewMenuItemWithIcon("Horizontal Split", theme.NewThemedResource(assert.HSplitIconRes), func() {
			p.Split(SplitHorizontal, 0.5)
		})
		vsplitItem := fyne.NewMenuItemWithIcon("Vertical Split", theme.NewThemedResource(assert.VSplitIconRes), func() {
			p.Split(SplitVertical, 0.5)
		})

		separatorItem := fyne.NewMenuItemSeparator()

		closePaneItem := fyne.NewMenuItemWithIcon("Close Pane", theme.Icon(theme.IconNameWindowClose), func() {
			p.term.OnExit = nil
			p.term.Exit()
			p.TryClose()
		})
		closeTabItem := fyne.NewMenuItemWithIcon("Close Tab", theme.Icon(theme.IconNameWindowClose), func() {
			p.root.TearDown()
			if f := p.root.OnTearDown; f != nil {
				f()
			}
		})

		showMarkDownItem := fyne.NewMenuItemWithIcon("Show as Markdown", theme.NewThemedResource(assert.MarkdownIconRes), func() {
			NewMarkdownTab(make([]rune, 1, 1))
		})

		searchItem := fyne.NewMenuItemWithIcon("Search", theme.Icon(theme.IconNameSearch), func() {
		})

		typeAgentItem := fyne.NewMenuItemWithIcon("Type Agent", theme.Icon(theme.IconNameAccount), func() {
		})

		menuItems := []*fyne.MenuItem{hsplitItem, vsplitItem, separatorItem, showMarkDownItem, separatorItem, searchItem, typeAgentItem, separatorItem, closePaneItem, closeTabItem}
		popUpMenu := widget.NewPopUpMenu(fyne.NewMenu("", menuItems...), c)
		popUpMenu.ShowAtRelativePosition(pe.Position, p)
	}
}
