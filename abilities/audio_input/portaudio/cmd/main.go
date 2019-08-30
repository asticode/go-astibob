package main

import (
	"log"

	"github.com/asticode/go-astibob/abilities/audio_input/portaudio"
	"github.com/pkg/errors"
)

func main() {
	// Create
	p := portaudio.New()

	// Initialize
	if err := p.Initialize(); err != nil {
		log.Fatal(errors.Wrap(err, "main: initializing failed"))
	}
	defer p.Close()

	// Info
	log.Println(p.Info())
}
