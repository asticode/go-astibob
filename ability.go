package astibob

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/asticode/go-astilog"
	"github.com/asticode/go-astiws"
	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
)

// Ability represents an ability.
type Ability interface {
	Run(ctx context.Context) error
}

// Initializer represents an object capable of initializing itself
type Initializer interface {
	Init() error
}

// Router represents an object capable of adding custom routes to the router
type Router interface {
	Routes() []Route
}

// Route represents an http route
type Route struct {
	Handle  httprouter.Handle
	Method  string
	Pattern string
}

// AbilityOptions represents ability options
type AbilityOptions struct {
	AutoStart bool
}

// ability represents an ability.
type ability struct {
	a    Ability
	key  string
	name string
	o    AbilityOptions
	t    *toggle
	ws   *astiws.Manager
}

// newAbility creates a new ability.
func newAbility(name string, a Ability, o AbilityOptions, ws *astiws.Manager) *ability {
	return &ability{
		a:    a,
		key:  abilityKey(name),
		name: name,
		o:    o,
		t:    newToggle(a.Run),
		ws:   ws,
	}
}

// regexpAbilityKey represents the ability key regexp
var regexpAbilityKey = regexp.MustCompile("[^\\w]+")

// abilityKey creates an ability key
func abilityKey(name string) string {
	return regexpAbilityKey.ReplaceAllString(strings.ToLower(name), "-")
}

// on switches the ability on.
func (a *ability) on() {
	// Ability is already on
	if a.t.isOn() {
		return
	}

	// Switch on
	astilog.Debugf("astibob: switching %s on", a.name)
	a.t.on()

	// Wait for the end of execution in a go routine
	go func() {
		// Wait
		if err := a.t.wait(); err != nil && err != context.Canceled {
			// Log
			astilog.Error(errors.Wrapf(err, "astibob: %s crashed", a.name))

			// Dispatch websocket event
			dispatchWsEvent(a.ws, websocketEventNameAbilityCrashed, a.key)
		} else {
			// Log
			astilog.Infof("astibob: %s have been switched off", a.name)

			// Dispatch websocket event
			dispatchWsEvent(a.ws, websocketEventNameAbilityOff, a.key)
		}
	}()

	// Log
	astilog.Infof("astibob: %s have been switched on", a.name)

	// Dispatch websocket event
	dispatchWsEvent(a.ws, websocketEventNameAbilityOn, a.key)
}

// off switches the ability off.
func (a *ability) off() {
	// Ability is already off
	if !a.t.isOn() {
		return
	}

	// Switch off
	astilog.Debugf("astibob: switching %s off", a.name)
	a.t.off()

	// The rest is handled through the wait function
}

// adaptRouter adapts the router.
func (a *ability) adaptRouter(r *httprouter.Router) {
	// Default routes
	r.GET(fmt.Sprintf("/api/abilities/%s/on", a.key), a.handleAPIOnGET)
	r.GET(fmt.Sprintf("/api/abilities/%s/off", a.key), a.handleAPIOffGET)

	// Custom routes
	if v, ok := a.a.(Router); ok {
		for _, rt := range v.Routes() {
			r.Handle(rt.Method, fmt.Sprintf("/api/abilities/%s%s", a.key, rt.Pattern), rt.Handle)
		}
	}
}

// handleAPIOnGET switches an ability on.
func (a *ability) handleAPIOnGET(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
	a.on()
}

// handleAPIOffGET switches an ability off.
func (a *ability) handleAPIOffGET(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
	a.off()
}
