package main

import (
	"github.com/asticode/go-astibob"
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

	// Run Bob
	if err = bob.Run(ctx); err != nil {
		astilog.Fatal(errors.Wrap(err, "main: running bob failed"))
	}
}
