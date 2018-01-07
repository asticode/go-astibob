package astiunderstanding

import (
	"context"
	"encoding/json"
	"time"

	"github.com/asticode/go-astibob/brain"
	"github.com/asticode/go-astilog"
	"github.com/asticode/go-astitools/audio"
	"github.com/asticode/go-astiws"
	"github.com/pkg/errors"
)

// Ability represents an object capable of doing speech to text analysis
type Ability struct {
	ch           chan PayloadSamples
	d            *astiaudio.SilenceDetector
	dispatchFunc astibrain.DispatchFunc
	o            AbilityOptions
	p            SpeechParser
}

// AbilityOptions represents ability options
type AbilityOptions struct {
	SilenceDetector astiaudio.SilenceDetectorOptions
}

// NewAbility creates a new ability
func NewAbility(p SpeechParser, o AbilityOptions) *Ability {
	return &Ability{
		d: astiaudio.NewSilenceDetector(o.SilenceDetector),
		o: o,
		p: p,
	}
}

// SetDispatchFunc implements the astibrain.Dispatcher interface
func (a *Ability) SetDispatchFunc(fn astibrain.DispatchFunc) {
	a.dispatchFunc = fn
}

// Name implements the astibrain.Ability interface
func (a *Ability) Name() string {
	return Name
}

// Run implements the astibrain.Runnable interface
func (a *Ability) Run(ctx context.Context) (err error) {
	// Reset in channel
	a.ch = make(chan PayloadSamples)

	// Listen
	for {
		select {
		case p := <-a.ch:
			// Add samples to silence detector and retrieve speech samples
			// TODO Ease finding silence max audio level
			speechSamples := a.d.Add(p.Samples, p.SampleRate)

			// No speech samples
			if len(speechSamples) <= 0 {
				continue
			}

			// Loop through speech samples
			for _, samples := range speechSamples {
				// Execute speech to text analysis
				start := time.Now()
				astilog.Debugf("astiunderstanding: starting speech to text analysis on %d samples", len(samples))
				s := a.p.SpeechToText(samples, len(samples), p.SampleRate, p.SignificantBits)
				astilog.Debugf("astiunderstanding: speech to text analysis done in %s", time.Now().Sub(start))

				// TODO Store everything for later validation and model improvement

				// Dispatch
				a.dispatchFunc(astibrain.Event{
					AbilityName: Name,
					Name:        websocketEventNameAnalysis,
					Payload:     s,
				})
			}
		case <-ctx.Done():
			err = errors.Wrap(err, "astiunderstanding: context error")
			return
		}
	}
	return
}

// WebsocketListeners implements the astibrain.WebsocketListener interface
func (a *Ability) WebsocketListeners() map[string]astiws.ListenerFunc {
	return map[string]astiws.ListenerFunc{
		websocketEventNameSamples: a.websocketListenerSamples,
	}
}

// websocketListenerSamples listens to the samples websocket event
func (a *Ability) websocketListenerSamples(c *astiws.Client, eventName string, payload json.RawMessage) error {
	// Unmarshal payload
	var p PayloadSamples
	if err := json.Unmarshal(payload, &p); err != nil {
		astilog.Error(errors.Wrapf(err, "astiunderstanding: json unmarshaling %s into %#v failed", payload, p))
		return nil
	}

	// Dispatch
	a.ch <- p
	return nil
}
