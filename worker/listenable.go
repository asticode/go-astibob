package worker

import (
	"github.com/asticode/go-astibob"
	astiptr "github.com/asticode/go-astitools/ptr"
	"github.com/pkg/errors"
)

type Listenable struct {
	Listenable astibob.Listenable
	Runnable   string
	Worker     string
}

func (w *Worker) RegisterListenables(ls ...Listenable) {
	// Loop through listenables
	for _, l := range ls {
		// Add dispatcher handler
		w.d.On(astibob.DispatchConditions{
			From: &astibob.Identifier{
				Name:   astiptr.Str(l.Runnable),
				Type:   astibob.RunnableIdentifierType,
				Worker: astiptr.Str(l.Worker),
			},
			To: &astibob.Identifier{
				Name: astiptr.Str(w.name),
				Type: astibob.WorkerIdentifierType,
			},
		}, l.Listenable.OnMessage)

		// Add message names
		ns := l.Listenable.MessageNames()
		if len(ns) > 0 {
			// Lock
			w.ml.Lock()

			// Add worker key
			if _, ok := w.ls[l.Worker]; !ok {
				w.ls[l.Worker] = make(map[string]map[string]bool)
			}

			// Add runnable key
			if _, ok := w.ls[l.Worker][l.Runnable]; !ok {
				w.ls[l.Worker][l.Runnable] = make(map[string]bool)
			}

			// Add message name keys
			for _, n := range ns {
				w.ls[l.Worker][l.Runnable][n] = true
			}

			// Unlock
			w.ml.Unlock()
		}
	}
}

func (w *Worker) sendRegisterListenables(worker string) (err error) {
	// Lock
	w.ml.Lock()
	defer w.ml.Unlock()

	// No listenables for this worker
	if _, ok := w.ls[worker]; !ok {
		return
	}

	// Loop through runnables
	for r, ns := range w.ls[worker] {
		// Loop through message names
		var p []string
		for n := range ns {
			p = append(p, n)
		}

		// Create message
		var m *astibob.Message
		if m, err = astibob.NewCmdListenablesRegisterMessage(*w.from(), &astibob.Identifier{
			Name:   astiptr.Str(r),
			Type:   astibob.RunnableIdentifierType,
			Worker: astiptr.Str(worker),
		}, p); err != nil {
			err = errors.Wrap(err, "worker: creating register message failed")
			return
		}

		// Dispatch
		w.d.Dispatch(m)
	}
	return
}

func (w *Worker) registerListenables(m *astibob.Message) (err error) {
	// Get worker name
	worker := m.From.WorkerName()

	// Invalid worker name
	if worker == "" {
		err = errors.New("worker: invalid worker name")
		return
	}

	// Only register listenables from other workers
	if worker == w.name {
		return
	}

	// To is invalid
	if m.To == nil {
		err = errors.New("worker: to is invalid")
		return
	}

	// Parse payload
	var ns []string
	if ns, err = astibob.ParseCmdListenablesRegisterPayload(m); err != nil {
		err = errors.Wrap(err, "worker: parsing register payload failed")
		return
	}

	// Lock
	w.mo.Lock()

	// Add runnable key
	if _, ok := w.ols[*m.To.Name]; !ok {
		w.ols[*m.To.Name] = make(map[string]map[string]bool)
	}

	// Add worker key
	if _, ok := w.ols[*m.To.Name][worker]; !ok {
		w.ols[*m.To.Name][worker] = make(map[string]bool)
	}

	// Add message name keys
	for _, n := range ns {
		w.ols[*m.To.Name][worker][n] = true
	}

	// Unlock
	w.mo.Unlock()
	return
}
