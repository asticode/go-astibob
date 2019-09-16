package main

import (
	"flag"

	"github.com/asticode/go-astibob"
	"github.com/asticode/go-astibob/abilities/audio_input"
	"github.com/asticode/go-astibob/abilities/speech_to_text"
	"github.com/asticode/go-astibob/abilities/speech_to_text/deepspeech"
	"github.com/asticode/go-astibob/worker"
	"github.com/asticode/go-astilog"
	"github.com/pkg/errors"
)

func main() {
	// Parse flags
	flag.Parse()
	astilog.FlagInit()

	// Create worker
	w := worker.New("Worker #3", worker.Options{
		Index: astibob.ServerOptions{
			Addr:     "127.0.0.1:4000",
			Password: "admin",
			Username: "admin",
		},
		Server: astibob.ServerOptions{Addr: "127.0.0.1:4003"},
	})
	defer w.Close()

	// Create deepspeech
	d := deepspeech.New(deepspeech.Options{
		AlphabetPath:   "demo/tmp/deepspeech/model/alphabet.txt",
		BeamWidth:      1024,
		ClientPath:     "demo/tmp/deepspeech/DeepSpeech/DeepSpeech.py",
		LMPath:         "demo/tmp/deepspeech/model/lm.binary",
		LMWeight:       0.75,
		ModelPath:      "demo/tmp/deepspeech/model/output_graph.pb",
		NCep:           26,
		NContext:       9,
		PrepareDirPath: "demo/tmp/deepspeech/prepare",
		TrainingArgs: map[string]string{
			"checkpoint_dir":    "demo/tmp/deepspeech/model/checkpoints",
			"export_dir":        "demo/tmp/deepspeech/model",
		},
		TriePath:             "demo/tmp/deepspeech/model/trie",
		ValidWordCountWeight: 1.85,
	})
	defer d.Close()

	// Initialize deepspeech
	if err := d.Init(); err != nil {
		astilog.Fatal(errors.Wrap(err, "main: initializing deepspeech failed"))
	}

	// Create runnable
	r := speech_to_text.NewRunnable("Speech to Text", d, speech_to_text.RunnableOptions{
		SpeechesDirPath: "demo/tmp/speech_to_text/speeches",
	})

	// Initialize runnable
	if err := r.Init(); err != nil {
		astilog.Fatal(errors.Wrap(err, "main: initializing runnable failed"))
	}
	defer r.Close()

	// Register runnables
	w.RegisterRunnables(worker.Runnable{
		Runnable: r,
	})

	// Register listenables
	w.RegisterListenables(
		worker.Listenable{
			Listenable: audio_input.NewListenable(audio_input.ListenableOptions{
				OnSamples: func(from astibob.Identifier, samples []int32, bitDepth int, sampleRate, maxSilenceAudioLevel float64) (err error) {
					// Send message
					if err = w.SendMessages("Worker #3", "Speech to Text", speech_to_text.NewSamplesMessage(
						from,
						samples,
						bitDepth,
						sampleRate,
						maxSilenceAudioLevel,
					)); err != nil {
						err = errors.Wrap(err, "main: sending message failed")
						return
					}
					return
				},
			}),
			Runnable: "Audio input",
			Worker:   "Worker #2",
		},
		worker.Listenable{
			Listenable: speech_to_text.NewListenable(speech_to_text.ListenableOptions{
				OnText: func(from astibob.Identifier, text string) (err error) {
					astilog.Warnf("main: on text: worker: %s - runnable: %s - text: %s", *from.Name, *from.Worker, text)
					return
				},
			}),
			Runnable: "Speech to Text",
			Worker:   "Worker #3",
		},
	)

	// Handle signals
	w.HandleSignals()

	// Serve
	w.Serve()

	// Register to index
	w.RegisterToIndex()

	// Blocking pattern
	w.Wait()
}
