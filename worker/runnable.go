package worker

import (
	"fmt"

	"github.com/asticode/go-astibob"
	"github.com/asticode/go-astilog"
	"github.com/pkg/errors"
)

type Runnable struct {
	AutoStart bool
	Runnable  astibob.Runnable
}

func (w *Worker) RegisterRunnables(rs ...Runnable) {
	// Loop through runnables
	for _, r := range rs {
		// Add to pool
		w.mr.Lock()
		w.rs[r.Runnable.Metadata().Name] = r.Runnable
		w.mr.Unlock()

		// Set dispatch func
		r.Runnable.SetDispatchFunc(w.dispatchFunc(r.Runnable.Metadata().Name))

		// Add dispatch handlers
		w.d.On(astibob.DispatchConditions{To: w.runnableIdentifier(r.Runnable.Metadata().Name)}, r.Runnable.OnMessage)

		// Log
		astilog.Infof("worker: registered runnable %s", r.Runnable.Metadata().Name)

		// Auto start
		if r.AutoStart {
			// Start runnable
			if err := w.startRunnable(r.Runnable.Metadata().Name); err != nil {
				astilog.Error(errors.Wrapf(err, "worker: starting runnable %s failed", r.Runnable.Metadata().Name))
			}
		}
	}
}

func (w *Worker) dispatchFunc(name string) astibob.DispatchFunc {
	return func(m *astibob.Message) {
		// Set from
		m.From = *w.runnableIdentifier(name)

		// Dispatch message to itself
		// Example: audio input calibration on samples
		cm := m.Clone()
		cm.To = w.runnableIdentifier(name)
		w.d.Dispatch(cm)

		// Lock
		w.mo.Lock()
		defer w.mo.Unlock()

		// Get listenables by workers
		wls, ok := w.ols[name]

		// No listenables for this runnable
		if !ok {
			return
		}

		// Loop through workers
		for n, ls := range wls {
			// No listenable for this worker
			if _, ok := ls[m.Name]; !ok {
				continue
			}

			// Clone message
			cm := m.Clone()

			// Set to
			cm.To = astibob.NewWorkerIdentifier(n)

			// Dispatch
			w.d.Dispatch(cm)
		}
		return
	}
}

func (w *Worker) startRunnableFromMessage(m *astibob.Message) (err error) {
	// Parse payload
	var name string
	if name, err = astibob.ParseCmdRunnableStartPayload(m); err != nil {
		err = errors.Wrap(err, "index: parsing start payload failed")
		return
	}

	// Start runnable
	if err = w.startRunnable(name); err != nil {
		err = errors.Wrapf(err, "worker: starting runnable %s failed", name)
		return
	}
	return
}

func (w *Worker) startRunnable(name string) (err error) {
	// Fetch runnable
	w.mr.Lock()
	r, ok := w.rs[name]
	w.mr.Unlock()

	// No runnable
	if !ok {
		err = fmt.Errorf("worker: no %s runnable", name)
		return
	}

	// Check status
	if r.Status() == astibob.RunningStatus {
		err = fmt.Errorf("worker: runnable %s is already running", name)
		return
	}

	// Log
	astilog.Infof("worker: starting runnable %s", name)

	// Create started message
	m := astibob.NewEventRunnableStartedMessage(*w.runnableIdentifier(name), &astibob.Identifier{Type: astibob.UIIdentifierType})

	// Dispatch
	w.d.Dispatch(m)

	// Create new task
	t := w.w.NewTask()

	// Execute the rest in a goroutine
	go func() {
		// Make sure to let the worker know when the task is done
		defer t.Done()

		// Start the runnable
		if err := r.Start(w.w.Context()); err != nil && err != astibob.ErrContextCancelled {
			astilog.Error(errors.Wrapf(err, "worker: starting runnable %s failed", r.Metadata().Name))
		}

		// Create message
		if err == nil || err == astibob.ErrContextCancelled {
			m = astibob.NewEventRunnableStoppedMessage(*w.runnableIdentifier(name), &astibob.Identifier{Type: astibob.UIIdentifierType})
			astilog.Infof("worker: runnable %s has stopped", name)
		} else {
			m = astibob.NewEventRunnableCrashedMessage(*w.runnableIdentifier(name), &astibob.Identifier{Type: astibob.UIIdentifierType})
			astilog.Infof("worker: runnable %s has crashed", name)
		}

		// Dispatch
		w.d.Dispatch(m)
	}()
	return
}

func (w *Worker) stopRunnableFromMessage(m *astibob.Message) (err error) {
	// Parse payload
	var name string
	if name, err = astibob.ParseCmdRunnableStopPayload(m); err != nil {
		err = errors.Wrap(err, "index: parsing stop payload failed")
		return
	}

	// Stop runnable
	if err = w.stopRunnable(name); err != nil {
		err = errors.Wrapf(err, "worker: stopping runnable %s failed", name)
		return
	}
	return
}

func (w *Worker) stopRunnable(name string) (err error) {
	// Fetch runnable
	w.mr.Lock()
	r, ok := w.rs[name]
	w.mr.Unlock()

	// No runnable
	if !ok {
		err = fmt.Errorf("worker: no %s runnable", name)
		return
	}

	// Check status
	if r.Status() == astibob.StoppedStatus {
		err = fmt.Errorf("worker: runnable %s is already stopped", name)
		return
	}

	// Log
	astilog.Infof("worker: stopping runnable %s", r.Metadata().Name)

	// Stop runnable
	r.Stop()
	return
}

func (w *Worker) SendCmds(worker, runnable string, cmds ...astibob.Cmd) (err error) {
	// Loop through cmds
	for _, cmd := range cmds {
		// Create message
		var m *astibob.Message
		if m, err = astibob.NewMessageFromCmd(*w.workerIdentifier(), astibob.NewRunnableIdentifier(runnable, worker), cmd); err != nil {
			err = errors.Wrap(err, "worker: creating message from cmd failed")
			return
		}

		// Dispatch
		w.d.Dispatch(m)
	}
	return
}
