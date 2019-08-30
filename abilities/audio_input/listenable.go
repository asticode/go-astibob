package audio_input

import (
	"github.com/asticode/go-astibob"
	"github.com/pkg/errors"
)

type ListenableOptions struct {
	OnSamples func(samples []int32, sampleRate, significantBits int, silenceMaxAudioLevel float64) error
}

func NewListenable(o ListenableOptions) astibob.Listenable {
	return newListenable(o)
}

type listenable struct {
	o ListenableOptions
}

func newListenable(o ListenableOptions) *listenable {
	return &listenable{o: o}
}

func (l *listenable) MessageNames() (ns []string) {
	if l.o.OnSamples != nil {
		ns = append(ns, eventHearSamplesMessage)
	}
	return
}

func (l *listenable) OnMessage(m *astibob.Message) (err error) {
	switch m.Name {
	case eventHearSamplesMessage:
		if err = l.onSamples(m); err != nil {
			err = errors.Wrap(err, "audio_input: on samples failed")
			return
		}
	}
	return
}

func (l *listenable) onSamples(m *astibob.Message) (err error) {
	// Parse payload
	var ss Samples
	if ss, err = parseSamplesPayload(m); err != nil {
		err = errors.Wrap(err, "audio_input: parsing samples payload failed")
		return
	}

	// Custom
	if l.o.OnSamples != nil {
		if err = l.o.OnSamples(ss.Samples, ss.SampleRate, ss.SignificantBits, ss.SilenceMaxAudioLevel); err != nil {
			err = errors.Wrap(err, "audio_input: custom on samples failed")
			return
		}
	}
	return
}
