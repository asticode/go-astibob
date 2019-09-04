package worker

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"

	"github.com/asticode/go-astibob"
	"github.com/asticode/go-astilog"
	astiworker "github.com/asticode/go-astitools/worker"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

// Register registers the worker to the index
func (w *Worker) RegisterToIndex() {
	// Create headers
	h := make(http.Header)
	if w.o.Index.Password != "" && w.o.Index.Username != "" {
		h.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(w.o.Index.Username+":"+w.o.Index.Password)))
	}

	// Dial
	w.w.Dial(astiworker.DialOptions{
		Addr:   "ws://" + w.o.Index.Addr + "/websockets/worker",
		Client: w.cw,
		Header: h,
		OnDial: w.sendRegister,
		OnReadError: func(err error) {
			if v, ok := errors.Cause(err).(*websocket.CloseError); ok && v.Code == websocket.CloseNormalClosure {
				astilog.Info("worker: worker has disconnected from index")
			} else {
				astilog.Error(errors.Wrap(err, "worker: reading websocket failed"))
			}
		},
	})
}

func (w *Worker) sendRegister() (err error) {
	// Get runnable keys
	w.mr.Lock()
	var ks []string
	for k := range w.rs {
		ks = append(ks, k)
	}

	// Sort
	sort.Strings(ks)

	// Loop through keys
	var rs []astibob.RunnableMessage
	for _, k := range ks {
		// Get runnable
		r := w.rs[k]

		// Create runnable message
		rm := astibob.RunnableMessage{
			Metadata: r.Metadata(),
			Status:   r.Status(),
		}

		// Add web homepage
		if _, ok := r.(astibob.Operatable); ok {
			rm.WebHomepage = fmt.Sprintf("/workers/%s/runnables/%s/web/index", url.QueryEscape(w.name), url.QueryEscape(r.Metadata().Name))
		}

		// Append runnable
		rs = append(rs, rm)
	}
	w.mr.Unlock()

	// Create register message
	var m *astibob.Message
	if m, err = astibob.NewWorkerRegisterMessage(*w.workerIdentifier(), &astibob.Identifier{
		Type: astibob.IndexIdentifierType,
	}, astibob.Worker{
		Addr:      "http://" + w.o.Server.Addr,
		Name:      w.name,
		Runnables: rs,
	}); err != nil {
		err = errors.Wrap(err, "worker: creating register message failed")
		return
	}

	// Dispatch
	w.d.Dispatch(m)
	return
}

func (w *Worker) finishRegistration(m *astibob.Message) (err error) {
	// Parse payload
	var ws []astibob.Worker
	if ws, err = astibob.ParseWorkerWelcomePayload(m); err != nil {
		err = errors.Wrap(err, "worker: parsing message payload failed")
		return
	}

	// Loop through workers
	for _, mw := range ws {
		// Add worker
		w.addWorker(mw)

		// Send register listenables
		if err = w.sendRegisterListenables(mw.Name); err != nil {
			err = errors.Wrapf(err, "worker: sending register listenables to worker %s failed", mw.Name)
			return
		}
	}

	// Log
	astilog.Info("worker: worker has registered to the index")
	return
}

func (w *Worker) handleIndexMessage(p []byte) (err error) {
	// Log
	astilog.Debugf("worker: handling index message %s", p)

	// Unmarshal
	m := astibob.NewMessage()
	if err = json.Unmarshal(p, m); err != nil {
		err = errors.Wrap(err, "worker: unmarshaling failed")
		return
	}

	// Dispatch
	w.d.Dispatch(m)
	return
}

func (w *Worker) sendMessageToIndex(m *astibob.Message) (err error) {
	// Only send message from current worker
	if m.From.WorkerName() != w.name {
		return
	}

	// Log
	astilog.Debugf("worker: sending %s message to index", m.Name)

	// Write
	if err = w.cw.WriteJSON(m); err != nil {
		err = errors.Wrap(err, "worker: writing JSON message failed")
		return
	}
	return
}

func (w *Worker) registerWorker(m *astibob.Message) (err error) {
	// Parse payload
	var mw astibob.Worker
	if mw, err = astibob.ParseWorkerRegisteredPayload(m); err != nil {
		err = errors.Wrap(err, "worker: parsing registered payload failed")
		return
	}

	// Do not process itself
	if mw.Name == w.name {
		return
	}

	// Add worker
	w.addWorker(mw)

	// Send register listenables
	if err = w.sendRegisterListenables(mw.Name); err != nil {
		err = errors.Wrapf(err, "worker: sending register listenables to worker %s failed", mw.Name)
		return
	}
	return
}

func (w *Worker) addWorker(m astibob.Worker) {
	// Lock
	w.mw.Lock()
	defer w.mw.Unlock()

	// Create worker
	nw := newWorker(m)

	// Update pool
	w.ws[nw.name] = nw
	return
}

func (w *Worker) unregisterWorker(m *astibob.Message) (err error) {
	// Parse payload
	var name string
	if name, err = astibob.ParseWorkerDisconnectedPayload(m); err != nil {
		err = errors.Wrap(err, "worker: parsing registered payload failed")
		return
	}

	// Do not process itself
	if name == w.name {
		return
	}

	// Delete worker
	w.delWorker(name)

	// Update listenables
	w.mo.Lock()
	for r := range w.ols {
		delete(w.ols[r], name)
	}
	w.mo.Unlock()
	return
}

func (w *Worker) delWorker(name string) {
	// Lock
	w.mw.Lock()
	defer w.mw.Unlock()

	// Update pool
	delete(w.ws, name)
	return
}
