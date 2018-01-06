package astihearing

import (
	"encoding/json"

	"github.com/asticode/go-astilog"
	"github.com/asticode/go-astiws"
	"github.com/pkg/errors"
)

// Interface is the interface of the ability
type Interface struct {
	onSamples SamplesFunc
}

// SamplesFunc represents the callback executed upon receiving samples
type SamplesFunc func(samples []int32) error

// NewInterface creates a new interface
func NewInterface() *Interface {
	return &Interface{}
}

// Name implements the astibob.Interface interface
func (i *Interface) Name() string {
	return Name
}

// OnSamples set the callback executed upon receiving samples
func (i *Interface) OnSamples(fn SamplesFunc) {
	i.onSamples = fn
}

// WebsocketListeners implements the astibob.WebsocketListener interface
func (i *Interface) WebsocketListeners() map[string]astiws.ListenerFunc {
	return map[string]astiws.ListenerFunc{
		websocketEventNameSamples: i.websocketListenerSamples,
	}
}

// websocketListenerSamples listens to the samples websocket event
func (i *Interface) websocketListenerSamples(c *astiws.Client, eventName string, payload json.RawMessage) error {
	// Unmarshal payload
	var samples []int32
	if err := json.Unmarshal(payload, &samples); err != nil {
		astilog.Error(errors.Wrapf(err, "astihearing: json unmarshaling %s into %#v failed", payload, samples))
		return nil
	}

	// No callback
	if i.onSamples == nil {
		astilog.Error("astihearing: onSamples is undefined")
		return nil
	}

	// Execute callback
	if err := i.onSamples(samples); err != nil {
		astilog.Error(errors.Wrap(err, "astihearing: executing samples callback failed"))
		return nil
	}
	return nil
}
