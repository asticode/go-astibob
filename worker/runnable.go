package worker

import (
	"fmt"

	"github.com/asticode/go-astibob"
	"github.com/asticode/go-astilog"
	astiptr "github.com/asticode/go-astitools/ptr"
	"github.com/pkg/errors"
)

func (w *Worker) RegisterRunnables(rs ...astibob.Runnable) {
	// Lock
	w.mr.Lock()
	defer w.mr.Unlock()

	// Loop through runnables
	for _, r := range rs {
		// Add to pool
		w.rs[r.Metadata().Name] = r

		// Add dispatch handlers
		w.d.On(astibob.DispatchConditions{To: &astibob.Identifier{
			Name:   astiptr.Str(r.Metadata().Name),
			Type:   astibob.AbilityIdentifierType,
			Worker: astiptr.Str(w.name),
		}}, r.OnMessage)

		// Log
		astilog.Infof("worker: registered runnable %s", r.Metadata().Name)
	}
}

func (w *Worker) startAbility(m *astibob.Message) (err error) {
	// Parse payload
	var name string
	if name, err = astibob.ParseCmdAbilityStartPayload(m); err != nil {
		err = errors.Wrap(err, "worker: parsing payload failed")
		return
	}

	// Fetch runnable
	w.mr.Lock()
	r, ok := w.rs[name]
	w.mr.Unlock()

	// No runnable
	if !ok {
		err = fmt.Errorf("worker: no %s runnable", name)
		return
	}

	// Log
	astilog.Infof("worker: starting runnable %s", name)

	// Create started message
	if m, err = astibob.NewEventAbilityStartedMessage(w.from(), &astibob.Identifier{Type: astibob.IndexIdentifierType}, name); err != nil {
		err = errors.Wrap(err, "worker: creating started message failed")
		return
	}

	// Dispatch
	w.d.Dispatch(m)

	// Create new task
	t := w.w.NewTask()

	// Execute the rest in a goroutine
	go func() {
		// Make sure to let the worker know when the task is done
		defer t.Done()

		// Start the runnable
		if err := r.Start(w.w.Context()); err != nil {
			astilog.Error(errors.Wrapf(err, "worker: starting runnable %s failed", r.Metadata().Name))
		}

		// Create message
		if err == nil {
			m, err = astibob.NewEventAbilityStoppedMessage(w.from(), &astibob.Identifier{Type: astibob.IndexIdentifierType}, name)
			astilog.Infof("worker: runnable %s has stopped", name)
		} else {
			m, err = astibob.NewEventAbilityCrashedMessage(w.from(), &astibob.Identifier{Type: astibob.IndexIdentifierType}, name)
			astilog.Infof("worker: runnable %s has crashed", name)
		}
		if err != nil {
			astilog.Error(errors.Wrap(err, "worker: creating message failed"))
		}

		// Dispatch
		w.d.Dispatch(m)
	}()
	return
}

func (w *Worker) stopAbility(m *astibob.Message) (err error) {
	// Parse payload
	var name string
	if name, err = astibob.ParseCmdAbilityStopPayload(m); err != nil {
		err = errors.Wrap(err, "worker: parsing payload failed")
		return
	}

	// Fetch runnable
	w.mr.Lock()
	r, ok := w.rs[name]
	w.mr.Unlock()

	// No runnable
	if !ok {
		err = fmt.Errorf("worker: no %s runnable", name)
		return
	}

	// Log
	astilog.Infof("worker: stopping runnable %s", r.Metadata().Name)

	// Stop runnable
	r.Stop()
	return
}
