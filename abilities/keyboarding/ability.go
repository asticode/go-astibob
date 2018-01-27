package astikeyboarding

import (
	"encoding/json"
	"sync"

	"strings"

	"github.com/asticode/go-astilog"
	"github.com/asticode/go-astiws"
	"github.com/pkg/errors"
)

// Ability represents an object capable of interacting with a keyboard.
type Ability struct {
	activated bool
	k         Keyboarder
	m         sync.Mutex
}

// NewAbility creates a new ability
func NewAbility(k Keyboarder) *Ability {
	return &Ability{
		k: k,
	}
}

// Name implements the astibrain.Ability interface
func (a *Ability) Name() string {
	return name
}

// Description implements the astibrain.Ability interface
func (a *Ability) Description() string {
	return "Interacts with your keyboard"
}

// Activate implements the astibrain.Activable interface
func (a *Ability) Activate(activated bool) {
	a.m.Lock()
	defer a.m.Unlock()
	a.activated = activated
}

// WebsocketListeners implements the astibrain.WebsocketListener interface
func (a *Ability) WebsocketListeners() map[string]astiws.ListenerFunc {
	return map[string]astiws.ListenerFunc{
		websocketEventNameAction: a.websocketListenerAction,
	}
}

// websocketListenerAction listens to the action websocket event
func (a *Ability) websocketListenerAction(c *astiws.Client, eventName string, payload json.RawMessage) error {
	// Ability is not activated
	a.m.Lock()
	activated := a.activated
	a.m.Unlock()
	if !activated {
		astilog.Error("astikeyboarding: ability is not activated")
		return nil
	}

	// Unmarshal payload
	var p PayloadAction
	if err := json.Unmarshal(payload, &p); err != nil {
		astilog.Error(errors.Wrapf(err, "astikeyboarding: json unmarshaling %s into %#v failed", payload, p))
		return nil
	}

	// Switch on action
	switch p.Action {
	case actionPress:
		astilog.Debugf("astikeyboarding: pressing %s", strings.Join(p.Keys, "/"))
		a.k.Press(p.Keys...)
	case actionType:
		astilog.Debugf("astikeyboarding: typing %s", p.String)
		a.k.Type(p.String)
	default:
		astilog.Errorf("astikeyboarding: unknown action %s", p.Action)
	}
	return nil
}
