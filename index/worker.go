package index

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"sync"

	"github.com/asticode/go-astibob"
	"github.com/asticode/go-astiws"
	"github.com/gorilla/websocket"
	"github.com/julienschmidt/httprouter"
)

type worker struct {
	addr string
	mr   *sync.Mutex // Locks rs
	name string
	rs   map[string]astibob.RunnableMessage
	ws   *astiws.Client
}

func newWorker(i astibob.Worker, ws *astiws.Client) (w *worker) {
	// Create
	w = &worker{
		addr: i.Addr,
		mr:   &sync.Mutex{},
		name: i.Name,
		rs:   make(map[string]astibob.RunnableMessage),
		ws:   ws,
	}

	// Loop through runnables
	for _, r := range i.Runnables {
		w.rs[r.Name] = r
	}
	return
}

func (w *worker) toMessage() (o astibob.Worker) {
	// Lock
	w.mr.Lock()
	defer w.mr.Unlock()

	// Create worker
	o = astibob.Worker{
		Addr: w.addr,
		Name: w.name,
	}

	// Get keys
	var ks []string
	for n := range w.rs {
		ks = append(ks, n)
	}

	// Sort keys
	sort.Strings(ks)

	// Loop through keys
	for _, k := range ks {
		// Append
		o.Runnables = append(o.Runnables, w.rs[k])
	}
	return
}

func (i *Index) handleWorkerWebsocket(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
	if err := i.ww.ServeHTTP(rw, r, func(c *astiws.Client) error {
		c.SetMessageHandler(i.handleWorkerMessage(c))
		return nil
	}); err != nil {
		var e *websocket.CloseError
		if ok := errors.As(err, &e); !ok || e.Code != websocket.CloseNormalClosure {
			i.l.Error(fmt.Errorf("index: handling worker websocket failed: %w", err))
		}
		return
	}
}

func (i *Index) handleWorkerMessage(c *astiws.Client) astiws.MessageHandler {
	return func(p []byte) (err error) {
		// Log
		i.l.Debugf("index: handling worker message %s", p)

		// Unmarshal
		m := astibob.NewMessage()
		if err = json.Unmarshal(p, m); err != nil {
			err = fmt.Errorf("index: unmarshaling failed: %w", err)
			return
		}

		// When the worker registers, we need to register the client
		if m.Name == astibob.WorkerRegisterMessage && m.From.Name != nil {
			i.ww.RegisterClient(*m.From.Name, c)
		}

		// Dispatch
		i.d.Dispatch(m)
		return
	}
}

func (i *Index) sendMessageToWorker(m *astibob.Message) (err error) {
	// Only send messages coming from uis or index as it would create an endless loop if we'd send messages coming
	// from workers
	if m.From.Type == astibob.RunnableIdentifierType || m.From.Type == astibob.WorkerIdentifierType {
		return
	}

	// Invalid to
	if m.To == nil {
		err = errors.New("index: invalid to")
		return
	}

	// Get names
	var names []string
	if worker := m.To.WorkerName(); worker != "" {
		names = append(names, worker)
	}

	// Send message
	if err = sendMessage(i.l, m, "worker", i.ww, names...); err != nil {
		err = fmt.Errorf("index: sending message failed: %w", err)
		return
	}
	return
}

func (i *Index) addWorker(m *astibob.Message) (err error) {
	// Parse payload
	var mw astibob.Worker
	if mw, err = astibob.ParseWorkerRegisterPayload(m); err != nil {
		err = fmt.Errorf("index: parsing payload failed: %w", err)
		return
	}

	// Retrieve client
	c, ok := i.ww.Client(mw.Name)
	if !ok {
		err = fmt.Errorf("index: client %s doesn't exist", mw.Name)
		return
	}

	// Create worker
	w := newWorker(mw, c)

	// Update pool
	i.mw.Lock()
	i.ws[w.name] = w
	i.mw.Unlock()

	// Handle disconnect
	c.SetListener(astiws.EventNameDisconnect, func(_ *astiws.Client, _ string, _ json.RawMessage) (err error) {
		// Create disconnected message
		var m *astibob.Message
		if m, err = astibob.NewWorkerDisconnectedMessage(
			*astibob.NewIndexIdentifier(),
			&astibob.Identifier{Types: map[string]bool{
				astibob.UIIdentifierType:     true,
				astibob.WorkerIdentifierType: true,
			}},
			w.name,
		); err != nil {
			err = fmt.Errorf("astibob: creating disconnected message failed: %w", err)
			return
		}

		// Dispatch
		i.d.Dispatch(m)
		return
	})

	// Log
	i.l.Infof("index: worker %s has registered", w.name)

	// Create welcome message
	if m, err = astibob.NewWorkerWelcomeMessage(
		*astibob.NewIndexIdentifier(),
		astibob.NewWorkerIdentifier(w.name),
		astibob.WelcomeWorker{
			UIMessageNames: i.uiMessageNames(),
			Workers:        i.workers(),
		},
	); err != nil {
		err = fmt.Errorf("index: creating welcome message failed: %w", err)
		return
	}

	// Dispatch
	i.d.Dispatch(m)

	// Create registered message
	if m, err = astibob.NewWorkerRegisteredMessage(
		*astibob.NewIndexIdentifier(),
		&astibob.Identifier{Types: map[string]bool{
			astibob.UIIdentifierType:     true,
			astibob.WorkerIdentifierType: true,
		}},
		mw,
	); err != nil {
		err = fmt.Errorf("index: creating registered message failed: %w", err)
		return
	}

	// Dispatch
	i.d.Dispatch(m)
	return
}

func (i *Index) delWorker(m *astibob.Message) (err error) {
	// Parse payload
	var name string
	if name, err = astibob.ParseWorkerDisconnectedPayload(m); err != nil {
		err = fmt.Errorf("index: parsing message payload failed: %w", err)
		return
	}

	// Update pool
	i.mw.Lock()
	delete(i.ws, name)
	i.mw.Unlock()

	// Unregister client
	i.ww.UnregisterClient(name)

	// Log
	i.l.Infof("index: worker %s has disconnected", name)
	return
}
