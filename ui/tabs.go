package ui

import (
	//"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	//"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	//"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// closeIcon is the small "x" glyph embedded inside a tabButton. It only
// paints its hover background when the pointer is directly over it, so the
// highlight is scoped to the close affordance rather than the whole tab.
type closeIcon struct {
	widget.BaseWidget
	onTapped func()

	bg *canvas.Rectangle
}

func newCloseIcon(onTapped func()) *closeIcon {
	c := &closeIcon{onTapped: onTapped}
	c.ExtendBaseWidget(c)
	return c
}

func (c *closeIcon) CreateRenderer() fyne.WidgetRenderer {
	c.bg = canvas.NewRectangle(color.Transparent)
	c.bg.CornerRadius = theme.Size(theme.SizeNameSelectionRadius)

	icon := widget.NewIcon(theme.CancelIcon())

	return widget.NewSimpleRenderer(container.NewStack(c.bg, container.NewPadded(icon)))
}

func (c *closeIcon) Tapped(*fyne.PointEvent) {
	if c.onTapped != nil {
		c.onTapped()
	}
}

func (c *closeIcon) MouseIn(*desktop.MouseEvent) {
	c.bg.FillColor = theme.Color(theme.ColorNameHover)
	c.bg.Refresh()
}

func (c *closeIcon) MouseMoved(*desktop.MouseEvent) {}

func (c *closeIcon) MouseOut() {
	c.bg.FillColor = color.Transparent
	c.bg.Refresh()
}

// tabButton renders a document tab and its close icon as a single fused
// control: one shared background, one tap target for selecting the tab, and
// the closeIcon handling its own hover/tap independently.
type tabButton struct {
	widget.BaseWidget
	text     string
	icon     fyne.Resource
	selected bool
	onTapped func()
	onClose  func()

	bg      *canvas.Rectangle
	hovered bool
}

func newTabButton(text string, icon fyne.Resource, selected bool, onTapped, onClose func()) *tabButton {
	t := &tabButton{text: text, icon: icon, selected: selected, onTapped: onTapped, onClose: onClose}
	t.ExtendBaseWidget(t)
	return t
}

func (t *tabButton) backgroundColor() color.Color {
	if t.selected {
		return theme.Color(theme.ColorNameBackground)
	}
	if t.hovered {
		return theme.Color(theme.ColorNameHover)
	}
	return color.Transparent
}

func (t *tabButton) CreateRenderer() fyne.WidgetRenderer {
	label := widget.NewLabel(t.text)
	closeBtn := newCloseIcon(t.onClose)

	var leading fyne.CanvasObject
	if t.icon != nil {
		leading = widget.NewIcon(t.icon)
	}

	inner := container.NewBorder(nil, nil, leading, closeBtn, label)

	t.bg = canvas.NewRectangle(t.backgroundColor())
	radius := 2 * theme.Size(theme.SizeNameSelectionRadius)
	t.bg.TopLeftCornerRadius = radius
	t.bg.TopRightCornerRadius = radius

	return widget.NewSimpleRenderer(container.NewStack(t.bg, container.NewPadded(inner)))
}

func (t *tabButton) Tapped(*fyne.PointEvent) {
	if t.onTapped != nil {
		t.onTapped()
	}
}

func (t *tabButton) MouseIn(*desktop.MouseEvent) {
	t.hovered = true
	t.bg.FillColor = t.backgroundColor()
	t.bg.Refresh()
}

func (t *tabButton) MouseMoved(*desktop.MouseEvent) {}

func (t *tabButton) MouseOut() {
	t.hovered = false
	t.bg.FillColor = t.backgroundColor()
	t.bg.Refresh()
}

type Tabs struct {
	widget.BaseWidget
	SelectedIndex int

	Items []*container.TabItem

	CreateTab      func() *container.TabItem
	CloseIntercept func(*container.TabItem)
	OnClosed       func(*container.TabItem)
	OnSelected     func(*container.TabItem)
	OnSettings     func()

	// Internal UI elements
	tabBar       *fyne.Container
	tabScroll    *container.Scroll
	tabBarBg     *canvas.Rectangle
	settingsBtn  *widget.Button
	contentArea  *fyne.Container
	topContainer *fyne.Container
}

func NewTabs(items ...*container.TabItem) *Tabs {
	t := &Tabs{
		Items:         items,
		SelectedIndex: -1,
		tabBar:        container.NewHBox(),
		contentArea:   container.NewMax(),
	}
	t.BaseWidget.ExtendBaseWidget(t)

	t.settingsBtn = widget.NewButtonWithIcon("", theme.SettingsIcon(), func() {
		if t.OnSettings != nil {
			t.OnSettings()
		}
	})
	t.settingsBtn.Importance = widget.LowImportance

	// The tab row scrolls horizontally instead of growing past the window's
	// width: a Scroll container reports a small, fixed MinSize regardless of
	// how many tabs it holds, so adding tabs never forces the window to
	// enlarge - once the row overflows, the extra tabs just scroll into view.
	t.tabScroll = container.NewHScroll(t.tabBar)

	// The tab bar sits on its own background so it reads as a distinct
	// strip from the content area beneath it. The "+" and settings buttons
	// are pinned to the right edge, outside the scrolling tab row, so they
	// stay visible regardless of how many tabs are open or scrolled past.
	t.tabBarBg = canvas.NewRectangle(theme.Color(theme.ColorNameMenuBackground))
	tabBarRow := container.NewBorder(nil, nil, nil, t.settingsBtn, t.tabScroll)
	tabBarWithBg := container.NewStack(t.tabBarBg, tabBarRow)

	t.topContainer = container.NewBorder(tabBarWithBg, nil, nil, nil, t.contentArea)

	if len(items) > 0 {
		t.Select(0)
	} else {
		t.Refresh()
	}

	return t
}

func (t *Tabs) Select(index int) {
	if index < 0 || index >= len(t.Items) {
		return
	}
	t.SelectedIndex = index
	t.Refresh()

	if t.OnSelected != nil {
		t.OnSelected(t.Items[index])
	}
}

func (t *Tabs) Append(item *container.TabItem) {
	t.Items = append(t.Items, item)
	// Newly created tabs become the focused tab.
	t.Select(len(t.Items) - 1)

	// Reveal the newly appended tab if the row is scrolled.
	t.tabScroll.ScrollToOffset(fyne.NewPos(t.tabBar.MinSize().Width, 0))
}

func (t *Tabs) CloseTab(item *container.TabItem) {
	if t.CloseIntercept != nil {
		t.CloseIntercept(item)
		return
	}
	t.Remove(item)
}

func (t *Tabs) Remove(item *container.TabItem) {
	index := -1
	for i, v := range t.Items {
		if v == item {
			index = i
			break
		}
	}
	if index == -1 {
		return
	}

	t.Items = append(t.Items[:index], t.Items[index+1:]...)

	if len(t.Items) == 0 {
		t.SelectedIndex = -1
	} else if t.SelectedIndex >= len(t.Items) {
		t.Select(len(t.Items) - 1)
	} else if t.SelectedIndex == index {
		t.Select(t.SelectedIndex)
	} else if t.SelectedIndex > index {
		t.SelectedIndex--
		t.Refresh()
	} else {
		t.Refresh()
	}

	if t.OnClosed != nil {
		t.OnClosed(item)
	}
}

func (t *Tabs) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(t.topContainer)
}

func (t *Tabs) Refresh() {
	t.tabBar.Objects = nil
	t.contentArea.Objects = nil

	if t.tabBarBg != nil {
		t.tabBarBg.FillColor = theme.Color(theme.ColorNameMenuBackground)
		t.tabBarBg.Refresh()
	}

	// 1. Build and render the existing document tabs
	for i, item := range t.Items {
		idx := i
		itm := item
		selected := idx == t.SelectedIndex

		tab := newTabButton(itm.Text, itm.Icon, selected, func() { t.Select(idx) }, func() { t.CloseTab(itm) })
		t.tabBar.Add(tab)

		if selected {
			t.contentArea.Objects = []fyne.CanvasObject{itm.Content}
			t.contentArea.Refresh()
		}
	}

	addBtn := widget.NewButtonWithIcon("", theme.ContentAddIcon(), func() {
		it := t.CreateTab()
		t.Append(it)

	})

	addBtn.Importance = widget.LowImportance
	t.tabBar.Add(addBtn)

	t.tabBar.Refresh()
	if t.tabScroll != nil {
		t.tabScroll.Refresh()
	}
	t.topContainer.Refresh()
	t.BaseWidget.Refresh()
}

func (t *Tabs) AcceptsTab() bool {
	return true
}
