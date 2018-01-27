package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/asticode/go-astibob/abilities/keyboarding"
	"github.com/asticode/go-astibob/abilities/mousing"
	"github.com/asticode/go-astibob/brain"
	"github.com/asticode/go-astibob/pkg/robotgo"
	"github.com/asticode/go-astilog"
	"github.com/pkg/errors"
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
		Name: "Brain #3",
		Websocket: astibrain.WebsocketConfiguration{
			Password: "admin",
			URL:      "ws://127.0.0.1:6970/websocket",
			Username: "admin",
		},
	})
	defer brain.Close()

	// Create keyboarder
	keyboarder := astirobotgo.NewKeyboarder()

	// Create mouser
	mouser := astirobotgo.NewMouser()

	// Create keyboarding
	keyboarding := astikeyboarding.NewAbility(keyboarder)

	// Create mousing
	mousing := astimousing.NewAbility(mouser)

	// Learn abilities
	brain.Learn(keyboarding, astibrain.AbilityConfiguration{})
	brain.Learn(mousing, astibrain.AbilityConfiguration{})

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
