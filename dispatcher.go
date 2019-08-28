package astibob

import (
	"context"
	"fmt"
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

type Dispatcher struct {
	ctx context.Context
	cs  map[string]*astisync.Chan
	hs  []dispatcherHandler
	mc  *sync.Mutex // Locks cs
	mh  *sync.Mutex // Locks hs
	t   astiworker.TaskFunc
}

func NewDispatcher(ctx context.Context, t astiworker.TaskFunc) *Dispatcher {
	return &Dispatcher{
		ctx: ctx,
		cs:  make(map[string]*astisync.Chan),
		mc:  &sync.Mutex{},
		mh:  &sync.Mutex{},
		t:   t,
	}
}

func (d *Dispatcher) Close() {
	// Lock
	d.mc.Lock()
	defer d.mc.Unlock()

	// Stop chans
	for _, c := range d.cs {
		c.Stop()
	}
}

func (d *Dispatcher) Dispatch(m *Message) {
	// Lock
	d.mh.Lock()
	defer d.mh.Unlock()

	// Loop through handlers
	var c *astisync.Chan
	for _, h := range d.hs {
		// No match
		if !h.c.match(m) {
			continue
		}

		// No chan
		if c == nil {
			// Get message key
			k := d.key(m)

			// Lock
			d.mc.Lock()

			// Get chan
			var ok bool
			if c, ok = d.cs[k]; !ok {
				// Log
				astilog.Debugf("astibob: creating new dispatcher chan with key %s", k)

				// Create chan
				c = astisync.NewChan(astisync.ChanOptions{TaskFunc: d.t})
				d.cs[k] = c

				// Start chan
				go c.Start(d.ctx)
			}

			// Unlock
			d.mc.Unlock()
		}

		// Dispatch
		d.dispatch(c, m, h.h)
	}
}

// We don't want one dispatch to delay Cmds and Events, that's why we create specific chans for each of them. For now
// we're limiting this behavior to Cmds and Events for lack of examples of other cases.
func (d *Dispatcher) key(m *Message) string {
	// Message to runnable: Cmds
	if m.To != nil && m.To.Type == RunnableIdentifierType {
		return fmt.Sprintf("to.runnable.%s.%s", *m.To.Worker, *m.To.Name)
	}

	// Message from runnable: Events
	if m.From.Type == RunnableIdentifierType {
		return fmt.Sprintf("from.runnable.%s.%s", *m.From.Worker, *m.From.Name)
	}
	return "default"
}

func (d *Dispatcher) dispatch(c *astisync.Chan, m *Message, h MessageHandler) {
	// Add to chan
	c.Add(func() {
		// Handle message
		if err := h(m); err != nil {
			astilog.Error(errors.Wrap(err, "astibob: handling message failed"))
		}
	})
}

func (d *Dispatcher) On(c DispatchConditions, h MessageHandler) {
	d.mh.Lock()
	defer d.mh.Unlock()
	d.hs = append(d.hs, dispatcherHandler{
		c: c,
		h: h,
	})
}
