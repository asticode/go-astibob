package main

import (
	"github.com/asticode/go-astibob"
	"github.com/asticode/go-astibob/abilities/speaking"
	"github.com/asticode/go-astibob/examples"
	"github.com/asticode/go-astilog"
	"github.com/pkg/errors"
)

func main() {
	// Base init
	ctx := astiexamples.Init()

	// Create bob
	bob, err := astibob.New(astiexamples.BobOptions)
	if err != nil {
		astilog.Fatal(errors.Wrap(err, "main: creating bob failed"))
	}
	defer bob.Close()

	// Create speaking
	speaking := astispeaking.NewInterface()

	// Add listeners
	bob.On(astibob.EventNameAbilityStarted, func(e astibob.Event) bool {
		if e.Ability != nil && e.Ability.Name == astispeaking.Name {
			if err := bob.Exec(speaking.Say("I love you Bob")); err != nil {
				astilog.Error(errors.Wrap(err, "main: executing cmd failed"))
			}
		}
		return false
	})

	// Run Bob
	if err = bob.Run(ctx); err != nil {
		astilog.Fatal(errors.Wrap(err, "main: running bob failed"))
	}
}
