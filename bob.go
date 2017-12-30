package astibob

import (
	"context"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/asticode/go-astilog"
	"github.com/asticode/go-astiws"
	"github.com/pkg/errors"
)

// Bob is a polite AI with special abilities.
type Bob struct {
	a  map[string]*ability
	ma sync.Mutex // Locks a.
	o  Options
	ws *astiws.Manager
}

// Options represents Bob's options.
type Options struct {
	ResourcesDirectory string
	ServerAddr         string
	ServerPassword     string
	ServerTimeout      time.Duration
	ServerUsername     string
}

// New creates a new Bob.
func New(o Options) *Bob {
	return &Bob{
		a:  make(map[string]*ability),
		o:  o,
		ws: astiws.NewManager(4096),
	}
}

// Close implements the io.Closer interface.
func (b *Bob) Close() (err error) {
	// Close abilities
	b.abilities(func(a *ability) error {
		// Log
		astilog.Debugf("astibob: closing ability %s", a.name)

		// Switch the ability off
		a.off()

		// Wait for the toggle to be switched off indicating that the ability is really switched off
		for {
			if !a.t.isOn() {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}

		// Close
		if v, ok := a.a.(io.Closer); ok {
			if err := v.Close(); err != nil {
				astilog.Error(errors.Wrapf(err, "closing ability %s failed", a.name))
			}
		}
		return nil
	})

	// Close ws
	astilog.Debug("astibob: closing ws")
	if err = b.ws.Close(); err != nil {
		err = errors.Wrap(err, "astibob: closing ws failed")
		return
	}
	return
}

// ability returns a specific ability.
func (b *Bob) ability(key string) (a *ability, ok bool) {
	b.ma.Lock()
	defer b.ma.Unlock()
	a, ok = b.a[key]
	return
}

// abilities loops through abilities and execute a function on each of them.
// If an error is returned by the function, the loop is stopped.
func (b *Bob) abilities(fn func(a *ability) error) (err error) {
	b.ma.Lock()
	defer b.ma.Unlock()
	for _, a := range b.a {
		if err = fn(a); err != nil {
			return
		}
	}
	return
}

// Learn allows Bob to learn a new ability.
func (b *Bob) Learn(name string, a Ability, o AbilityOptions) *Bob {
	b.ma.Lock()
	defer b.ma.Unlock()
	wa := newAbility(name, a, o, b.ws)
	b.a[wa.key] = wa
	return b
}

// Run runs Bob.
func (b *Bob) Run(parentCtx context.Context) (err error) {
	// Loop through abilities
	if err = b.abilities(func(a *ability) (err error) {
		// Initialize
		if v, ok := a.a.(Initializer); ok {
			astilog.Debugf("astibob: initializing %s", a.name)
			if err = v.Init(); err != nil {
				err = errors.Wrapf(err, "astibob: initializing %s failed", a.name)
				return
			}

		}

		// Auto start
		if a.o.AutoStart {
			a.on()
		}
		return
	}); err != nil {
		err = errors.Wrap(err, "astibob: initializing abilities failed")
		return
	}

	// Create local ctx
	ctx, cancel := context.WithCancel(parentCtx)

	// Run server
	if err = b.runServer(ctx, cancel, b.o); err != nil {
		if err != http.ErrServerClosed {
			err = errors.Wrap(err, "astibob: running server failed")
		} else {
			err = nil
		}
		return
	}
	return
}

// dispatchWsEvent dispatches a websocket event.
func dispatchWsEvent(ws *astiws.Manager, name string, payload interface{}) {
	ws.Loop(func(k interface{}, c *astiws.Client) {
		// Write
		if err := c.Write(name, payload); err != nil {
			astilog.Error(errors.Wrapf(err, "astibob: writing to ws client %v failed", k))
			return
		}
	})
}
