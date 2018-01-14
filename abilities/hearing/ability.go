package astihearing

import (
	"context"
	"math"
	"time"

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
	DispatchDuration     time.Duration `toml:"dispatch_duration"`
	SampleRate           int           `toml:"sample_rate"`
	SignificantBits      int           `toml:"significant_bits"`
	SilenceMaxAudioLevel float64       `toml:"silence_max_audio_level"`
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
	return name
}

// Description implements the astibrain.Ability interface
func (a *Ability) Description() string {
	return "Listens to an audio input and dispatches audio samples"
}

// PayloadSamples represents the samples payload
type PayloadSamples struct {
	SampleRate           int     `json:"sample_rate"`
	Samples              []int32 `json:"samples"`
	SignificantBits      int     `json:"significant_bits"`
	SilenceMaxAudioLevel float64 `json:"silence_max_audio_level"`
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
	var dispatchCount = a.o.SampleRate
	if a.o.DispatchDuration > 0 {
		dispatchCount = int(math.Floor(float64(a.o.SampleRate) * a.o.DispatchDuration.Seconds()))
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
				AbilityName: name,
				Name:        websocketEventNameSamples,
				Payload: PayloadSamples{
					SampleRate:           a.o.SampleRate,
					Samples:              dispatchBuf,
					SignificantBits:      a.o.SignificantBits,
					SilenceMaxAudioLevel: a.o.SilenceMaxAudioLevel,
				},
			})
		}
	}
}
