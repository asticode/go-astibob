package worker

import (
	"sync"

	"github.com/asticode/go-astibob"
	astiptr "github.com/asticode/go-astitools/ptr"
	astiworker "github.com/asticode/go-astitools/worker"
	"github.com/asticode/go-astiws"
)

type Options struct {
	Index astibob.ServerOptions `toml:"index"`
}

type Worker struct {
	d    *astibob.Dispatcher
	name string
	mr   *sync.Mutex // Locks rs
	o    Options
	rs   map[string]astibob.Runnable
	w    *astiworker.Worker
	ws   *astiws.Client
}

// New creates a new worker
func New(name string, o Options) (w *Worker) {
	// Create worker
	w = &Worker{
		d:    astibob.NewDispatcher(),
		name: name,
		mr:   &sync.Mutex{},
		o:    o,
		rs:   make(map[string]astibob.Runnable),
		w:    astiworker.NewWorker(),
		ws:   astiws.NewClient(astiws.ClientConfiguration{}),
	}

	// Add websocket message handler
	w.ws.SetMessageHandler(w.handleIndexMessage)

	// Add dispatcher handlers
	w.d.On(astibob.DispatchConditions{Name: astiptr.Str(astibob.CmdAbilityStartMessage)}, w.startAbility)
	w.d.On(astibob.DispatchConditions{Name: astiptr.Str(astibob.CmdAbilityStopMessage)}, w.stopAbility)
	w.d.On(astibob.DispatchConditions{Name: astiptr.Str(astibob.EventWorkerWelcomeMessage)}, w.finishRegistration)
	w.d.On(astibob.DispatchConditions{
		From: &astibob.Identifier{
			Name: astiptr.Str(w.name),
			Type: astibob.WorkerIdentifierType,
		},
		To: &astibob.Identifier{Type: astibob.IndexIdentifierType},
	}, w.sendMessageToIndex)
	w.d.On(astibob.DispatchConditions{
		From: &astibob.Identifier{
			Type:   astibob.AbilityIdentifierType,
			Worker: astiptr.Str(w.name),
		},
		To: &astibob.Identifier{Type: astibob.UIIdentifierType},
	}, w.sendMessageToIndex)
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

func (w *Worker) from() astibob.Identifier {
	return astibob.Identifier{
		Name: astiptr.Str(w.name),
		Type: astibob.WorkerIdentifierType,
	}
}

func (w *Worker) fromAbility(name string) astibob.Identifier {
	return astibob.Identifier{
		Name:   astiptr.Str(name),
		Type:   astibob.AbilityIdentifierType,
		Worker: astiptr.Str(w.name),
	}
}
