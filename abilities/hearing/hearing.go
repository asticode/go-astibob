package astihearing

import (
	"context"
	"os"

	"github.com/asticode/go-astilog"
	"github.com/pkg/errors"
)

// Hearing represents an object capable of parsing an audio reader, split it in valuable chunks and execute a speech to
// text analysis on each of them.
type Hearing struct {
	o Options
	r SampleReader
}

// SampleReader represents a sample reader
type SampleReader interface {
	ReadSample() (int32, error)
}

// Starter represents an object capable of starting and stopping itself
type Starter interface {
	Start() error
	Stop() error
}

// Options represents hearing options.
type Options struct {
	WorkingDirectory string `toml:"working_directory"`
}

// New creates a new hearing.
func New(r SampleReader, o Options) *Hearing {
	return &Hearing{
		o: o,
		r: r,
	}
}

// Close implements the io.Closer interface.
func (h *Hearing) Close() error {
	return nil
}

// Init implements the astibrain.Initializable interface.
func (h *Hearing) Init() (err error) {
	// Create the working directory
	astilog.Debugf("astihearing: creating working directory %s", h.o.WorkingDirectory)
	if err = os.MkdirAll(h.o.WorkingDirectory, 0755); err != nil {
		err = errors.Wrapf(err, "astihearing: mkdirall %s failed", h.o.WorkingDirectory)
		return
	}
	return
}

// Run implements the astibrain.Runnable interface
// TODO Fix when running after having switched it off
func (h *Hearing) Run(ctx context.Context) (err error) {
	// Start and stop the reader
	if v, ok := h.r.(Starter); ok {
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
		if s, err = h.r.ReadSample(); err != nil {
			err = errors.Wrap(err, "astihearing: reading sample failed")
			return
		}
		_ = s

		// TODO Split in smart chunks depending on audio level
		// TODO Speech to text
		// TODO If success, send in channel
		// TODO If failure, store on disk for future use
	}
	return
}
