package index

import (
	"fmt"

	"github.com/asticode/go-astibob"
	"github.com/pkg/errors"
)

func (i *Index) updateRunnableStatus(m *astibob.Message) (err error) {
	// Check worker
	if m.From.Worker == nil {
		err = errors.New("index: no from worker")
		return
	}

	// Get worker
	i.mw.Lock()
	w, ok := i.ws[*m.From.Worker]
	i.mw.Unlock()

	// No worker
	if !ok {
		err = fmt.Errorf("index: worker %s doesn't exist", *m.From.Worker)
		return
	}

	// Check runnable
	if m.From.Name == nil {
		err = errors.New("index: no from name")
		return
	}

	// Get runnable
	w.mr.Lock()
	r, ok := w.rs[*m.From.Name]
	w.mr.Unlock()

	// No runnable
	if !ok {
		err = fmt.Errorf("index: runnable %s doesn't exist", *m.From.Name)
		return
	}

	// Update status
	if m.Name == astibob.EventRunnableStartedMessage {
		r.Status = astibob.RunningStatus
	} else {
		r.Status = astibob.StoppedStatus
	}

	// Update runnable
	w.rs[*m.From.Name] = r
	return
}
