package speak

import (
	"sync"

	"github.com/asticode/go-astibob"
	"github.com/pkg/errors"
)

type Speaker interface {
	Say(s string) error
}

func NewRunnable(name string, s Speaker) astibob.Runnable {
	r := newRunnable(s)
	return astibob.NewRunnable(astibob.RunnableOptions{
		Metadata: astibob.Metadata{
			Description: "Says words to your audio output using speech synthesis",
			Name:        name,
		},
		OnMessage: r.onMessage,
	})
}

type runnable struct {
	m *sync.Mutex
	s Speaker
}

func newRunnable(s Speaker) *runnable {
	return &runnable{
		m: &sync.Mutex{},
		s: s,
	}
}

func (r *runnable) onMessage(m *astibob.Message) (err error) {
	switch m.Name {
	case cmdSayMessage:
		if err = r.onSay(m); err != nil {
			err = errors.Wrap(err, "speak: on say failed")
			return
		}
	}
	return
}

func (r *runnable) onSay(m *astibob.Message) (err error) {
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
