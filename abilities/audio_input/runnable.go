package audio_input

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/asticode/go-astibob"
	"github.com/asticode/go-astilog"
	"github.com/pkg/errors"
)

// Message names
const (
	eventHearSamplesMessage = "event.hear.samples"
)

type Stream interface {
	BitDepth() int
	MaxSilenceAudioLevel() float64
	Read() ([]int32, error)
	SampleRate() float64
	Start() error
	Stop() error
}

type runnable struct {
	*astibob.BaseRunnable
	m *sync.Mutex
	s Stream
}

func NewRunnable(name string, s Stream) astibob.Runnable {
	// Create runnable
	r := &runnable{
		m: &sync.Mutex{},
		s: s,
	}

	// Set base runnable
	r.BaseRunnable = astibob.NewBaseRunnable(astibob.BaseRunnableOptions{
		Metadata: astibob.Metadata{
			Description: "Reads an audio input and dispatches audio samples",
			Name:        name,
		},
		OnStart: r.onStart,
	})
	return r
}

func (r *runnable) onStart(ctx context.Context) (err error) {
	// Start stream
	if err = r.s.Start(); err != nil {
		err = errors.Wrap(err, "audio_input: starting stream failed")
		return
	}

	// Make sure to stop stream
	defer func() {
		if err := r.s.Stop(); err != nil {
			astilog.Error(errors.Wrap(err, "audio_input: stopping stream failed"))
			return
		}
	}()

	// Read
	for {
		// Check context
		if ctx.Err() != nil {
			return
		}

		// Read
		var b []int32
		if b, err = r.s.Read(); err != nil {
			err = errors.Wrap(err, "audio_input: reading failed")
			return
		}

		// Create message
		var m *astibob.Message
		if m, err = r.newSamplesMessage(b); err != nil {
			err = errors.Wrap(err, "audio_input: creating samples message failed")
			return
		}

		// Dispatch
		r.Dispatch(m)
	}
	return
}

type Samples struct {
	BitDepth             int     `json:"bit_depth"`
	MaxSilenceAudioLevel float64 `json:"max_silence_audio_level"`
	Samples              []int32 `json:"samples"`
	SampleRate           float64 `json:"sample_rate"`
}

func (r *runnable) newSamplesMessage(b []int32) (m *astibob.Message, err error) {
	// Create message
	m = astibob.NewMessage()

	// Set name
	m.Name = eventHearSamplesMessage

	// Marshal
	if m.Payload, err = json.Marshal(Samples{
		BitDepth:             r.s.BitDepth(),
		MaxSilenceAudioLevel: r.s.MaxSilenceAudioLevel(),
		Samples:              b,
		SampleRate:           r.s.SampleRate(),
	}); err != nil {
		err = errors.Wrap(err, "audio_input: marshaling payload failed")
		return
	}
	return
}

func parseSamplesPayload(m *astibob.Message) (ss Samples, err error) {
	if err = json.Unmarshal(m.Payload, &ss); err != nil {
		err = errors.Wrap(err, "audio_input: unmarshaling failed")
		return
	}
	return
}
