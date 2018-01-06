package astihearing

import (
	"context"

	"github.com/asticode/go-astilog"
	"github.com/pkg/errors"
)

// Ability represents an object capable of parsing an audio reader and split it in valuable chunks.
type Ability struct {
	r SampleReader
}

// NewAbility creates a new ability.
func NewAbility(r SampleReader) *Ability {
	return &Ability{
		r: r,
	}
}

// Close implements the io.Closer interface.
func (a *Ability) Close() error {
	return nil
}

// Name implements the astibrain.Ability interface
func (a *Ability) Name() string {
	return Name
}

// Run implements the astibrain.Runnable interface
// TODO Fix when running after having switched it off
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

	// Read
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
		_ = s

		// TODO Split in smart chunks depending on audio level => use input function for that
	}
	return
}
