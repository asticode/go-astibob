package astimousing

import (
	"encoding/json"
	"sync"

	"github.com/asticode/go-astilog"
	"github.com/asticode/go-astiws"
	"github.com/pkg/errors"
)

// Ability represents an object capable of saying words to an audio output.
type Ability struct {
	activated bool
	m         sync.Mutex
	ms        Mouser
}

// NewAbility creates a new ability
func NewAbility(ms Mouser) *Ability {
	return &Ability{
		ms: ms,
	}
}

// Name implements the astibrain.Ability interface
func (a *Ability) Name() string {
	return name
}

// Description implements the astibrain.Ability interface
func (a *Ability) Description() string {
	return "Interacts with your mouse"
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
		astilog.Error("astimousing: ability is not activated")
		return nil
	}

	// Unmarshal payload
	var p PayloadAction
	if err := json.Unmarshal(payload, &p); err != nil {
		astilog.Error(errors.Wrapf(err, "astimousing: json unmarshaling %s into %#v failed", payload, p))
		return nil
	}

	// Switch on action
	switch p.Action {
	case actionClickLeft:
		astilog.Debugf("astimousing: clicking left mouse button with double %v", p.Double)
		a.ms.ClickLeft(p.Double)
	case actionClickMiddle:
		astilog.Debugf("astimousing: clicking middle mouse button with double %v", p.Double)
		a.ms.ClickMiddle(p.Double)
	case actionClickRight:
		astilog.Debugf("astimousing: clicking right mouse button with double %v", p.Double)
		a.ms.ClickRight(p.Double)
	case actionMove:
		astilog.Debugf("astimousing: moving mouse to %dx%d", p.X, p.Y)
		a.ms.Move(p.X, p.Y)
	case actionScrollDown:
		astilog.Debugf("astimousing: scrolling down with x %d", p.X)
		a.ms.ScrollDown(p.X)
	case actionScrollUp:
		astilog.Debugf("astimousing: scrolling up with x %d", p.X)
		a.ms.ScrollUp(p.X)
	default:
		astilog.Errorf("astimousing: unknown action %s", p.Action)
	}
	return nil
}
