package astihearing

import (
	"context"

	"github.com/asticode/go-astibob/brain"
	"github.com/asticode/go-astilog"
	"github.com/pkg/errors"
)

// Ability represents an object capable of parsing an audio reader and dispatch n chunks.
type Ability struct {
	dispatchFunc astibrain.DispatchFunc
	o            AbilityOptions
	r            SampleReader
}

// AbilityOptions represents ability options
type AbilityOptions struct {
	DispatchCount   int `toml:"dispatch_count"`
	SampleRate      int `toml:"sample_rate"`
	SignificantBits int `toml:"significant_bits"`
}

// NewAbility creates a new ability.
func NewAbility(r SampleReader, o AbilityOptions) *Ability {
	return &Ability{
		o: o,
		r: r,
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

// PayloadSamples represents the samples payload
type PayloadSamples struct {
	SampleRate      int     `json:"sample_rate"`
	Samples         []int32 `json:"samples"`
	SignificantBits int     `json:"significant_bits"`
}

// Run implements the astibrain.Runnable interface
func (a *Ability) Run(ctx context.Context) (err error) {
	// Start and stop the reader
	if v, ok := a.r.(Starter); ok {
		// Start the reader
		astilog.Debug("astihearing: starting reader")
		if err = v.Start(); err != nil {
			err = errors.Wrap(err, "astihearing: starting reader failed")
			return
		}

		// Stop the reader
		defer func() {
			astilog.Debug("astihearing: stopping reader")
			if err := v.Stop(); err != nil {
				astilog.Error(errors.Wrap(err, "astihearing: stopping reader failed"))
			}
		}()
	}

	// Get dispatch count
	var dispatchCount = a.o.DispatchCount
	if dispatchCount <= 0 {
		dispatchCount = 1
	}

	// Read
	var buf = make([]int32, dispatchCount)
	var s int32
	for {
		// Check context
		if err = ctx.Err(); err != nil {
			err = errors.Wrap(err, "astihearing: context error")
			return
		}

		// Read sample
		if s, err = a.r.ReadSample(); err != nil {
			err = errors.Wrap(err, "astihearing: reading sample failed")
			return
		}

		// Add sample to buffer
		buf = append(buf, s)

		// Dispatch
		if len(buf) >= dispatchCount {
			dispatchBuf := make([]int32, len(buf))
			copy(dispatchBuf, buf)
			buf = buf[:0]
			a.dispatchFunc(astibrain.Event{
				AbilityName: Name,
				Name:        websocketEventNameSamples,
				Payload: PayloadSamples{
					SampleRate:      a.o.SampleRate,
					Samples:         dispatchBuf,
					SignificantBits: a.o.SignificantBits,
				},
			})
		}
	}
	return
}
