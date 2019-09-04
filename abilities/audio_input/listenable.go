package audio_input

import (
	"github.com/asticode/go-astibob"
	"github.com/pkg/errors"
)

type ListenableOptions struct {
	OnSamples func(from astibob.Identifier, samples []int32, bitDepth int, sampleRate, maxSilenceAudioLevel float64) error
}

type Listenable struct {
	o ListenableOptions
}

func NewListenable(o ListenableOptions) *Listenable {
	return newListenable(o)
}

func newListenable(o ListenableOptions) *Listenable {
	return &Listenable{o: o}
}

func (l *Listenable) MessageNames() (ns []string) {
	if l.o.OnSamples != nil {
		ns = append(ns, samplesMessage)
	}
	return
}

func (l *Listenable) OnMessage(m *astibob.Message) (err error) {
	switch m.Name {
	case samplesMessage:
		if err = l.onSamples(m); err != nil {
			err = errors.Wrap(err, "audio_input: on samples failed")
			return
		}
	}
	return
}

func (l *Listenable) onSamples(m *astibob.Message) (err error) {
	// Parse payload
	var s Samples
	if s, err = parseSamplesPayload(m); err != nil {
		err = errors.Wrap(err, "audio_input: parsing samples payload failed")
		return
	}

	// Custom
	if l.o.OnSamples != nil {
		if err = l.o.OnSamples(m.From, s.Samples, s.BitDepth, s.SampleRate, s.MaxSilenceAudioLevel); err != nil {
			err = errors.Wrap(err, "audio_input: custom on samples failed")
			return
		}
	}
	return
}
