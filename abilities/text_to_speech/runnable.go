package text_to_speech

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/asticode/go-astibob"
	"github.com/asticode/go-astibob/worker"
	"github.com/asticode/go-astikit"
)

// Message names
const (
	sayMessage = "text_to_speech.say"
)

type Speaker interface {
	Say(s string) error
}

type Runnable struct {
	*astibob.BaseRunnable
	c *astikit.Chan
	l astikit.SeverityLogger
	s Speaker
}

func NewRunnable(name string, s Speaker, l astikit.StdLogger) (r *Runnable) {
	// Create runnable
	r = &Runnable{
		c: astikit.NewChan(astikit.ChanOptions{}),
		l: astikit.AdaptStdLogger(l),
		s: s,
	}

	// Set base runnable
	r.BaseRunnable = astibob.NewBaseRunnable(astibob.BaseRunnableOptions{
		Logger: l,
		Metadata: astibob.Metadata{
			Description: "Converts text into spoken voice output using a form of speech synthesis",
			Name:        name,
		},
		OnMessage: r.onMessage,
		OnStart:   r.onStart,
	})
	return
}

func (r *Runnable) onStart(ctx context.Context) (err error) {
	// Reset chan
	r.c.Reset()

	// Start chan
	r.c.Start(ctx)

	// Stop chan
	r.c.Stop()
	return
}

func (r *Runnable) onMessage(m *astibob.Message) (err error) {
	switch m.Name {
	case sayMessage:
		if err = r.onSay(m); err != nil {
			err = fmt.Errorf("text_to_speech: on say failed: %w", err)
			return
		}
	}
	return
}

func NewSayMessage(s string) worker.Message {
	return worker.Message{
		Name:    sayMessage,
		Payload: s,
	}
}

func parseSayPayload(m *astibob.Message) (s string, err error) {
	if err = json.Unmarshal(m.Payload, &s); err != nil {
		err = fmt.Errorf("text_to_speech: unmarshaling failed: %w", err)
		return
	}
	return
}

func (r *Runnable) onSay(m *astibob.Message) (err error) {
	// Check status
	if r.Status() != astibob.RunningStatus {
		return
	}

	// Parse payload
	var s string
	if s, err = parseSayPayload(m); err != nil {
		err = fmt.Errorf("text_to_speech: parsing payload failed: %w", err)
		return
	}

	// Make sure this is non blocking but still executed in FIFO order
	r.c.Add(r.sayFunc(s))
	return
}

func (r *Runnable) sayFunc(s string) func() {
	return func() {
		// Say
		if err := r.s.Say(s); err != nil {
			r.l.Error(fmt.Errorf("text_to_speech: say failed: %w", err))
			return
		}
	}
}
