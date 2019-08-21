package worker

import (
	"github.com/asticode/go-astibob"
	"github.com/asticode/go-astitools/ptr"
	"github.com/asticode/go-astitools/worker"
	"github.com/asticode/go-astiws"
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

	// Add websocket message handler
	w.ws.SetMessageHandler(w.handleIndexMessage)

	// Add dispatcher handlers
	w.d.On(astibob.DispatchConditions{Name: astiptr.Str(astibob.EventWorkerWelcomeMessage)}, w.finishRegistration)
	w.d.On(astibob.DispatchConditions{To: &astibob.Identifier{Type: astibob.IndexIdentifierType}}, w.sendMessageToIndex)
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
