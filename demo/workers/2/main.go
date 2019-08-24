package main

import (
	"flag"

	"github.com/asticode/go-astibob"
	"github.com/asticode/go-astibob/abilities/speak"
	"github.com/asticode/go-astibob/worker"
	"github.com/asticode/go-astilog"
)

func main() {
	// Parse flags
	flag.Parse()
	astilog.FlagInit()

	// Create worker
	w := worker.New("Worker #2", worker.Options{
		Index: astibob.ServerOptions{
			Addr:     "127.0.0.1:4000",
			Password: "admin",
			Username: "admin",
		},
	})
	defer w.Close()

	// Handle signals
	w.HandleSignals()

	// Register runnables
	w.RegisterRunnables(speak.NewRunnable("Speak 1"))
	w.RegisterRunnables(speak.NewRunnable("Speak 2"))

	// Register to index
	w.RegisterToIndex()

	// Blocking pattern
	w.Wait()
}
