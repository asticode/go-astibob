package audio_input

import (
	"fmt"

	"github.com/asticode/go-astibob"
)

type ListenableOptions struct {
	OnSamples func(from astibob.Identifier, samples []int, bitDepth, numChannels, sampleRate int, maxSilenceLevel float64) error
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
			err = fmt.Errorf("audio_input: on samples failed: %w", err)
			return
		}
	}
	return
}

func (l *Listenable) onSamples(m *astibob.Message) (err error) {
	// Parse payload
	var s Samples
	if s, err = parseSamplesPayload(m); err != nil {
		err = fmt.Errorf("audio_input: parsing samples payload failed: %w", err)
		return
	}

	// Custom
	if l.o.OnSamples != nil {
		if err = l.o.OnSamples(m.From, s.Samples, s.BitDepth, s.NumChannels, s.SampleRate, s.MaxSilenceLevel); err != nil {
			err = fmt.Errorf("audio_input: custom on samples failed: %w", err)
			return
		}
	}
	return
}
