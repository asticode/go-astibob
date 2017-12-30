package astibob

import (
	"context"
	"sync"
)

// toggle represents a toggle.
type toggle struct {
	cancel   context.CancelFunc
	chanDone chan error
	ctx      context.Context
	fn       toggleFunc
	m        sync.Mutex
}

// toggleFunc represents a toggle func
type toggleFunc func(ctx context.Context) error

// newToggle creates a new toggle.
func newToggle(fn toggleFunc) *toggle {
	return &toggle{
		chanDone: make(chan error),
		fn:       fn,
	}
}

// isOnUnsafe returns whether the toggle is on while making the assumption that the mutex is locked.
func (t *toggle) isOnUnsafe() bool {
	return t.ctx != nil && t.ctx.Err() == nil
}

// isOn returns whether the toggle is on.
func (t *toggle) isOn() bool {
	t.m.Lock()
	defer t.m.Unlock()
	return t.isOnUnsafe()
}

// on switches the toggle on.
func (t *toggle) on() {
	// Lock
	t.m.Lock()
	defer t.m.Unlock()

	// Toggle is already on
	if t.isOnUnsafe() {
		return
	}

	// Reset the context
	t.ctx, t.cancel = context.WithCancel(context.Background())

	// Execute in a go routine
	go func() {
		t.chanDone <- t.fn(t.ctx)
	}()
}

// off switches the toggle off.
func (t *toggle) off() {
	t.m.Lock()
	defer t.m.Unlock()
	t.cancel()
}

// wait waits for the toggle function to stop or for the context to be done.
func (t *toggle) wait() (err error) {
	// Toggle is not on
	if !t.isOn() {
		return
	}

	// Make sure the context is cancelled
	defer t.cancel()

	// Listen to channels
	select {
	case err = <-t.chanDone:
		if t.ctx.Err() != nil {
			err = t.ctx.Err()
		}
		return
	}
	return
}
