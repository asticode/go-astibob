package main

import (
	"fmt"
	"log"

	"github.com/asticode/go-astibob/abilities/audio_input/portaudio"
)

func main() {
	// Create logger
	l := log.New(log.Writer(), log.Prefix(), log.Flags())

	// Create
	p := portaudio.New(l)

	// Initialize
	if err := p.Initialize(); err != nil {
		log.Fatal(fmt.Errorf("main: initializing failed: %w", err))
	}
	defer p.Close()

	// Info
	log.Println(p.Info())
}
