package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/asticode/go-astibob/abilities/speaking"
	"github.com/asticode/go-astibob/brain"
	"github.com/asticode/go-astilog"
	"github.com/pkg/errors"
	"github.com/asticode/go-astibob/pkg/speak"
)

// Context
var ctx, cancel = context.WithCancel(context.Background())

func main() {
	// Parse flags
	flag.Parse()
	astilog.FlagInit()

	// Handle signals
	handleSignals()

	// Create brain
	brain := astibrain.New(astibrain.Configuration{
		Name: "Brain #1",
		Websocket: astibrain.WebsocketConfiguration{
			Password: "admin",
			URL:      "ws://127.0.0.1:6970/websocket",
			Username: "admin",
		},
	})
	defer brain.Close()

	// Create speaker
	s := astispeak.New(astispeak.Configuration{})

	// Init speaker
	if err := s.Init(); err != nil {
		astilog.Fatal(errors.Wrap(err, "main: initializing speaker failed"))
	}
	defer s.Close()

	// Create speaking
	speaking := astispeaking.NewAbility(s)

	// Learn ability
	brain.Learn(speaking, astibrain.AbilityConfiguration{})

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
