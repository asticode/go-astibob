package astibrain

import (
	"context"
	"sync"

	"github.com/asticode/go-astilog"
	"github.com/asticode/go-astiws"
	"github.com/pkg/errors"
)

// Ability represents required methods of an ability
type Ability interface {
	Name() string
}

// Initializable represents an object that can be initialized.
type Initializable interface {
	Init() error
}

// Activable represents an object that can be activated.
type Activable interface {
	Activate(a bool)
}

// Runnable represents an object that can be run.
type Runnable interface {
	Run(ctx context.Context) error
}

// DispatchFunc represents a dispatch func
type DispatchFunc func(e Event)

// Dispatcher represents an object that can dispatch an event to the brain
type Dispatcher interface {
	SetDispatchFunc(DispatchFunc)
}

// WebsocketListener represents an object that can listen to a websocket
type WebsocketListener interface {
	WebsocketListeners() map[string]astiws.ListenerFunc
}

// AbilityOptions represents ability options
type AbilityOptions struct {
	AutoStart bool
}

// ability represents an ability.
type ability struct {
	a          Ability
	cancel     context.CancelFunc
	chanDone   chan error
	ctx        context.Context
	isOnUnsafe bool
	m          sync.Mutex // Locks attributes
	mr         sync.Mutex // Locks when ability is running
	name       string
	o          AbilityOptions
	ws         *websocket
}

// newAbility creates a new ability.
func newAbility(a Ability, ws *websocket, o AbilityOptions) *ability {
	return &ability{
		a:        a,
		chanDone: make(chan error),
		name:     a.Name(),
		o:        o,
		ws:       ws,
	}
}

// isOn returns whether the ability is on.
func (a *ability) isOn() bool {
	a.m.Lock()
	defer a.m.Unlock()
	return a.isOnUnsafe
}

// on switches the ability on.
// Its execution must not be blocking as it's used in a websocket call.
func (a *ability) on() {
	// Ability is already on
	if a.isOn() {
		return
	}

	// Log
	astilog.Debugf("astibrain: switching %s on", a.name)

	// Reset the context
	a.ctx, a.cancel = context.WithCancel(context.Background())

	// Switch on the activity
	if v, ok := a.a.(Activable); ok {
		a.onActivable(v)
	} else if v, ok := a.a.(Runnable); ok {
		a.onRunnable(v)
	}

	// Update ability status
	a.m.Lock()
	a.isOnUnsafe = true
	a.m.Unlock()

	// Lock running mutex
	a.mr.Lock()

	// Wait for the end of execution in a go routine
	go a.wait()

	// Log
	astilog.Infof("astibrain: %s have been switched on", a.name)

	// Dispatch websocket event
	a.ws.send(WebsocketEventNameAbilityStarted, a.name)
}

// onActivable switches the activable ability on.
func (a *ability) onActivable(v Activable) {
	// Activate
	v.Activate(true)

	// Listen to context in a goroutine
	go func() {
		<-a.ctx.Done()
		v.Activate(false)
		a.chanDone <- nil
	}()
}

// onRunnable switches the runnable ability on.
func (a *ability) onRunnable(v Runnable) {
	// Run in a goroutine
	go func() {
		a.chanDone <- v.Run(a.ctx)
	}()
}

// wait waits for the ability to stop or for the context to be done
func (a *ability) wait() {
	// Ability is not on
	if !a.isOn() {
		return
	}

	// Make sure the context is cancelled
	defer a.cancel()

	// Listen to chanDone
	if err := <-a.chanDone; a.ctx.Err() == nil {
		// Log
		astilog.Error(errors.Wrapf(err, "astibrain: %s crashed", a.name))

		// Dispatch websocket event
		a.ws.send(WebsocketEventNameAbilityCrashed, a.name)
	} else {
		// Log
		astilog.Infof("astibrain: %s have been switched off", a.name)

		// Dispatch websocket event
		a.ws.send(WebsocketEventNameAbilityStopped, a.name)
	}

	// Update ability status
	a.m.Lock()
	a.isOnUnsafe = false
	a.m.Unlock()

	// Unlock running mutex
	a.mr.Unlock()
	return
}

// off switches the ability off.
// Its execution must not be blocking as it's used in a websocket call.
func (a *ability) off() {
	// Ability is already off
	if !a.isOn() {
		return
	}

	// Log
	astilog.Debugf("astibrain: switching %s off", a.name)

	// Switch off
	a.cancel()

	// The rest is handled through the wait function
}
