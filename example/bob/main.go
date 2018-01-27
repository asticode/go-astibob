package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"strings"

	"github.com/asticode/go-astibob"
	"github.com/asticode/go-astibob/abilities/hearing"
	"github.com/asticode/go-astibob/abilities/speaking"
	"github.com/asticode/go-astibob/abilities/understanding"
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

	// Create bob
	bob, err := astibob.New(astibob.Configuration{
		BrainsServer: astibob.ServerConfiguration{
			ListenAddr: "127.0.0.1:6970",
			Password:   "admin",
			Username:   "admin",
		},
		ClientsServer: astibob.ServerConfiguration{
			ListenAddr: "127.0.0.1:6969",
			Password:   "admin",
			Username:   "admin",
		},
		ResourcesDirectory: "resources",
	})
	if err != nil {
		astilog.Fatal(errors.Wrap(err, "main: creating bob failed"))
	}
	defer bob.Close()

	// Create interfaces
	hearing := astihearing.NewInterface(astihearing.InterfaceConfiguration{})
	speaking := astispeaking.NewInterface()
	understanding, err := astiunderstanding.NewInterface(astiunderstanding.InterfaceConfiguration{SamplesDirectory: "example/tmp/understanding"})
	if err != nil {
		astilog.Fatal(errors.Wrap(err, "main: creating understanding failed"))
	}

	// Declare interfaces
	bob.Declare(hearing)
	bob.Declare(speaking)
	bob.Declare(understanding)

	// Handle ability start
	bob.On(astibob.EventNameAbilityStarted, func(e astibob.Event) bool {
		if e.Ability != nil && e.Ability.Name == speaking.Name() {
			if err := bob.Exec(speaking.Say("Hello")); err != nil {
				astilog.Error(errors.Wrap(err, "main: executing cmd failed"))
			}
		}
		return false
	})

	// Handle samples
	hearing.OnSamples(func(brainName string, samples []int32, sampleRate, significantBits int, silenceMaxAudioLevel float64) error {
		// Send samples
		bob.Exec(understanding.Samples(samples, sampleRate, significantBits, silenceMaxAudioLevel))
		return nil
	})

	// Add analysis
	understanding.OnAnalysis(func(brainName, text string) error {
		astilog.Debugf("main: processing analysis <%s>", text)
		if strings.TrimSpace(text) == "bob" {
			if err := bob.Exec(speaking.Say("Yes")); err != nil {
				astilog.Error(errors.Wrap(err, "main: executing cmd failed"))
			}
		}
		return nil
	})

	// Run Bob
	if err = bob.Run(ctx); err != nil {
		astilog.Fatal(errors.Wrap(err, "main: running bob failed"))
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
