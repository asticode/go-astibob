package astihearing

import (
	"context"

	"github.com/asticode/go-astibob/brain"
	"github.com/asticode/go-astilog"
	"github.com/pkg/errors"
)

// Ability represents an object capable of parsing an audio reader and dispatch n chunks.
type Ability struct {
	ch chan astibrain.Event
	o  AbilityOptions
	r  SampleReader
}

// AbilityOptions represents ability options
type AbilityOptions struct {
	DispatchCount int `toml:"dispatch_count"`
}

// NewAbility creates a new ability.
func NewAbility(r SampleReader, o AbilityOptions) *Ability {
	return &Ability{
		o: o,
		r: r,
	}
}

// Close implements the io.Closer interface.
func (a *Ability) Close() error {
	return nil
}

// SetDispatchChan implements the astibrain.WebsocketDispatcher interface
func (a *Ability) SetDispatchChan(ch chan astibrain.Event) {
	a.ch = ch
}

// Name implements the astibrain.Ability interface
func (a *Ability) Name() string {
	return Name
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

		// Reset buffer
		if len(buf) == 0 || len(buf) >= dispatchCount {
			buf = buf[:0]
		}

		// Add sample to buffer
		buf = append(buf, s)

		// Dispatch
		if len(buf) >= dispatchCount {
			dispatchBuf := make([]int32, len(buf))
			copy(dispatchBuf, buf)
			a.ch <- astibrain.Event{
				AbilityName: Name,
				Name:        websocketEventNameSamples,
				Payload:     dispatchBuf,
			}
		}
	}
	return
}
