package worker

import (
	"github.com/asticode/go-astibob"
	"github.com/asticode/go-astilog"
	"github.com/pkg/errors"
)

func (w *Worker) registerUI(m *astibob.Message) (err error) {
	// Parse payload
	var u astibob.UI
	if u, err = astibob.ParseUIRegisterPayload(m); err != nil {
		err = errors.Wrap(err, "index: parsing message payload failed")
		return
	}

	// Add ui
	w.addUI(u)
	return
}

func (w *Worker) addUI(u astibob.UI) {
	// Lock
	w.mu.Lock()
	defer w.mu.Unlock()

	// Loop through message names
	for _, n := range u.MessageNames {
		// Create message key
		if _, ok := w.us[n]; !ok {
			w.us[n] = make(map[string]bool)
		}

		// Add ui
		w.us[n][u.Name] = true
	}
}

func (w *Worker) unregisterUI(m *astibob.Message) (err error) {
	// Parse payload
	var name string
	if name, err = astibob.ParseUIDisconnectedPayload(m); err != nil {
		err = errors.Wrap(err, "index: parsing message payload failed")
		return
	}

	// Delete ui
	w.delUI(name)
	return
}

func (w *Worker) delUI(name string) {
	// Lock
	w.mu.Lock()
	defer w.mu.Unlock()

	// Loop through message names
	for n := range w.us {
		// Delete UI
		delete(w.us[n], name)

		// Delete message if no UI needs it anymore
		if len(w.us[n]) == 0 {
			delete(w.us, n)
		}
	}
}

func (w *Worker) sendMessageToUI(m *astibob.Message) (err error) {
	// Only send message from current worker
	if m.From.WorkerName() != w.name {
		return
	}

	// No UI requested this message
	w.mu.Lock()
	if _, ok := w.us[m.Name]; !ok {
		w.mu.Unlock()
		return
	}
	w.mu.Unlock()

	// Log
	astilog.Debugf("worker: sending %s message to ui", m.Name)

	// Write
	if err = w.cw.WriteJSON(m); err != nil {
		err = errors.Wrap(err, "worker: writing JSON message failed")
		return
	}
	return
}
