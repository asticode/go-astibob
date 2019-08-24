package astibob

import (
	"sync"

	"github.com/asticode/go-astilog"
	"github.com/pkg/errors"
)

type MessageHandler func(m *Message) error

type dispatcherHandler struct {
	c DispatchConditions
	h MessageHandler
}

type Dispatcher struct {
	hs []dispatcherHandler
	m  *sync.Mutex
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{m: &sync.Mutex{}}
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
	d.m.Lock()
	defer d.m.Unlock()

	// Loop through handlers
	for _, h := range d.hs {
		// No match
		if !h.c.match(m) {
			continue
		}

		// Handle in a goroutine so that it's non blocking
		go func(h MessageHandler) {
			if err := h(m); err != nil {
				astilog.Error(errors.Wrap(err, "astibob: handling message failed"))
				return
			}
		}(h.h)
	}
}

func (d *Dispatcher) On(c DispatchConditions, h MessageHandler) {
	d.m.Lock()
	defer d.m.Unlock()
	d.hs = append(d.hs, dispatcherHandler{
		c: c,
		h: h,
	})
}
