package index

import (
	"encoding/json"
	"fmt"
	"net/http"

	"sync"

	"github.com/asticode/go-astibob"
	"github.com/asticode/go-astilog"
	astiptr "github.com/asticode/go-astitools/ptr"
	"github.com/asticode/go-astiws"
	"github.com/gorilla/websocket"
	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
)

type worker struct {
	as   map[string]astibob.Ability
	ma   *sync.Mutex // Locks as
	name string
	ws   *astiws.Client
}

func newWorker(name string, ws *astiws.Client, as []astibob.Ability) (w *worker) {
	// Create
	w = &worker{
		as:   make(map[string]astibob.Ability),
		ma:   &sync.Mutex{},
		name: name,
		ws:   ws,
	}

	// Loop through abilities
	for _, a := range as {
		w.as[a.Name] = a
	}
	return
}

func (i *Index) handleWorkerWebsocket(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
	if err := i.ww.ServeHTTP(rw, r, func(c *astiws.Client) error {
		c.SetMessageHandler(i.handleWorkerMessage(c))
		return nil
	}); err != nil {
		if v, ok := errors.Cause(err).(*websocket.CloseError); !ok || v.Code != websocket.CloseNormalClosure {
			astilog.Error(errors.Wrap(err, "index: handling worker websocket failed"))
		}
		return
	}
}

func (i *Index) handleWorkerMessage(c *astiws.Client) astiws.MessageHandler {
	return func(p []byte) (err error) {
		// Log
		astilog.Debugf("index: handling worker message %s", p)

		// Unmarshal
		m := astibob.NewMessage()
		if err = json.Unmarshal(p, m); err != nil {
			err = errors.Wrap(err, "index: unmarshaling failed")
			return
		}

		// When the worker registers, we need to register the client
		if m.Name == astibob.CmdWorkerRegisterMessage && m.From.Name != nil {
			i.ww.RegisterClient(*m.From.Name, c)
		}

		// Dispatch
		i.d.Dispatch(m)
		return
	}
}

func (i *Index) sendMessageToWorkers(m *astibob.Message) (err error) {
	// Log
	astilog.Debugf("index: sending %s message to workers", m.Name)

	// Send message
	if err = sendMessage(m, i.ww); err != nil {
		err = errors.Wrap(err, "index: sending message failed")
		return
	}
	return
}

func (i *Index) addWorker(m *astibob.Message) (err error) {
	// Parse payload
	var as []astibob.Ability
	if as, err = astibob.ParseCmdWorkerRegisterPayload(m); err != nil {
		err = errors.Wrap(err, "index: parsing payload failed")
		return
	}

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
	w := newWorker(*m.From.Name, c, as)

	// Update pool
	i.mw.Lock()
	i.ws[w.name] = w
	i.mw.Unlock()

	// Handle disconnect
	c.SetListener(astiws.EventNameDisconnect, func(_ *astiws.Client, _ string, _ json.RawMessage) (err error) {
		// Create disconnected message
		var m *astibob.Message
		if m, err = astibob.NewEventWorkerDisconnectedMessage(from, &astibob.Identifier{Types: map[string]bool{
			astibob.UIIdentifierType:     true,
			astibob.WorkerIdentifierType: true,
		}}, w.name); err != nil {
			err = errors.Wrap(err, "astibob: creating disconnected message failed")
			return
		}

		// Dispatch
		i.d.Dispatch(m)
		return
	})

	// Log
	astilog.Infof("index: worker %s has registered", w.name)

	// Create registered message
	if m, err = astibob.NewEventWorkerRegisteredMessage(from, &astibob.Identifier{Types: map[string]bool{
		astibob.UIIdentifierType:     true,
		astibob.WorkerIdentifierType: true,
	}}, w.name, as); err != nil {
		err = errors.Wrap(err, "astibob: creating registered message failed")
		return
	}

	// Dispatch
	i.d.Dispatch(m)

	// Create welcome message
	if m, err = astibob.NewEventWorkerWelcomeMessage(from, &astibob.Identifier{
		Name: astiptr.Str(w.name),
		Type: astibob.WorkerIdentifierType,
	}); err != nil {
		err = errors.Wrap(err, "astibob: creating welcome message failed")
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
