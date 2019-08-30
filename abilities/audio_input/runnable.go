package audio_input

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/asticode/go-astibob"
	"github.com/pkg/errors"
)

// Message names
const (
	eventHearSamplesMessage = "event.hear.samples"
)

type SampleReader interface {
	ReadSample() (int32, error)
}

type runnable struct {
	*astibob.BaseRunnable
	m *sync.Mutex
	s SampleReader
}

func NewRunnable(name string, s SampleReader) astibob.Runnable {
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
	t := time.NewTicker(2 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			// Create message
			var m *astibob.Message
			if m, err = r.newSamplesMessage([]int32{1, 2, 3}); err != nil {
				err = errors.Wrap(err, "audio_input: creating sample message failed")
				return
			}

			// Dispatch
			r.Dispatch(m)
		case <-ctx.Done():
			return
		}
	}
	return
}

type Samples struct {
	Samples              []int32 `json:"samples"`
	SampleRate           int     `json:"sample_rate"`
	SignificantBits      int     `json:"significant_bits"`
	SilenceMaxAudioLevel float64 `json:"silence_max_audio_level"`
}

func (r *runnable) newSamplesMessage(samples []int32) (m *astibob.Message, err error) {
	// Create message
	m = astibob.NewMessage()

	// Set name
	m.Name = eventHearSamplesMessage

	// Marshal
	if m.Payload, err = json.Marshal(Samples{
		Samples:              samples,
		SampleRate:           1,
		SignificantBits:      2,
		SilenceMaxAudioLevel: 0.9,
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
