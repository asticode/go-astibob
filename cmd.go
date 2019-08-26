package astibob

import (
	"encoding/json"

	"github.com/pkg/errors"
)

type Cmd struct {
	Name    string      `json:"name"`
	Payload interface{} `json:"payload"`
}

func NewMessageFromCmd(from Identifier, to *Identifier, cmd Cmd) (m *Message, err error) {
	// Create message
	m = newMessage(from, to, cmd.Name)

	// Marshal payload
	if cmd.Payload != nil {
		if m.Payload, err = json.Marshal(cmd.Payload); err != nil {
			err = errors.Wrap(err, "astibob: marshaling payload failed")
			return
		}
	}
	return
}
