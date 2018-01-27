package astikeyboarding

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

// Press presses keys simultaneously
func (i *Interface) Press(keys ...string) *astibob.Cmd {
	return &astibob.Cmd{
		AbilityName: name,
		EventName:   websocketEventNameAction,
		Payload: PayloadAction{
			Action: actionPress,
			Keys:   keys,
		},
	}
}

// Type types a string with a delay
func (i *Interface) Type(s string) *astibob.Cmd {
	return &astibob.Cmd{
		AbilityName: name,
		EventName:   websocketEventNameAction,
		Payload: PayloadAction{
			Action: actionType,
			String: s,
		},
	}
}
