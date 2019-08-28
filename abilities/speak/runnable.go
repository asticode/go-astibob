package speak

import (
	"encoding/json"
	"sync"

	"github.com/asticode/go-astibob"
	"github.com/pkg/errors"
)

// Message names
const (
	cmdSayMessage = "cmd.say"
)

type Speaker interface {
	Say(s string) error
}

type runnable struct {
	*astibob.BaseRunnable
	m *sync.Mutex
	s Speaker
}

func NewRunnable(name string, s Speaker) astibob.Runnable {
	return &runnable{
		BaseRunnable: astibob.NewBaseRunnable(astibob.BaseRunnableOptions{
			Metadata: astibob.Metadata{
				Description: "Says words to your audio output using speech synthesis",
				Name:        name,
			},
		}),
		m: &sync.Mutex{},
		s: s,
	}
}

func (r *runnable) OnMessage(m *astibob.Message) (err error) {
	switch m.Name {
	case cmdSayMessage:
		if err = r.onSay(m); err != nil {
			err = errors.Wrap(err, "speak: on say failed")
			return
		}
	}
	return
}

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

func (r *runnable) onSay(m *astibob.Message) (err error) {
	// Check status
	if r.Status() != astibob.RunningStatus {
		return
	}

	// Parse payload
	var s string
	if s, err = parseSayPayload(m); err != nil {
		err = errors.Wrap(err, "speak: parsing payload failed")
		return
	}

	// Lock
	r.m.Lock()
	defer r.m.Unlock()

	// Say
	if err = r.s.Say(s); err != nil {
		err = errors.Wrap(err, "speak: say failed")
		return
	}
	return
}
