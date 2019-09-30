package main

import (
	"flag"

	"github.com/asticode/go-astibob"
	"github.com/asticode/go-astibob/abilities/text_to_speech"
	"github.com/asticode/go-astibob/abilities/text_to_speech/speak"
	"github.com/asticode/go-astibob/worker"
	"github.com/asticode/go-astilog"
	astiptr "github.com/asticode/go-astitools/ptr"
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

	// Say "Hello world" when the runnable has started
	w.On(astibob.DispatchConditions{
		From: &astibob.Identifier{
			Name:   astiptr.Str("Text to Speech"),
			Type:   astibob.RunnableIdentifierType,
			Worker: astiptr.Str("Worker #1"),
		},
		Name: astiptr.Str(astibob.RunnableStartedMessage),
	}, func(m *astibob.Message) (err error) {
		// Send message
		if err = w.SendMessages("Worker #1", "Text to Speech", text_to_speech.NewSayMessage("Hello world")); err != nil {
			err = errors.Wrap(err, "main: sending message failed")
			return
		}
		return
	})

	// Create speaker
	s := speak.New(speak.Options{})

	// Initialize speaker
	if err := s.Initialize(); err != nil {
		astilog.Fatal(errors.Wrap(err, "main: initializing speaker failed"))
	}
	defer s.Close()

	// Register runnables
	w.RegisterRunnables(worker.Runnable{
		AutoStart: true,
		Runnable:  text_to_speech.NewRunnable("Text to Speech", s),
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
