package terminal

import (
    "fmt"
    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/driver/desktop"
    "fyne.io/fyne/v2/widget"
    "fyne.io/fyne/v2/lang"
    "github.com/fyne-io/terminal"
    "yaterm/assert"
)

type Terminal struct {
    terminal.Terminal

    popUpMenu *widget.PopUpMenu
    //popMenu *fyne.MenuItem
    OnSplit func(int)
}

func New() *Terminal {
	t := &Terminal{}
	t.ExtendBaseWidget(t)

	menuItems := make([]*fyne.MenuItem, 2, 2)
	hsplitItem := fyne.NewMenuItemWithIcon(lang.L("Horionzontal Split"), assert.HSplitIconRes, func() {
        if t.OnSplit != nil {
            t.OnSplit(1)
        }
    })
	vsplitItem := fyne.NewMenuItemWithIcon(lang.L("Vertical Split"), assert.VSplitIconRes, func() {
        if t.OnSplit != nil {
            t.OnSplit(2)
        }
    })

    menuItems = append(menuItems, hsplitItem)
    menuItems = append(menuItems, vsplitItem)

	driver := fyne.CurrentApp().Driver()
	c := driver.CanvasForObject(t)
	t.popUpMenu = widget.NewPopUpMenu(fyne.NewMenu("", menuItems...), c)
	return t
}

func (t *Terminal) TappedSecondary(pe *fyne.PointEvent) {
	if c := fyne.CurrentApp().Driver().CanvasForObject(t); c != nil {
        menuItems := make([]*fyne.MenuItem, 2, 2)
        hsplitItem := fyne.NewMenuItemWithIcon(lang.L("Horionzontal Split"), assert.HSplitIconRes, func() {
            if t.OnSplit != nil {
                t.OnSplit(1)
            }
        })
        vsplitItem := fyne.NewMenuItemWithIcon(lang.L("Vertical Split"), assert.VSplitIconRes, func() {
            if t.OnSplit != nil {
                t.OnSplit(2)
            }
        })

        //menuItems = append(menuItems, hsplitItem)
        //menuItems = append(menuItems, vsplitItem)
        menuItems[0] = hsplitItem
        menuItems[1] = vsplitItem

        //driver := fyne.CurrentApp().Driver()
	    //c := driver.CanvasForObject(t)
	    popUpMenu := widget.NewPopUpMenu(fyne.NewMenu("", menuItems...), c)
	    popUpMenu.ShowAtPosition(pe.Position)
	}
    fmt.Println("Tapped")
}

func (t *Terminal) MouseDown(ev *desktop.MouseEvent) {
    if ev.Button == desktop.MouseButtonSecondary {
        if c := fyne.CurrentApp().Driver().CanvasForObject(t); c != nil {
            c.Focus(t)
        }
        return
    }


	if ev.Button == desktop.MouseButtonTertiary {
        ev.Button = desktop.MouseButtonSecondary
    }

    t.Terminal.MouseDown(ev)
}
