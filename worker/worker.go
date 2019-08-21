package worker

import (
	"encoding/json"

	"github.com/asticode/go-astibob"
	astiptr "github.com/asticode/go-astitools/ptr"
	astiworker "github.com/asticode/go-astitools/worker"
	"github.com/asticode/go-astiws"
	"github.com/pkg/errors"
)

type Options struct {
	Index astibob.ServerOptions `toml:"index"`
}

type Worker struct {
	d    *astibob.Dispatcher
	name string
	o    Options
	w    *astiworker.Worker
	ws   *astiws.Client
}

// New creates a new worker
func New(name string, o Options) (w *Worker) {
	// Create worker
	w = &Worker{
		d:    astibob.NewDispatcher(),
		name: name,
		o:    o,
		w:    astiworker.NewWorker(),
		ws:   astiws.NewClient(astiws.ClientConfiguration{}),
	}

	// Add websocket handler
	w.ws.SetMessageHandler(w.handleIndexMessages)

	// Add dispatcher handlers
	w.d.On(astibob.DispatchConditions{Name: astiptr.Str(astibob.EventWorkerWelcomeMessage)}, w.finishRegistration)
	w.d.On(astibob.DispatchConditions{Name: astiptr.Str(astibob.CmdWorkerRegisterMessage)}, w.sendWebsocketMessage)
	return
}

// HandleSignals handles signals
func (w *Worker) HandleSignals() {
	w.w.HandleSignals()
}

// Wait waits for the index to be stopped
func (w *Worker) Wait() {
	w.w.Wait()
}

func (w *Worker) handleIndexMessages(p []byte) (err error) {
	// Unmarshal
	m := astibob.NewMessage()
	if err = json.Unmarshal(p, m); err != nil {
		err = errors.Wrap(err, "index: unmarshaling failed")
		return
	}

	// Dispatch
	w.d.Dispatch(m)
	return
}
