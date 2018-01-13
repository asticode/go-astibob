package astihearing

import (
	"encoding/json"

	"github.com/asticode/go-astilog"
	"github.com/asticode/go-astiws"
	"github.com/pkg/errors"
)

// Interface is the interface of the ability
// TODO Add calibrate UI to get max audio level + silence max audio level
type Interface struct {
	onSamples []SamplesFunc
}

// SamplesFunc represents the callback executed upon receiving samples
type SamplesFunc func(samples []int32, sampleRate, significantBits int, silenceMaxAudioLevel float64) error

// NewInterface creates a new interface
func NewInterface() *Interface {
	return &Interface{}
}

// Name implements the astibob.Interface interface
func (i *Interface) Name() string {
	return Name
}

// OnSamples adds a callback executed upon receiving samples
func (i *Interface) OnSamples(fn SamplesFunc) {
	i.onSamples = append(i.onSamples, fn)
}

// BrainWebsocketListeners implements the astibob.BrainWebsocketListener interface
func (i *Interface) BrainWebsocketListeners() map[string]astiws.ListenerFunc {
	return map[string]astiws.ListenerFunc{
		websocketEventNameSamples: i.brainWebsocketListenerSamples,
	}
}

// brainWebsocketListenerSamples listens to the samples brain websocket event
func (i *Interface) brainWebsocketListenerSamples(c *astiws.Client, eventName string, payload json.RawMessage) error {
	// Unmarshal payload
	var p PayloadSamples
	if err := json.Unmarshal(payload, &p); err != nil {
		astilog.Error(errors.Wrapf(err, "astihearing: json unmarshaling %s into %#v failed", payload, p))
		return nil
	}

	// No callback
	if i.onSamples == nil {
		astilog.Error("astihearing: onSamples is undefined")
		return nil
	}

	// Execute callbacks
	for _, fn := range i.onSamples {
		if err := fn(p.Samples, p.SampleRate, p.SignificantBits, p.SilenceMaxAudioLevel); err != nil {
			astilog.Error(errors.Wrap(err, "astihearing: executing samples callback failed"))
		}
	}
	return nil
}
