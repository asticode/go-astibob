package worker

import (
	"github.com/asticode/go-astibob"
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
			From: astibob.NewRunnableIdentifier(l.Runnable, l.Worker),
			To:   w.workerIdentifier(),
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
		if m, err = astibob.NewListenablesRegisterMessage(
			*w.workerIdentifier(),
			astibob.NewWorkerIdentifier(worker),
			astibob.Listenables{
				Names:    p,
				Runnable: r,
			},
		); err != nil {
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

	// Parse payload
	var l astibob.Listenables
	if l, err = astibob.ParseListenablesRegisterPayload(m); err != nil {
		err = errors.Wrap(err, "worker: parsing register payload failed")
		return
	}

	// Lock
	w.mo.Lock()

	// Add runnable key
	if _, ok := w.ols[l.Runnable]; !ok {
		w.ols[l.Runnable] = make(map[string]map[string]bool)
	}

	// Add worker key
	if _, ok := w.ols[l.Runnable][worker]; !ok {
		w.ols[l.Runnable][worker] = make(map[string]bool)
	}

	// Add message name keys
	for _, n := range l.Names {
		w.ols[l.Runnable][worker][n] = true
	}

	// Unlock
	w.mo.Unlock()
	return
}
