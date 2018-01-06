package astiportaudio

import (
	"github.com/asticode/go-astilog"
	"github.com/gordonklaus/portaudio"
	"github.com/pkg/errors"
)

// Stream represents a portaudio stream
type Stream struct {
	b     []int32
	o     StreamOptions
	queue []int32
	s     *portaudio.Stream
}

// StreamOptions represents stream options
type StreamOptions struct {
	NumInputChannels  int     `toml:"num_input_channels"`
	NumOutputChannels int     `toml:"num_output_channels"`
	SampleRate        float64 `toml:"sample_rate"`
}

// NewDefaultStream creates a new default stream
func (p *PortAudio) NewDefaultStream(b []int32, o StreamOptions) (s *Stream, err error) {
	// Init
	s = &Stream{
		b: b,
		o: o,
	}

	// Open default stream
	astilog.Debugf("astiportaudio: opening default stream %p", s)
	if s.s, err = portaudio.OpenDefaultStream(s.o.NumInputChannels, s.o.NumOutputChannels, s.o.SampleRate, len(s.b), s.b); err != nil {
		err = errors.Wrapf(err, "astiportaudio: opening default stream %p failed", s)
		return
	}
	return
}

// Close implements the io.Closer interface
func (s *Stream) Close() (err error) {
	// Close stream
	astilog.Debugf("astiportaudio: closing stream %p", s)
	if err = s.s.Close(); err != nil {
		err = errors.Wrapf(err, "astiportaudio: closing stream %s failed", s)
		return
	}
	return
}

// Start starts the stream
func (s *Stream) Start() (err error) {
	// Start stream
	astilog.Debugf("astiportaudio: starting stream %p", s)
	if err = s.s.Start(); err != nil {
		err = errors.Wrapf(err, "astiportaudio: starting stream %p failed", s)
		return
	}
	return
}

// Stop stops the stream
func (s *Stream) Stop() (err error) {
	// Stop stream
	astilog.Debugf("astiportaudio: stopping stream %p", s)
	if err = s.s.Stop(); err != nil {
		err = errors.Wrapf(err, "astiportaudio: stopping stream %p failed", s)
		return
	}
	return
}

// ReadSample implements the astihearing.SampleReader interface.
func (s *Stream) ReadSample() (r int32, err error) {
	// Queue is empty
	if len(s.queue) == 0 {
		// Read
		if err = s.s.Read(); err != nil {
			err = errors.Wrap(err, "astiportaudio: reading failed")
			return
		}

		// Copy buffer to queue
		s.queue = make([]int32, len(s.b))
		copy(s.queue, s.b)
	}

	// Process queue
	r = s.queue[0]

	// Update queue
	s.queue = s.queue[1:]
	return
}
