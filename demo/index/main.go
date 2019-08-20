package main

import (
	"flag"

	"github.com/asticode/go-astibob"
	"github.com/asticode/go-astibob/index"
	"github.com/asticode/go-astilog"
)

func main() {
	// Parse flags
	flag.Parse()
	astilog.FlagInit()

	// Create index
	i := index.New(index.Options{
		Server: astibob.ServerOptions{
			Addr:     "127.0.0.1:4000",
			Password: "admin",
			Username: "admin",
		},
	})
	defer i.Close()

	// Handle signals
	i.HandleSignals()

	// Serve
	i.Serve()

	// Blocking pattern
	i.Wait()
}
