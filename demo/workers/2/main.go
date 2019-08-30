package main

import (
	"flag"
	"time"

	"github.com/asticode/go-astibob"
	"github.com/asticode/go-astibob/abilities/audio_input"
	"github.com/asticode/go-astibob/abilities/text_to_speech"
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

	// Register runnables
	w.RegisterRunnables(worker.Runnable{
		AutoStart: true,
		Runnable:  audio_input.NewRunnable("Audio input", nil),
	})

	// Handle signals
	w.HandleSignals()

	// Serve
	w.Serve()

	// Register to index
	w.RegisterToIndex()

	// TODO Testing
	go func() {
		time.Sleep(time.Second)
		w.SendCmds("Worker #1", "Text to Speech", text_to_speech.NewSayCmd("hello world"), text_to_speech.NewSayCmd("how are you today?"))
	}()

	// Blocking pattern
	w.Wait()
}
