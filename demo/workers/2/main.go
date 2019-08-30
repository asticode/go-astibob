package main

import (
	"flag"

	"github.com/asticode/go-astibob"
	"github.com/asticode/go-astibob/abilities/audio_input"
	"github.com/asticode/go-astibob/abilities/audio_input/portaudio"
	"github.com/asticode/go-astibob/worker"
	"github.com/asticode/go-astilog"
	"github.com/pkg/errors"
)

// Constants
const (
	sampleRate = 44100
)

func main() {
	// Parse flags
	flag.Parse()
	astilog.FlagInit()

	// Create worker
	w := worker.New("Worker #2", worker.Options{
		Index: astibob.ServerOptions{
			Addr:     "127.0.0.1:4000",
			Password: "admin",
			Username: "admin",
		},
		Server: astibob.ServerOptions{Addr: "127.0.0.1:4002"},
	})
	defer w.Close()

	// Create portaudio
	p := portaudio.New()

	// Initialize portaudio
	if err := p.Initialize(); err != nil {
		astilog.Fatal(errors.Wrap(err, "main: initializing portaudio failed"))
	}
	defer p.Close()

	// Create default stream
	s, err := p.NewDefaultStream(portaudio.StreamOptions{
		BitDepth:             32,
		BufferLength:         5000,
		MaxSilenceAudioLevel: 35 * 1e6,
		NumInputChannels:     1,
		SampleRate:           sampleRate,
	})
	if err != nil {
		astilog.Fatal(errors.Wrap(err, "main: creating default stream failed"))
	}

	// Register runnables
	w.RegisterRunnables(worker.Runnable{
		AutoStart: true,
		Runnable:  audio_input.NewRunnable("Audio input", s),
	})

	// Handle signals
	w.HandleSignals()

	// Serve
	w.Serve()

	// Register to index
	w.RegisterToIndex()

	// Blocking pattern
	w.Wait()
}
