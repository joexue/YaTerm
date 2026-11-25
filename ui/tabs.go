package ui

import (
	"image/color"
	"time"

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

	// A drag moves the tab along the bar, see tabs.dragTab
	onDragged func(*tabButton, *fyne.DragEvent)
	onDragEnd func()

	// While a tab is dragged its own button shows where it will drop
	// (dropSlot), and a copy of it (floating) follows the mouse
	dropSlot bool
	floating bool

	bg      *canvas.Rectangle
	slot    *canvas.Rectangle // covers the button while it is the drop slot
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
	if t.selected || t.floating {
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

	// The drop slot is an empty box in the accent color, so the tab is seen
	// once, floating. Opaque, as it hides the label: the accent mixed into
	// the tab bar's color.
	th, v := t.Theme(), fyne.CurrentApp().Settings().ThemeVariant()
	t.slot = canvas.NewRectangle(mix(th.Color(theme.ColorNameMenuBackground, v), th.Color(theme.ColorNamePrimary, v), 0.35))
	t.slot.TopLeftCornerRadius = radius
	t.slot.TopRightCornerRadius = radius
	t.slot.Hidden = !t.dropSlot

	stack := container.NewStack(t.bg, container.NewPadded(inner), t.slot)
	if t.floating {
		// The accent outline lifts the dragged copy off the tab bar
		outline := canvas.NewRectangle(color.Transparent)
		outline.StrokeColor = th.Color(theme.ColorNamePrimary, v)
		outline.StrokeWidth = 2
		outline.TopLeftCornerRadius = radius
		outline.TopRightCornerRadius = radius
		stack.Add(outline)
	}
	return widget.NewSimpleRenderer(stack)
}

// setDropSlot shows the button as the place the dragged tab drops to, or as
// a tab again
func (t *tabButton) setDropSlot(on bool) {
	t.dropSlot = on
	if t.slot != nil {
		t.slot.Hidden = !on
		t.slot.Refresh()
	}
}

// mix is the color f of the way from a to b
func mix(a, b color.Color, f float32) color.Color {
	ar, ag, ab, _ := a.RGBA()
	br, bg, bb, _ := b.RGBA()
	m := func(x, y uint32) uint8 { return uint8((float32(x)*(1-f) + float32(y)*f) / 0x101) }
	return color.NRGBA{R: m(ar, br), G: m(ag, bg), B: m(ab, bb), A: 0xff}
}

func (t *tabButton) Tapped(*fyne.PointEvent) {
	if t.onTapped != nil {
		t.onTapped()
	}
}

// Dragged moves the tab along the bar, see fyne.Draggable
func (t *tabButton) Dragged(e *fyne.DragEvent) {
	if t.onDragged != nil {
		t.onDragged(t, e)
	}
}

// DragEnd puts the tab where it was dropped, see fyne.Draggable
func (t *tabButton) DragEnd() {
	if t.onDragEnd != nil {
		t.onDragEnd()
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

	// While a tab is dragged: the layer above the tab bar with its floating
	// copy, where in the button the mouse holds it, and the slides of the
	// buttons making room
	dragLayer *fyne.Container
	floating  *tabButton
	grab      float32
	slides    map[fyne.CanvasObject]*fyne.Animation

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
	t.dragLayer = container.NewWithoutLayout()
	t.tabScroll = container.NewHScroll(container.NewStack(t.tabBar, t.dragLayer))

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
		tab.onDragged = t.dragTab
		tab.onDragEnd = t.endTabDrag
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

// dragTab moves the dragged tab past a neighbor once the mouse passes the
// middle of that neighbor, its item moves along. The buttons are moved, not
// made again, as that would end the drag; endTabDrag makes them again.
func (t *tabs) dragTab(b *tabButton, e *fyne.DragEvent) {
	// Fyne gives the position from where the button is now, also after it
	// moved during the drag, so this is the mouse in the tab bar
	x := b.Position().X + e.Position.X

	// The tab buttons come first in the bar, the "+" after them
	n := len(t.items)
	i := -1
	for k, o := range t.tabBar.Objects[:n] {
		if o == b {
			i = k
			break
		}
	}
	if i < 0 {
		return
	}

	if t.floating == nil {
		t.startTabDrag(b, e)
	}

	// The middles are where the layout puts the buttons, not where they are
	// while sliding: a button sliding past the mouse would swap back
	for {
		at := t.tabSlots()
		middle := func(k int) float32 {
			return at[k] + t.tabBar.Objects[k].MinSize().Width/2
		}
		j := i
		if i+1 < n && x > middle(i+1) {
			j = i + 1
		} else if i > 0 && x < middle(i-1) {
			j = i - 1
		}
		if j == i {
			break
		}
		t.swapTabs(i, j)
		i = j
	}

	// The copy follows the mouse, kept over the tabs
	at := t.tabSlots()
	last := n - 1
	maxX := at[last] + t.tabBar.Objects[last].MinSize().Width - t.floating.Size().Width
	t.floating.Move(fyne.NewPos(max(0, min(x-t.grab, maxX)), b.Position().Y))
}

// startTabDrag starts the drag of b: its floating copy goes on the drag layer,
// b shows where the tab drops
func (t *tabs) startTabDrag(b *tabButton, e *fyne.DragEvent) {
	// The drag starts after the mouse moved a bit, this is where it went down
	t.grab = e.Position.X - e.Dragged.DX

	t.floating = newTabButton(b.text, b.icon, b.selected, nil, nil)
	t.floating.floating = true
	t.floating.Resize(b.Size())
	t.floating.Move(b.Position())
	t.dragLayer.Objects = []fyne.CanvasObject{t.floating}
	t.dragLayer.Refresh()

	b.setDropSlot(true)
	t.slides = map[fyne.CanvasObject]*fyne.Animation{}
}

// tabSlots is where the layout of the tab bar puts each tab button: their
// widths with the padding between them, as the HBox layout does
func (t *tabs) tabSlots() []float32 {
	n := len(t.items)
	at := make([]float32, n)
	x := float32(0)
	for k, o := range t.tabBar.Objects[:n] {
		at[k] = x
		x += o.MinSize().Width + theme.Padding()
	}
	return at
}

// swapTabs swaps the tabs at i and j, their buttons and items, the selected
// tab stays selected
func (t *tabs) swapTabs(i, j int) {
	objs := t.tabBar.Objects
	objs[i], objs[j] = objs[j], objs[i]
	t.items[i], t.items[j] = t.items[j], t.items[i]

	switch t.selectedIndex {
	case i:
		t.selectedIndex = j
	case j:
		t.selectedIndex = i
	}

	// Both buttons slide from where they are to their new places, the
	// layout would put them there at once
	at := t.tabSlots()
	for _, k := range []int{i, j} {
		o := objs[k]
		if a := t.slides[o]; a != nil {
			a.Stop()
		}
		to := fyne.NewPos(at[k], o.Position().Y)
		a := canvas.NewPositionAnimation(o.Position(), to, tabSlideTime, o.Move)
		a.Curve = fyne.AnimationEaseOut
		t.slides[o] = a
		a.Start()
	}
}

// tabSlideTime is how long a tab takes to slide to its new place in a drag
const tabSlideTime = 150 * time.Millisecond

// endTabDrag makes the buttons again for the new order of the tabs, their
// taps select by index
func (t *tabs) endTabDrag() {
	for _, a := range t.slides {
		a.Stop()
	}
	t.slides = nil
	t.floating = nil
	t.dragLayer.Objects = nil
	t.dragLayer.Refresh()
	t.Refresh()
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
