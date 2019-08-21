package index

import (
	"encoding/json"
	"fmt"

	"github.com/asticode/go-astibob"
	"github.com/asticode/go-astilog"
	astiptr "github.com/asticode/go-astitools/ptr"
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
	// Name is empty
	if m.From.Name == nil || *m.From.Name == "" {
		err = errors.New("index: from name is empty")
		return
	}

	// Retrieve client
	c, ok := i.ww.Client(*m.From.Name)
	if !ok {
		err = fmt.Errorf("index: client %s doesn't exist", *m.From.Name)
		return
	}

	// Create worker
	w := newWorker(*m.From.Name, c)

	// Update pool
	i.mw.Lock()
	i.ws[w.name] = w
	i.mw.Unlock()

	// Handle disconnect
	c.SetListener(astiws.EventNameDisconnect, func(_ *astiws.Client, _ string, _ json.RawMessage) (err error) {
		// Create message
		var m *astibob.Message
		if m, err = astibob.NewEventWorkerDisconnectedMessage(from, nil, w.name); err != nil {
			err = errors.Wrap(err, "astibob: creating message failed")
			return
		}

		// Dispatch
		i.d.Dispatch(m)
		return
	})

	// Log
	astilog.Infof("index: worker %s has registered", w.name)

	// Create message
	if m, err = astibob.NewEventWorkerWelcomeMessage(from, &astibob.Identifier{
		Name: astiptr.Str(w.name),
		Type: astibob.WorkerIdentifierType,
	}); err != nil {
		err = errors.Wrap(err, "astibob: creating message failed")
		return
	}

	// Dispatch
	i.d.Dispatch(m)
	return
}

func (i *Index) delWorker(m *astibob.Message) (err error) {
	// Parse payload
	var name string
	if name, err = astibob.ParseEventWorkerDisconnectedPayload(m); err != nil {
		err = errors.Wrap(err, "index: parsing message payload failed")
		return
	}

	// Update pool
	i.mw.Lock()
	delete(i.ws, name)
	i.mw.Unlock()

	// Unregister client
	i.ww.UnregisterClient(name)

	// Log
	astilog.Infof("index: worker %s has disconnected", name)
	return
}

func (i *Index) sendWebsocketMessage(m *astibob.Message) (err error) {
	// No name
	if m.To == nil || m.To.Name == nil {
		err = errors.New("index: to name is empty")
		return
	}

	// Retrieve client from manager
	c, ok := i.ww.Client(*m.To.Name)
	if !ok {
		err = fmt.Errorf("index: client %s doesn't exist", *m.To.Name)
		return
	}

	// Write
	if err = c.WriteJSON(m); err != nil {
		err = errors.Wrap(err, "worker: writing JSON message failed")
		return
	}
	return
}
