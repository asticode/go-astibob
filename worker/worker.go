package worker

import (
	"net/http"
	"sync"

	"github.com/asticode/go-astibob"
	"github.com/asticode/go-astilog"
	astiptr "github.com/asticode/go-astitools/ptr"
	astiworker "github.com/asticode/go-astitools/worker"
	"github.com/asticode/go-astiws"
	"github.com/pkg/errors"
)

type Options struct {
	Index  astibob.ServerOptions `toml:"index"`
	Server astibob.ServerOptions `toml:"server"`
}

type Worker struct {
	ch   *http.Client
	cw   *astiws.Client
	d    *astibob.Dispatcher
	name string
	mr   *sync.Mutex // Locks rs
	mw   *sync.Mutex // Locks ws
	o    Options
	rs   map[string]astibob.Runnable
	w    *astiworker.Worker
	ws   map[string]*worker
}

// New creates a new worker
func New(name string, o Options) (w *Worker) {
	// Create worker
	w = &Worker{
		ch:   &http.Client{},
		cw:   astiws.NewClient(astiws.ClientConfiguration{}),
		name: name,
		mr:   &sync.Mutex{},
		mw:   &sync.Mutex{},
		o:    o,
		rs:   make(map[string]astibob.Runnable),
		w:    astiworker.NewWorker(),
		ws:   make(map[string]*worker),
	}

	// Create dispatcher
	w.d = astibob.NewDispatcher(w.w.NewTask)

	// Start dispatcher
	go w.d.Start(w.w.Context())

	// Add websocket message handler
	w.cw.SetMessageHandler(w.handleIndexMessage)

	// Add dispatcher handlers
	w.d.On(astibob.DispatchConditions{Name: astiptr.Str(astibob.CmdAbilityStartMessage)}, w.startAbility)
	w.d.On(astibob.DispatchConditions{Name: astiptr.Str(astibob.CmdAbilityStopMessage)}, w.stopAbility)
	w.d.On(astibob.DispatchConditions{Name: astiptr.Str(astibob.EventWorkerRegisteredMessage)}, w.registerWorker)
	w.d.On(astibob.DispatchConditions{Name: astiptr.Str(astibob.EventWorkerDisconnectedMessage)}, w.unregisterWorker)
	w.d.On(astibob.DispatchConditions{Name: astiptr.Str(astibob.EventWorkerWelcomeMessage)}, w.finishRegistration)
	w.d.On(astibob.DispatchConditions{
		From: w.from(),
		To:   &astibob.Identifier{Type: astibob.IndexIdentifierType},
	}, w.sendMessageToIndex)
	w.d.On(astibob.DispatchConditions{
		From: &astibob.Identifier{
			Type:   astibob.AbilityIdentifierType,
			Worker: astiptr.Str(w.name),
		},
		To: &astibob.Identifier{Type: astibob.UIIdentifierType},
	}, w.sendMessageToIndex)
	w.d.On(astibob.DispatchConditions{
		From: w.from(),
		To:   &astibob.Identifier{Type: astibob.AbilityIdentifierType},
	}, w.sendMessageToAbility)
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

// Close closes the worker properly
func (w *Worker) Close() error {
	// Stop dispatcher
	w.d.Stop()

	// Close client
	if w.cw != nil {
		if err := w.cw.Close(); err != nil {
			astilog.Error(errors.Wrap(err, "worker: closing client failed"))
		}
	}
	return nil
}

func (w *Worker) from() *astibob.Identifier {
	return &astibob.Identifier{
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

func (w *Worker) SendCmds(worker, ability string, cmds ...astibob.Cmd) (err error) {
	// Loop through cmds
	for _, cmd := range cmds {
		// Create message
		var m *astibob.Message
		if m, err = astibob.NewMessageFromCmd(*w.from(), &astibob.Identifier{
			Name:   astiptr.Str(ability),
			Type:   astibob.AbilityIdentifierType,
			Worker: astiptr.Str(worker),
		}, cmd); err != nil {
			err = errors.Wrap(err, "worker: creating message from cmd failed")
			return
		}

		// Dispatch
		w.d.Dispatch(m)
	}
	return
}

type worker struct {
	addr string
	as   map[string]astibob.Ability
	ma   *sync.Mutex // Locks as
	name string
}

func newWorker(i astibob.Worker) (w *worker) {
	// Create
	w = &worker{
		addr: i.Addr,
		as:   make(map[string]astibob.Ability),
		ma:   &sync.Mutex{},
		name: i.Name,
	}

	// Loop through abilities
	for _, a := range i.Abilities {
		w.as[a.Name] = a
	}
	return
}

func (w *worker) toMessage() (o astibob.Worker) {
	// Lock
	w.ma.Lock()
	defer w.ma.Unlock()

	// Create worker
	o = astibob.Worker{
		Addr: w.addr,
		Name: w.name,
	}

	// Loop through abilities
	for _, a := range w.as {
		// Append
		o.Abilities = append(o.Abilities, a)
	}
	return
}
