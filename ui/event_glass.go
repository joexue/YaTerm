package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type EventReceiver interface {
	// This should be implemented by buttons etc that wish to handle pointer interactions.
	Tapped(*fyne.PointEvent)

	// SecondaryTappable describes a [CanvasObject] that can be right-clicked or long-tapped.
	TappedSecondary(*fyne.PointEvent)

	// FocusGained is a hook called by the focus handling logic after this object gained the focus.
	FocusGained()
	// FocusLost is a hook called by the focus handling logic after this object lost the focus.
	FocusLost()

	// TypedRune is a hook called by the input handling logic on text input events if this object is focused.
	TypedRune(rune)

	// TypedKey is a hook called by the input handling logic on key events if this object is focused.
	TypedKey(*fyne.KeyEvent)
}

// Event Glass is a transparent glass covers on the top of all objects and capture all interested event and pass them to receiver
type EventGlass struct {
	widget.BaseWidget
	receiver EventReceiver
}

func NewEventGlass(r EventReceiver) *EventGlass {
	e := &EventGlass{
		receiver: r,
	}

	e.ExtendBaseWidget(e)

	return e
}

func (e *EventGlass) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewMax())
}

func (e *EventGlass) Tapped(pe *fyne.PointEvent) {
	if c := fyne.CurrentApp().Driver().CanvasForObject(e); c != nil {
		c.Focus(e)
	}
	e.receiver.Tapped(pe)
}

func (e *EventGlass) TappedSecondary(pe *fyne.PointEvent) {
	if c := fyne.CurrentApp().Driver().CanvasForObject(e); c != nil {
		c.Focus(e)
	}
	e.receiver.TappedSecondary(pe)
}

func (e *EventGlass) FocusGained() {
	e.receiver.FocusGained()
}

func (e *EventGlass) FocusLost() {
	e.receiver.FocusLost()
}

func (e *EventGlass) TypedRune(r rune) {
	e.receiver.TypedRune(r)
}

func (e *EventGlass) TypedKey(ke *fyne.KeyEvent) {
	e.receiver.TypedKey(ke)
}
