package speak

import (
	"encoding/json"

	"github.com/asticode/go-astibob"
	"github.com/pkg/errors"
)

// Message names
const (
	cmdSayMessage = "cmd.say"
)

func NewSayCmd(s string) astibob.Cmd {
	return astibob.Cmd{
		Name:    cmdSayMessage,
		Payload: s,
	}
}

func parseSayPayload(m *astibob.Message) (s string, err error) {
	if err = json.Unmarshal(m.Payload, &s); err != nil {
		err = errors.Wrap(err, "speak: unmarshaling failed")
		return
	}
	return
}
