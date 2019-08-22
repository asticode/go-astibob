package main

import (
	"flag"

	"github.com/asticode/go-astibob"
	"github.com/asticode/go-astibob/index"
	"github.com/asticode/go-astilog"
	"github.com/pkg/errors"
)

func main() {
	// Parse flags
	flag.Parse()
	astilog.FlagInit()

	// Create index
	i, err := index.New(index.Options{
		Server: astibob.ServerOptions{
			Addr:     "127.0.0.1:4000",
			Password: "admin",
			Username: "admin",
		},
	})
	if err != nil {
		astilog.Fatal(errors.Wrap(err, "main: creating index failed"))
	}
	defer i.Close()

	// Handle signals
	i.HandleSignals()

	// Serve
	i.Serve()

	// Blocking pattern
	i.Wait()
}
