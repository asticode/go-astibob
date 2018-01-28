package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/asticode/go-astibob/abilities/hearing"
	"github.com/asticode/go-astibob/abilities/understanding"
	"github.com/asticode/go-astibob/brain"
	"github.com/asticode/go-astibob/pkg/portaudio"
	"github.com/asticode/go-astibob/pkg/speechtotext"
	"github.com/asticode/go-astilog"
	"github.com/asticode/go-astitools/audio"
	"github.com/pkg/errors"
)

// Constants
const (
	sampleRate             = 44100
	understandingDirectory = "demo/tmp/understanding"
)

// Context
var ctx, cancel = context.WithCancel(context.Background())

func main() {
	// Parse flags
	flag.Parse()
	astilog.FlagInit()

	// Handle signals
	handleSignals()

	// Create portaudio
	p, err := astiportaudio.New()
	if err != nil {
		astilog.Fatal(errors.Wrap(err, "main: creating portaudio failed"))
	}
	defer p.Close()

	// Create portaudio stream
	s, err := p.NewDefaultStream(make([]int32, 192), astiportaudio.StreamOptions{
		NumInputChannels: 1,
		SampleRate:       sampleRate,
	})
	if err != nil {
		astilog.Fatal(errors.Wrap(err, "main: creating portaudio default stream failed"))
	}
	defer s.Close()

	// Create silence detector
	sd := func() astiunderstanding.SilenceDetector {
		return astiaudio.NewSilenceDetector(astiaudio.SilenceDetectorConfiguration{})
	}

	// Create speech to text
	stt := astispeechtotext.New(astispeechtotext.Configuration{
		AlphabetConfigPath: "demo/alphabet.txt",
		BeamWidth:          500,
		ModelPath:          understandingDirectory + "/deepspeech/export/output_graph.pb",
		NCep:               26,
		NContext:           9,
	})

	// Create brain
	brain := astibrain.New(astibrain.Configuration{
		Name: "Brain #2",
		Websocket: astibrain.WebsocketConfiguration{
			Password: "admin",
			URL:      "ws://127.0.0.1:6970/websocket",
			Username: "admin",
		},
	})
	defer brain.Close()

	// Create hearing
	hearing := astihearing.NewAbility(s, astihearing.AbilityConfiguration{
		SampleRate:           sampleRate,
		SignificantBits:      32,
		SilenceMaxAudioLevel: 35 * 1e6,
	})

	// Create understanding
	understanding, err := astiunderstanding.NewAbility(stt, sd, astiunderstanding.AbilityConfiguration{
		SamplesDirectory: understandingDirectory,
	})
	if err != nil {
		astilog.Fatal(errors.Wrap(err, "main: creating understanding failed"))
	}

	// Learn abilities
	brain.Learn(hearing, astibrain.AbilityConfiguration{})
	brain.Learn(understanding, astibrain.AbilityConfiguration{})

	// Run the brain
	if err := brain.Run(ctx); err != nil {
		astilog.Fatal(errors.Wrap(err, "main: running brain failed"))
	}
}

func handleSignals() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch)
	go func() {
		for s := range ch {
			astilog.Debugf("main: received signal %s", s)
			if s == syscall.SIGABRT || s == syscall.SIGINT || s == syscall.SIGQUIT || s == syscall.SIGTERM {
				cancel()
			}
		}
	}()
}
