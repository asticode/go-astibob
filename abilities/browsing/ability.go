package astibrowsing

import (
	"context"
	"net"
	"net/http"

	"github.com/asticode/go-astilog"
	"github.com/asticode/go-astitools/exec"
	"github.com/asticode/go-astitools/http"
	"github.com/pkg/errors"
)

// Ability represents an object capable of spawning a server and opening its homepage in the default browser.
// TODO Add demo
type Ability struct {
	h http.Handler
}

// NewAbility creates a new ability.
func NewAbility(h http.Handler) *Ability {
	return &Ability{h: h}
}

// Name implements the astibrain.Ability interface
func (a *Ability) Name() string {
	return name
}

// Description implements the astibrain.Ability interface
func (a *Ability) Description() string {
	return "Spawns a server and opens its homepage in the default browser"
}

// Run implements the astibrain.Runnable interface
func (a *Ability) Run(ctx context.Context) (err error) {
	// Serve
	if err = astihttp.Serve(ctx, a.h, func(a net.Addr) {
		if err := astiexec.OpenBrowser(ctx, "http://" +a.String()); err != nil {
			astilog.Error(errors.Wrap(err, "astibrowsing: opening browser failed"))
		}
	}); err != nil {
		err = errors.Wrap(err, "astibrowsing: serving failed")
		return
	}
	return
}
