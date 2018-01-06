package astibrain

import (
	"context"
	"io"
	"os"

	"github.com/asticode/go-astilog"
	"github.com/pkg/errors"
)

// Brain is an object handling one or more abilities
type Brain struct {
	abilities *abilities
	cancel    context.CancelFunc
	ctx       context.Context
	o         Options
	ws        *websocket
}

// Options are brain options
type Options struct {
	Name      string           `toml:"name"`
	Websocket WebsocketOptions `toml:"websocket"`
}

// New creates a new brain
func New(o Options) (b *Brain) {
	// Create brain
	b = &Brain{
		abilities: newAbilities(),
		o:         o,
	}

	// Add websocket
	b.ws = newWebsocket(b.abilities, o.Websocket)
	return
}

// Close implements the io.Closer interface
func (b *Brain) Close() (err error) {
	// Close abilities
	b.abilities.abilities(func(a *ability) error {
		// Log
		astilog.Debugf("astibrain: closing ability %s", a.name)

		// Switch the ability off
		a.off()

		// Wait for the ability to be really off
		a.mr.Lock()
		a.mr.Unlock()

		// Close
		if v, ok := a.a.(io.Closer); ok {
			if err := v.Close(); err != nil {
				astilog.Error(errors.Wrapf(err, "astibrain: closing ability %s failed", a.name))
			}
		}
		return nil
	})

	// Close ws
	astilog.Debug("astibrain: closing websocket")
	if err = b.ws.Close(); err != nil {
		err = errors.Wrap(err, "astibrain: closing websocket failed")
		return
	}
	return
}

// Learn allows the brain to learn a new ability.
func (b *Brain) Learn(a Ability, o AbilityOptions) {
	// Log
	astilog.Debugf("astibrain: learning %s", a.Name())

	// Add ability
	b.abilities.set(newAbility(a, b.ws, o))

	// Add custom websocket listeners
	if v, ok := a.(WebsocketListener); ok {
		for n, l := range v.WebsocketListeners() {
			b.ws.c.AddListener(WebsocketAbilityEventName(a.Name(), n), l)
		}
	}
}

// Run runs the brain
func (b *Brain) Run(ctx context.Context) (err error) {
	// Reset context
	b.ctx, b.cancel = context.WithCancel(ctx)
	defer b.cancel()

	// Get name
	var name = b.o.Name
	if len(name) == 0 {
		// Get hostname
		if name, err = os.Hostname(); err != nil {
			err = errors.Wrap(err, "astibrain: getting hostname failed")
			return
		}
	}

	// Dial
	go b.ws.dial(b.ctx, name)

	// Loop through abilities
	if err = b.abilities.abilities(func(a *ability) (err error) {
		// Initialize
		if v, ok := a.a.(Initializable); ok {
			astilog.Debugf("astibrain: initializing %s", a.name)
			if err = v.Init(); err != nil {
				err = errors.Wrapf(err, "astibrain: initializing %s failed", a.name)
				return
			}

		}

		// Auto start
		if a.o.AutoStart {
			a.on()
		}
		return
	}); err != nil {
		err = errors.Wrap(err, "astibrain: initializing abilities failed")
		return
	}

	// Wait for context to be done
	<-b.ctx.Done()
	return
}
