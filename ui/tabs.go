package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"yaterm/assert"
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
	c.bg.FillColor = c.Theme().Color(theme.ColorNameHover, fyne.CurrentApp().Settings().ThemeVariant())
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

// backgroundColor is of the widget's theme, not the app's: in the settings
// preview a tab shows the scheme looked at
func (t *tabButton) backgroundColor() color.Color {
	th, v := t.Theme(), fyne.CurrentApp().Settings().ThemeVariant()
	if t.selected {
		return th.Color(theme.ColorNameBackground, v)
	}
	if t.hovered {
		return th.Color(theme.ColorNameHover, v)
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

// tabs is the tab bar of the window and the content of the selected tab
type tabs struct {
	widget.BaseWidget
	selectedIndex int

	items []*container.TabItem

	createTab      func() *container.TabItem
	closeIntercept func(*container.TabItem)
	onClosed       func(*container.TabItem)
	onSelected     func(*container.TabItem)
	onSettings     func()

	// Internal UI elements
	tabBar       *fyne.Container
	tabScroll    *container.Scroll
	tabBarBg     *canvas.Rectangle
	settingsBtn  *widget.Button
	contentArea  *fyne.Container
	topContainer *fyne.Container
}

// newTabs makes a tab bar with items
func newTabs(items ...*container.TabItem) *tabs {
	t := &tabs{
		items:         items,
		selectedIndex: -1,
		tabBar:        container.NewHBox(),
		contentArea:   container.NewStack(),
	}
	t.BaseWidget.ExtendBaseWidget(t)

	t.settingsBtn = widget.NewButtonWithIcon("", theme.SettingsIcon(), func() {
		if t.onSettings != nil {
			t.onSettings()
		}
		t.selectTab(t.selectedIndex)
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
		t.selectTab(0)
	} else {
		t.Refresh()
	}

	return t
}

// selectTab shows the tab at index, onSelected is told
func (t *tabs) selectTab(index int) {
	if index < 0 || index >= len(t.items) {
		return
	}
	t.selectedIndex = index
	t.Refresh()

	if t.onSelected != nil {
		t.onSelected(t.items[index])
	}
}

// append adds item as the last tab and selects it
func (t *tabs) append(item *container.TabItem) {
	t.items = append(t.items, item)
	// Newly created tabs become the focused tab.
	t.selectTab(len(t.items) - 1)

	// Reveal the newly appended tab if the row is scrolled.
	t.tabScroll.ScrollToOffset(fyne.NewPos(t.tabBar.MinSize().Width, 0))
}

// closeTab closes item as its close button does: closeIntercept decides when
// set, else it is removed
func (t *tabs) closeTab(item *container.TabItem) {
	if t.closeIntercept != nil {
		t.closeIntercept(item)
		return
	}
	t.remove(item)
}

// remove takes item out of the tabs, another one is selected if it was, and
// onClosed is told
func (t *tabs) remove(item *container.TabItem) {
	index := -1
	for i, v := range t.items {
		if v == item {
			index = i
			break
		}
	}
	if index == -1 {
		return
	}

	t.items = append(t.items[:index], t.items[index+1:]...)

	if len(t.items) == 0 {
		t.selectedIndex = -1
	} else if t.selectedIndex >= len(t.items) {
		t.selectTab(len(t.items) - 1)
	} else if t.selectedIndex == index {
		t.selectTab(t.selectedIndex)
	} else if t.selectedIndex > index {
		t.selectedIndex--
		t.Refresh()
	} else {
		t.Refresh()
	}

	if t.onClosed != nil {
		t.onClosed(item)
	}
}

// CreateRenderer is a private method to Fyne which links this widget to its
// renderer
func (t *tabs) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(t.topContainer)
}

// Refresh builds the tab bar again from Items and shows the selected tab
func (t *tabs) Refresh() {
	t.tabBar.Objects = nil
	t.contentArea.Objects = nil

	if t.tabBarBg != nil {
		t.tabBarBg.FillColor = theme.Color(theme.ColorNameMenuBackground)
		t.tabBarBg.Refresh()
	}

	// The icon of a local tab only tells it from the tmux tabs, it is shown
	// while there are some
	hasTmux := false
	for _, item := range t.items {
		if item.Icon == assert.TmuxIconRes {
			hasTmux = true
			break
		}
	}

	// 1. Build and render the existing document tabs
	for i, item := range t.items {
		idx := i
		itm := item
		selected := idx == t.selectedIndex

		icon := itm.Icon
		if icon == assert.LocalIconRes && !hasTmux {
			icon = nil
		}
		tab := newTabButton(itm.Text, icon, selected, func() { t.selectTab(idx) }, func() { t.closeTab(itm) })
		t.tabBar.Add(tab)

		if selected {
			t.contentArea.Objects = []fyne.CanvasObject{itm.Content}
			t.contentArea.Refresh()
		}
	}

	addBtn := widget.NewButtonWithIcon("", theme.ContentAddIcon(), func() {
		// No tab when the new one comes later, e.g. a tmux window
		if it := t.createTab(); it != nil {
			t.append(it)
		}

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

// closeTabByObj closes the tab whose content is obj
func (t *tabs) closeTabByObj(obj fyne.CanvasObject) {
	for _, item := range t.items {
		if item.Content == obj {
			t.closeTab(item)
			break
		}
	}
}

// focused is the selected tab
func (t *tabs) focused() *container.TabItem {
	return t.items[t.selectedIndex]
}

// Tapped moves the focus back to the tab's pane when the user clicks the
// empty area of the tab bar
func (t *tabs) Tapped(pe *fyne.PointEvent) {
	t.selectTab(t.selectedIndex)
}

// TappedSecondary moves the focus back to the tab's pane, as Tapped does
func (t *tabs) TappedSecondary(pe *fyne.PointEvent) {
	t.selectTab(t.selectedIndex)
}

// AcceptsTab keeps Tab from moving the focus away, see fyne.Tabbable
func (t *tabs) AcceptsTab() bool {
	return true
}
