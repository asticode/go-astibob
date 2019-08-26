package main

import (
	"flag"
	"time"

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
		Server: astibob.ServerOptions{Addr: "127.0.0.1:4002"},
	})
	defer w.Close()

	// Handle signals
	w.HandleSignals()

	// Serve
	w.Serve()

	// Register to index
	w.RegisterToIndex()

	// TODO Testing
	go func() {
		time.Sleep(time.Second)
		w.SendCmds("Worker #1", "Speak", speak.NewSayCmd("hello world"), speak.NewSayCmd("how are you today?"))
	}()

	// Blocking pattern
	w.Wait()
}
