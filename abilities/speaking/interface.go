package astispeaking

import "github.com/asticode/go-astibob"

// Interface is the interface of the ability
type Interface struct{}

// NewInterface creates a new interface
func NewInterface() *Interface {
	return &Interface{}
}

// Say creates a say cmd
func (i *Interface) Say(s string) *astibob.Cmd {
	return &astibob.Cmd{
		AbilityName: Name,
		EventName:   websocketEventNameSay,
		Payload:     s,
	}
}
