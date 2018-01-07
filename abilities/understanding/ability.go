package astiunderstanding

import (
	"context"
	"encoding/json"
	"time"

	"github.com/asticode/go-astibob/brain"
	"github.com/asticode/go-astilog"
	"github.com/asticode/go-astiws"
	"github.com/pkg/errors"
)

// Ability represents an object capable of doing speech to text analysis
type Ability struct {
	ch           chan PayloadSamples
	dispatchFunc astibrain.DispatchFunc
	p            SpeechParser
}

// NewAbility creates a new ability
func NewAbility(p SpeechParser) *Ability {
	return &Ability{
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
	var buf []int32
	for {
		select {
		case p := <-a.ch:
			// Retrieve valid samples
			validSamples := a.validSamples(p, &buf)

			// No valid samples
			if len(validSamples) <= 0 {
				continue
			}

			// Execute speech to text analysis
			start := time.Now()
			astilog.Debugf("astiunderstanding: starting speech to text analysis on %d samples", len(validSamples))
			s := a.p.SpeechToText(validSamples, len(validSamples), p.SampleRate, p.SignificantBits)
			astilog.Debugf("astiunderstanding: speech to text analysis done in %s", time.Now().Sub(start))

			// TODO Store everything for later validation and model improvement

			// Dispatch
			a.dispatchFunc(astibrain.Event{
				AbilityName: Name,
				Name:        websocketEventNameAnalysis,
				Payload:     s,
			})
		case <-ctx.Done():
			err = errors.Wrap(err, "astiunderstanding: context error")
			return
		}
	}
	return
}

// validSamples processes new samples and checks whether it adds up to valid samples when appended to buffered samples
func (a *Ability) validSamples(p PayloadSamples, buf *[]int32) (validSamples []int32) {
	// TODO Detect with audio level

	// Append new samples
	*buf = append(*buf, p.Samples...)

	// TEST
	if len(*buf) > 16000 {
		validSamples = make([]int32, len(*buf))
		copy(validSamples, *buf)
		*buf = (*buf)[:0]
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
