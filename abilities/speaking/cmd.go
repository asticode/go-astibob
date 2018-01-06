package astispeaking

import "github.com/asticode/go-astibob"

// CmdSay creates a say cmd
func CmdSay(i string) *astibob.Cmd {
	return &astibob.Cmd{
		AbilityName: Name,
		EventName:   websocketEventNameSay,
		Payload:     i,
	}
}
