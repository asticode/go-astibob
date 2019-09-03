package audio_input

import (
	"github.com/asticode/go-astibob"
	"github.com/pkg/errors"
)

type ListenableOptions struct {
	OnSamples func(samples []int32, bitDepth int, sampleRate, maxSilenceAudioLevel float64) error
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
		ns = append(ns, eventSamplesMessage)
	}
	return
}

func (l *listenable) OnMessage(m *astibob.Message) (err error) {
	switch m.Name {
	case eventSamplesMessage:
		if err = l.onSamples(m); err != nil {
			err = errors.Wrap(err, "audio_input: on samples failed")
			return
		}
	}
	return
}

func (l *listenable) onSamples(m *astibob.Message) (err error) {
	// Parse payload
	var s Samples
	if s, err = parseSamplesPayload(m); err != nil {
		err = errors.Wrap(err, "audio_input: parsing samples payload failed")
		return
	}

	// Custom
	if l.o.OnSamples != nil {
		if err = l.o.OnSamples(s.Samples, s.BitDepth, s.SampleRate, s.MaxSilenceAudioLevel); err != nil {
			err = errors.Wrap(err, "audio_input: custom on samples failed")
			return
		}
	}
	return
}
