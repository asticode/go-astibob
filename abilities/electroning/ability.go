package astilectroning

import (
	"context"

	"io"

	"github.com/asticode/go-astilectron"
	"github.com/pkg/errors"
)

// Ability represents an object capable of starting an Electron process.
// TODO Add demo
type Ability struct {
	h AstilectronHandler
	o astilectron.Options
}

// AstilectronHandler represents an object capable of handling astilectron
type AstilectronHandler interface {
	HandleAstilectron(a *astilectron.Astilectron) error
}

// NewAbility creates a new ability.
func NewAbility(o astilectron.Options, h AstilectronHandler) *Ability {
	return &Ability{
		h: h,
		o: o,
	}
}

// Name implements the astibrain.Ability interface
func (a *Ability) Name() string {
	return name
}

// Description implements the astibrain.Ability interface
func (a *Ability) Description() string {
	return "Spawns an Electron app"
}

// Run implements the astibrain.Runnable interface
func (a *Ability) Run(ctx context.Context) (err error) {
	// Create astilectron
	var as *astilectron.Astilectron
	if as, err = astilectron.New(a.o); err != nil {
		err = errors.Wrap(err, "astilectroning: creating astilectron failed")
		return
	}
	defer as.Close()

	// Start astilectron
	if err = as.Start(); err != nil {
		err = errors.Wrap(err, "astilectroning: starting astilectron failed")
		return
	}

	// Custom callback
	if err = a.h.HandleAstilectron(as); err != nil {
		err = errors.Wrap(err, "astilectroning: handling astilectron failed")
		return
	}

	// Wait for the context to be done
	select {
	case <-ctx.Done():
		// Stop astilectron
		as.Stop()

		// Close
		if v, ok := a.h.(io.Closer); ok {
			if err = v.Close(); err != nil {
				err = errors.Wrap(err, "astilectroning: closing failed")
				return
			}
		}
	}
	return
}
