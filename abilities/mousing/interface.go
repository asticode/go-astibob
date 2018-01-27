package astimousing

import "github.com/asticode/go-astibob"

// Interface is the interface of the ability
type Interface struct{}

// NewInterface creates a new interface
func NewInterface() *Interface {
	return &Interface{}
}

// Name implements the astibob.Interface interface
func (i *Interface) Name() string {
	return name
}

// ClickLeft clicks the left button of the mouse
func (i *Interface) ClickLeft(double bool) *astibob.Cmd {
	return &astibob.Cmd{
		AbilityName: name,
		EventName:   websocketEventNameAction,
		Payload: PayloadAction{
			Action: actionClickLeft,
			Double: double,
		},
	}
}

// ClickMiddle clicks the middle button of the mouse
func (i *Interface) ClickMiddle(double bool) *astibob.Cmd {
	return &astibob.Cmd{
		AbilityName: name,
		EventName:   websocketEventNameAction,
		Payload: PayloadAction{
			Action: actionClickMiddle,
			Double: double,
		},
	}
}

// ClickRight clicks the right button of the mouse
func (i *Interface) ClickRight(double bool) *astibob.Cmd {
	return &astibob.Cmd{
		AbilityName: name,
		EventName:   websocketEventNameAction,
		Payload: PayloadAction{
			Action: actionClickRight,
			Double: double,
		},
	}
}

// Move moves the mouse
func (i *Interface) Move(x, y int) *astibob.Cmd {
	return &astibob.Cmd{
		AbilityName: name,
		EventName:   websocketEventNameAction,
		Payload: PayloadAction{
			Action: actionMove,
			X:      x,
			Y:      y,
		},
	}
}

// ScrollDown scrolls the mouse down
func (i *Interface) ScrollDown(x int) *astibob.Cmd {
	return &astibob.Cmd{
		AbilityName: name,
		EventName:   websocketEventNameAction,
		Payload: PayloadAction{
			Action: actionScrollDown,
			X:      x,
		},
	}
}

// ScrollUp scrolls the mouse up
func (i *Interface) ScrollUp(x int) *astibob.Cmd {
	return &astibob.Cmd{
		AbilityName: name,
		EventName:   websocketEventNameAction,
		Payload: PayloadAction{
			Action: actionScrollUp,
			X:      x,
		},
	}
}
