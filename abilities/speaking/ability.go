package astispeaking

import (
	"sync"

	"encoding/json"

	"github.com/asticode/go-astilog"
	"github.com/asticode/go-astiws"
	"github.com/pkg/errors"
)

// Ability represents an object capable of saying words to an audio output.
type Ability struct {
	activated bool
	m         sync.Mutex
	s         Speaker
}

// NewAbility creates a new ability
func NewAbility(s Speaker) *Ability {
	return &Ability{s: s}
}

// Name implements the astibrain.Ability interface
func (a *Ability) Name() string {
	return name
}

// Description implements the astibrain.Ability interface
func (a *Ability) Description() string {
	return "Says words to your audio output using speech synthesis"
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
		websocketEventNameSay: a.websocketListenerSay,
	}
}

// websocketListenerSay listens to the say websocket event
func (a *Ability) websocketListenerSay(c *astiws.Client, eventName string, payload json.RawMessage) error {
	// Ability is not activated
	a.m.Lock()
	activated := a.activated
	a.m.Unlock()
	if !activated {
		astilog.Error("astispeaking: ability is not activated")
		return nil
	}

	// Unmarshal payload
	var i string
	if err := json.Unmarshal(payload, &i); err != nil {
		astilog.Error(errors.Wrapf(err, "astispeaking: json unmarshaling %s into %#v failed", payload, i))
		return nil
	}

	// Say
	astilog.Debugf("astispeaking: saying %s", i)
	if err := a.s.Say(i); err != nil {
		astilog.Error(errors.Wrapf(err, "astispeaking: saying %s failed", i))
		return nil
	}
	return nil
}
