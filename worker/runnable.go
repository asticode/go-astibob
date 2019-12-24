package worker

import (
	"encoding/json"
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

		// Set root context
		r.Runnable.SetRootCtx(w.w.Context())

		// Set task func
		r.Runnable.SetTaskFunc(w.w.NewTask)

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

		// Create messages
		var ms []*astibob.Message
		if m.To != nil && m.To.Types != nil {
			if _, ok := m.To.Types[astibob.UIIdentifierType]; ok {
				cm := m.Clone()
				cm.To = &astibob.Identifier{Type: astibob.UIIdentifierType}
				ms = append(ms, cm)
			}
			if _, ok := m.To.Types[astibob.WorkerIdentifierType]; ok {
				ms = append(ms, w.cloneMessageForWorkers(name, m)...)
			}
		} else if m.To == nil || (m.To.Type == astibob.WorkerIdentifierType && m.To.Name == nil) {
			ms = append(ms, w.cloneMessageForWorkers(name, m)...)
		} else {
			ms = append(ms, m)
		}

		// Dispatch messages
		for _, m := range ms {
			w.d.Dispatch(m)
		}
	}
}

func (w *Worker) cloneMessageForWorkers(runnable string, i *astibob.Message) (ms []*astibob.Message) {
	// Lock
	w.mo.Lock()
	defer w.mo.Unlock()

	// Get listenables by workers
	wls, ok := w.ols[runnable]

	// No listenables for this runnable
	if !ok {
		return
	}

	// Loop through workers
	for n, ls := range wls {
		// No listenable for this worker
		if _, ok := ls[i.Name]; !ok {
			continue
		}

		// Append
		m := i.Clone()
		m.To = astibob.NewWorkerIdentifier(n)
		ms = append(ms, m)
	}
	return
}

func (w *Worker) startRunnableFromMessage(m *astibob.Message) (err error) {
	// Parse payload
	var name string
	if name, err = astibob.ParseRunnableStartPayload(m); err != nil {
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
	m := astibob.NewRunnableStartedMessage(*w.runnableIdentifier(name), &astibob.Identifier{Types: map[string]bool{
		astibob.UIIdentifierType:     true,
		astibob.WorkerIdentifierType: true,
	}})

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
			m = astibob.NewRunnableStoppedMessage(*w.runnableIdentifier(name), &astibob.Identifier{Types: map[string]bool{
				astibob.UIIdentifierType:     true,
				astibob.WorkerIdentifierType: true,
			}})
			astilog.Infof("worker: runnable %s has stopped", name)
		} else {
			m = astibob.NewRunnableCrashedMessage(*w.runnableIdentifier(name), &astibob.Identifier{Types: map[string]bool{
				astibob.UIIdentifierType:     true,
				astibob.WorkerIdentifierType: true,
			}})
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
	if name, err = astibob.ParseRunnableStopPayload(m); err != nil {
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

type MessageOptions struct {
	OnDone   OnDone
	Message  Message
	Runnable string
	Worker   string
}

type OnDone func(success bool) error

type Message struct {
	Name    string
	Payload interface{}
}

func (w *Worker) SendMessage(o MessageOptions) (err error) {
	// Create message
	m := astibob.NewMessage()

	// Default worker
	if o.Worker == "" {
		o.Worker = w.name
	}

	// Set basic info
	m.From = *w.workerIdentifier()
	m.To = astibob.NewRunnableIdentifier(o.Runnable, o.Worker)
	m.Name = o.Message.Name

	// Marshal payload
	if o.Message.Payload != nil {
		if m.Payload, err = json.Marshal(o.Message.Payload); err != nil {
			err = errors.Wrap(err, "worker: marshaling payload failed")
			return
		}
	}

	// On done
	if o.OnDone != nil {
		// Set id
		w.mi.Lock()
		w.id++
		m.ID = w.id
		w.mi.Unlock()

		// Add callback
		w.md.Lock()
		w.ds[m.ID] = o.OnDone
		w.md.Unlock()
	}

	// Dispatch
	w.d.Dispatch(m)
	return
}

func (w *Worker) doneMessage(m *astibob.Message) (err error) {
	// Parse payload
	var d astibob.RunnableDone
	if d, err = astibob.ParseRunnableDonePayload(m); err != nil {
		err = errors.Wrap(err, "worker: parsing runnable done payload failed")
		return
	}

	// Get callback
	w.md.Lock()
	c, ok := w.ds[d.ID]
	w.md.Unlock()

	// No callback
	if !ok {
		return
	}

	// On done
	if err = c(d.Success); err != nil {
		err = errors.Wrap(err, "worker: on done failed")
		return
	}
	return
}
