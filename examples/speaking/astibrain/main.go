package main

import (
	"github.com/asticode/go-astibob/abilities/speaking"
	"github.com/asticode/go-astibob/brain"
	"github.com/asticode/go-astibob/examples"
	"github.com/asticode/go-astilog"
	"github.com/pkg/errors"
)

func main() {
	// Base init
	ctx := astiexamples.Init()

	// Create brain
	brain := astibrain.New(astiexamples.BrainOptions)
	defer brain.Close()

	// Create speaking
	speaking := astispeaking.NewAbility(astispeaking.AbilityOptions{})

	// Learn ability
	brain.Learn(speaking, astiexamples.AbilityOptions)

	// Run the brain
	if err := brain.Run(ctx); err != nil {
		astilog.Fatal(errors.Wrap(err, "main: running brain failed"))
	}
}
