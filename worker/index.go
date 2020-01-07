package worker

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"

	"github.com/asticode/go-astibob"
	"github.com/asticode/go-astiws"
	"github.com/gorilla/websocket"
)

// Register registers the worker to the index
func (w *Worker) RegisterToIndex() {
	// Create headers
	h := make(http.Header)
	if w.o.Index.Password != "" && w.o.Index.Username != "" {
		h.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(w.o.Index.Username+":"+w.o.Index.Password)))
	}

	// Dial and read
	w.cw.DialAndRead(w.w, astiws.DialAndReadOptions{
		Addr:   "ws://" + w.o.Index.Addr + "/websockets/worker",
		Header: h,
		OnDial: w.sendRegister,
		OnReadError: func(err error) {
			var e *websocket.CloseError
			if ok := errors.As(err, &e); ok && e.Code == websocket.CloseNormalClosure {
				w.l.Info("worker: worker has disconnected from index")
			} else {
				w.l.Error(fmt.Errorf("worker: reading websocket failed: %w", err))
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
		if o, ok := r.(astibob.Operatable); ok && len(o.Templates()) > 0 {
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
		err = fmt.Errorf("worker: creating register message failed: %w", err)
		return
	}

	// Dispatch
	w.d.Dispatch(m)
	return
}

func (w *Worker) finishRegistration(m *astibob.Message) (err error) {
	// Parse payload
	var wl astibob.WelcomeWorker
	if wl, err = astibob.ParseWorkerWelcomePayload(m); err != nil {
		err = fmt.Errorf("worker: parsing message payload failed: %w", err)
		return
	}

	// Reset and add ui message names
	w.mu.Lock()
	w.us = make(map[string]bool)
	for _, n := range wl.UIMessageNames {
		w.us[n] = true
	}
	w.mu.Unlock()

	// Index workers
	iws := make(map[string]bool)
	for _, w := range wl.Workers {
		iws[w.Name] = true
	}

	// Remove useless other workers listenables
	w.mo.Lock()
	for r, ws := range w.ols {
		for n := range ws {
			if _, ok := iws[n]; !ok {
				delete(w.ols[r], n)
			}
		}
	}
	w.mo.Unlock()

	// Reset workers
	w.resetWorkers()

	// Loop through workers
	for _, mw := range wl.Workers {
		// Add worker
		w.addWorker(mw)

		// Send register listenables
		if err = w.sendRegisterListenables(mw.Name); err != nil {
			err = fmt.Errorf("worker: sending register listenables to worker %s failed: %w", mw.Name, err)
			return
		}
	}

	// Log
	w.l.Info("worker: worker has registered to the index")
	return
}

func (w *Worker) handleIndexMessage(p []byte) (err error) {
	// Log
	w.l.Debugf("worker: handling index message %s", p)

	// Unmarshal
	m := astibob.NewMessage()
	if err = json.Unmarshal(p, m); err != nil {
		err = fmt.Errorf("worker: unmarshaling failed: %w", err)
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
	w.l.Debugf("worker: sending %s message to index", m.Name)

	// Write
	if err = w.cw.WriteJSON(m); err != nil {
		err = fmt.Errorf("worker: writing JSON message failed: %w", err)
		return
	}
	return
}

func (w *Worker) registerWorker(m *astibob.Message) (err error) {
	// Parse payload
	var mw astibob.Worker
	if mw, err = astibob.ParseWorkerRegisteredPayload(m); err != nil {
		err = fmt.Errorf("worker: parsing registered payload failed: %w", err)
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
		err = fmt.Errorf("worker: sending register listenables to worker %s failed: %w", mw.Name, err)
		return
	}
	return
}

func (w *Worker) resetWorkers() {
	// Lock
	w.mw.Lock()
	defer w.mw.Unlock()

	// Reset
	w.ws = make(map[string]*worker)
}

func (w *Worker) addWorker(m astibob.Worker) {
	// Lock
	w.mw.Lock()
	defer w.mw.Unlock()

	// Create worker
	nw := newWorker(m)

	// Update pool
	w.ws[nw.name] = nw
}

func (w *Worker) unregisterWorker(m *astibob.Message) (err error) {
	// Parse payload
	var name string
	if name, err = astibob.ParseWorkerDisconnectedPayload(m); err != nil {
		err = fmt.Errorf("worker: parsing registered payload failed: %w", err)
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
}
