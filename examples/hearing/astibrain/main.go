package main

import (
	"github.com/asticode/go-astibob/abilities/hearing"
	"github.com/asticode/go-astibob/brain"
	"github.com/asticode/go-astibob/examples"
	"github.com/asticode/go-astibob/pkg/portaudio"
	"github.com/asticode/go-astilog"
	"github.com/pkg/errors"
)

func main() {
	// Base init
	ctx := astiexamples.Init()

	// Create portaudio
	p, err := astiportaudio.New()
	if err != nil {
		astilog.Fatal(errors.Wrap(err, "main: creating portaudio failed"))
	}
	defer p.Close()

	// Create portaudio stream
	s, err := p.NewDefaultStream(make([]int32, 192), astiportaudio.StreamOptions{
		NumInputChannels: 1,
		SampleRate:       16000,
	})
	if err != nil {
		astilog.Fatal(errors.Wrap(err, "main: creating portaudio default stream failed"))
	}
	defer s.Close()

	// Create brain
	brain := astibrain.New(astiexamples.BrainOptions)
	defer brain.Close()

	// Create hearing
	hearing := astihearing.NewAbility(s, astihearing.AbilityOptions{
		DispatchCount: 16000,
	})

	// Learn ability
	brain.Learn(hearing, astiexamples.AbilityOptions)

	// Run the brain
	if err := brain.Run(ctx); err != nil {
		astilog.Fatal(errors.Wrap(err, "main: running brain failed"))
	}
}
