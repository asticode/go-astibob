package index

import (
	"encoding/json"
	"fmt"

	"github.com/asticode/go-astibob"
	"github.com/asticode/go-astilog"
	"github.com/asticode/go-astiws"
	"github.com/pkg/errors"
)

type worker struct {
	name string
	ws   *astiws.Client
}

func newWorker(name string, ws *astiws.Client) *worker {
	return &worker{
		name: name,
		ws:   ws,
	}
}

func (i *Index) addWorker(m *astibob.Message) (err error) {
	// Parse payload
	var name string
	if name, err = astibob.ParseWorkerRegisterCmdPayload(m); err != nil {
		err = errors.Wrap(err, "index: parsing message payload failed")
		return
	}

	// Name is empty
	if name == "" {
		err = errors.New("index: worker name is empty")
		return
	}

	// Retrieve client from state
	ck, ok := m.State[clientMessageStateKey]
	if !ok {
		err = errors.New("index: client key not found in state")
		return
	}

	// Retrieve client from manager
	c, ok := i.ww.Client(ck)
	if !ok {
		err = fmt.Errorf("index: client %s doesn't exist", ck)
		return
	}

	// Create worker
	w := newWorker(name, c)

	// Update pool
	i.mw.Lock()
	i.ws[name] = w
	i.mw.Unlock()

	// Handle disconnect
	c.SetListener(astiws.EventNameDisconnect, func(_ *astiws.Client, _ string, _ json.RawMessage) (err error) {
		// Create message
		var m *astibob.Message
		if m, err = astibob.NewEventWorkerDisconnectedMessage(from, name); err != nil {
			err = errors.Wrap(err, "astibob: creating message failed")
			return
		}

		// Dispatch
		i.d.Dispatch(m)

		// Unregister client
		i.ww.UnregisterClient(ck)
		return
	})

	// Log
	astilog.Infof("index: worker %s has registered", w.name)

	// Create message
	if m, err = astibob.NewEventWorkerWelcomeMessage(from); err != nil {
		err = errors.Wrap(err, "astibob: creating message failed")
		return
	}

	// Add client to state
	m.State[clientMessageStateKey] = ck

	// Dispatch
	i.d.Dispatch(m)
	return
}

func (i *Index) delWorker(m *astibob.Message) (err error) {
	// Parse payload
	var name string
	if name, err = astibob.ParseWorkerDisconnectedEventPayload(m); err != nil {
		err = errors.Wrap(err, "index: parsing message payload failed")
		return
	}

	// Update pool
	i.mw.Lock()
	delete(i.ws, name)
	i.mw.Unlock()

	// Log
	astilog.Infof("index: worker %s has disconnected", name)
	return
}

func (i *Index) sendWebsocketMessage(m *astibob.Message) (err error) {
	// Retrieve client from state
	ck, ok := m.State[clientMessageStateKey]
	if !ok {
		err = errors.New("index: client key not found in state")
		return
	}

	// Retrieve client from manager
	c, ok := i.ww.Client(ck)
	if !ok {
		err = fmt.Errorf("index: client %s doesn't exist", ck)
		return
	}

	// Write
	if err = c.WriteJSON(m); err != nil {
		err = errors.Wrap(err, "worker: writing JSON message failed")
		return
	}
	return
}
