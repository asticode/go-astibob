package main

import (
	"flag"

	"github.com/asticode/go-astibob"
	"github.com/asticode/go-astibob/abilities/audio_input"
	"github.com/asticode/go-astibob/abilities/text_to_speech"
	"github.com/asticode/go-astibob/abilities/text_to_speech/speak"
	"github.com/asticode/go-astibob/worker"
	"github.com/asticode/go-astilog"
	"github.com/pkg/errors"
)

func main() {
	// Parse flags
	flag.Parse()
	astilog.FlagInit()

	// Create worker
	w := worker.New("Worker #1", worker.Options{
		Index: astibob.ServerOptions{
			Addr:     "127.0.0.1:4000",
			Password: "admin",
			Username: "admin",
		},
		Server: astibob.ServerOptions{Addr: "127.0.0.1:4001"},
	})
	defer w.Close()

	// Handle signals
	w.HandleSignals()

	// Create speaker
	s := speak.New(speak.Options{})

	// Init speaker
	if err := s.Init(); err != nil {
		astilog.Fatal(errors.Wrap(err, "main: initializing speaker failed"))
	}
	defer s.Close()

	// Register runnables
	w.RegisterRunnables(worker.Runnable{
		AutoStart: true,
		Runnable:  text_to_speech.NewRunnable("Text to Speech", s),
	})

	// Register listenables
	w.RegisterListenables(worker.Listenable{
		Listenable: audio_input.NewListenable(audio_input.ListenableOptions{
			OnSamples: func(samples []int32, sampleRate, significantBits int, silenceMaxAudioLevel float64) (err error) {
				astilog.Warnf("samples: %+v - sample rate: %v - significant bits: %v - silence max audio level: %v", samples, sampleRate, significantBits, silenceMaxAudioLevel)
				return
			},
		}),
		Runnable: "Audio input",
		Worker:   "Worker #2",
	})

	// Serve
	w.Serve()

	// Register to index
	w.RegisterToIndex()

	// Blocking pattern
	w.Wait()
}
