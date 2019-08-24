package index

import (
	"fmt"

	"github.com/asticode/go-astibob"
	"github.com/asticode/go-astilog"
	"github.com/pkg/errors"
)

func (i *Index) sendMessageToAbility(m *astibob.Message) (err error) {
	// Log
	astilog.Debugf("index: sending %s message to ability", m.Name)

	// Check worker
	if m.To == nil || m.To.Worker == nil {
		err = errors.New("index: no to worker")
		return
	}

	// Retrieve client from manager
	c, ok := i.ww.Client(*m.To.Worker)
	if !ok {
		err = fmt.Errorf("index: client %s doesn't exist", *m.To.Worker)
		return
	}

	// Write
	if err = c.WriteJSON(m); err != nil {
		err = errors.Wrap(err, "worker: writing JSON message failed")
		return
	}
	return
}

func (i *Index) updateAbilityStatus(m *astibob.Message) (err error) {
	// Check worker
	if m.From.Worker == nil {
		err = errors.New("index: no from worker")
		return
	}

	// Get worker
	i.mw.Lock()
	w, ok := i.ws[*m.From.Worker]
	i.mw.Unlock()

	// No worker
	if !ok {
		err = fmt.Errorf("index: worker %s doesn't exist", *m.From.Worker)
		return
	}

	// Check ability
	if m.From.Name == nil {
		err = errors.New("index: no from name")
		return
	}

	// Get ability
	w.ma.Lock()
	a, ok := w.as[*m.From.Name]
	w.ma.Unlock()

	// No ability
	if !ok {
		err = fmt.Errorf("index: ability %s doesn't exist", *m.From.Name)
		return
	}

	// Update status
	if m.Name == astibob.EventAbilityStartedMessage {
		a.Status = astibob.RunningStatus
	} else {
		a.Status = astibob.StoppedStatus
	}

	// Update ability
	w.as[*m.From.Name] = a
	return
}
