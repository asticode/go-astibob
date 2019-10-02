package text_to_speech

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/asticode/go-astibob"
	"github.com/asticode/go-astibob/worker"
	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
)

// Message names
const (
	sayMessage = "text_to_speech.say"
)

type Speaker interface {
	Say(s string) error
}

type Runnable struct {
	*astibob.BaseOperatable
	*astibob.BaseRunnable
	m *sync.Mutex
	s Speaker
}

func NewRunnable(name string, s Speaker) (r *Runnable) {
	// Create runnable
	r = &Runnable{
		BaseOperatable: astibob.NewBaseOperatable(),
		BaseRunnable: astibob.NewBaseRunnable(astibob.BaseRunnableOptions{
			Metadata: astibob.Metadata{
				Description: "Converts text into spoken voice output using a form of speech synthesis",
				Name:        name,
			},
		}),
		m: &sync.Mutex{},
		s: s,
	}

	// Add routes
	r.AddRoute("/say", http.MethodPost, r.handleSay)
	return
}

func (r *Runnable) OnMessage(m *astibob.Message) (err error) {
	switch m.Name {
	case sayMessage:
		if err = r.onSay(m); err != nil {
			err = errors.Wrap(err, "text_to_speech: on say failed")
			return
		}
	}
	return
}

func NewSayMessage(s string) worker.Message {
	return worker.Message{
		Name:    sayMessage,
		Method:  http.MethodPost,
		Payload: s,
		URL:     "/say",
	}
}

func parseSayCmdPayload(m *astibob.Message) (s string, err error) {
	if err = json.Unmarshal(m.Payload, &s); err != nil {
		err = errors.Wrap(err, "text_to_speech: unmarshaling failed")
		return
	}
	return
}

func (r *Runnable) onSay(m *astibob.Message) (err error) {
	// Parse payload
	var s string
	if s, err = parseSayCmdPayload(m); err != nil {
		err = errors.Wrap(err, "text_to_speech: parsing payload failed")
		return
	}

	// Say
	if err = r.say(s); err != nil {
		err = errors.Wrap(err, "text_to_speech: saying failed")
		return
	}
	return
}

func (r *Runnable) handleSay(rw http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	// Unmarshal
	var s string
	err := json.NewDecoder(req.Body).Decode(&s)
	if err != nil {
		astibob.WriteHTTPError(rw, http.StatusInternalServerError, errors.Wrap(err, "text_to_speech: unmarshaling failed"))
		return
	}

	// Say
	if err = r.say(s); err != nil {
		astibob.WriteHTTPError(rw, http.StatusInternalServerError, errors.Wrap(err, "text_to_speech: saying failed"))
		return
	}
}

func (r *Runnable) say(s string) (err error) {
	// Check status
	if r.Status() != astibob.RunningStatus {
		return
	}

	// Lock
	r.m.Lock()
	defer r.m.Unlock()

	// Say
	if err = r.s.Say(s); err != nil {
		err = errors.Wrap(err, "text_to_speech: say failed")
		return
	}
	return
}
