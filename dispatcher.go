package astibob

import (
	"context"
	"sync"

	"github.com/asticode/go-astilog"
	astisync "github.com/asticode/go-astitools/sync"
	astiworker "github.com/asticode/go-astitools/worker"
	"github.com/pkg/errors"
)

type MessageHandler func(m *Message) error

type dispatcherHandler struct {
	c DispatchConditions
	h MessageHandler
}

type Dispatcher struct {
	c  *astisync.Chan
	hs []dispatcherHandler
	mh *sync.Mutex // Locks hs
}

func NewDispatcher(t astiworker.TaskFunc) *Dispatcher {
	return &Dispatcher{
		c:  astisync.NewChan(astisync.ChanOptions{TaskFunc: t}),
		mh: &sync.Mutex{},
	}
}

type DispatchConditions struct {
	From  *Identifier
	Name  *string
	Names map[string]bool
	To    *Identifier
}

func (c DispatchConditions) match(m *Message) bool {
	// Check from
	if c.From != nil && !c.From.match(m.From) {
		return false
	}

	// Check name
	if c.Names != nil {
		if _, ok := c.Names[m.Name]; !ok {
			return false
		}
	} else if c.Name != nil && *c.Name != m.Name {
		return false
	}

	// Check to
	if c.To != nil {
		// Check message
		if m.To == nil {
			return false
		}

		// Check identifier
		if !c.To.match(*m.To) {
			return false
		}
	}
	return true
}

func (d *Dispatcher) Dispatch(m *Message) {
	// Lock
	d.mh.Lock()
	defer d.mh.Unlock()

	// Loop through handlers
	for _, h := range d.hs {
		// No match
		if !h.c.match(m) {
			continue
		}

		// Dispatch
		d.dispatch(m, h.h)
	}
}

func (d *Dispatcher) dispatch(m *Message, h MessageHandler) {
	// Add to chan
	d.c.Add(func() {
		// Handle message
		if err := h(m); err != nil {
			astilog.Error(errors.Wrap(err, "astibob: handling message failed"))
		}
	})
}

func (d *Dispatcher) Start(ctx context.Context) {
	d.c.Start(ctx)
}

func (d *Dispatcher) Stop() {
	d.c.Stop()
}

func (d *Dispatcher) On(c DispatchConditions, h MessageHandler) {
	d.mh.Lock()
	defer d.mh.Unlock()
	d.hs = append(d.hs, dispatcherHandler{
		c: c,
		h: h,
	})
}
