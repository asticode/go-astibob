package astiexamples

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/asticode/go-astibob"
	"github.com/asticode/go-astibob/brain"
	"github.com/asticode/go-astilog"
)

// Context
var ctx, cancel = context.WithCancel(context.Background())

// Options
var (
	AbilityOptions = astibrain.AbilityOptions{
		AutoStart: true,
	}
	BobOptions = astibob.Options{
		BrainsServer: astibob.ServerOptions{
			ListenAddr:     "127.0.0.1:6970",
			MaxMessageSize: 512 * 1024,
			Password:       "admin",
			PublicAddr:     "127.0.0.1:6970",
			Timeout:        5 * time.Second,
			Username:       "admin",
		},
		ClientsServer: astibob.ServerOptions{
			ListenAddr:     "127.0.0.1:6969",
			MaxMessageSize: 4 * 1024,
			Password:       "admin",
			PublicAddr:     "127.0.0.1:6969",
			Timeout:        5 * time.Second,
			Username:       "admin",
		},
		ResourcesDirectory: "resources",
	}
	BrainOptions = astibrain.Options{
		Websocket: astibrain.WebsocketOptions{
			MaxMessageSize: 512 * 1024,
			Password:       "admin",
			URL:            "ws://127.0.0.1:6970/websocket",
			Username:       "admin",
		},
	}
)

// Init initializes the brain
func Init() context.Context {
	// Parse flags
	flag.Parse()
	astilog.FlagInit()

	// Handle signals
	handleSignals()
	return ctx
}

func handleSignals() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch)
	go func() {
		for s := range ch {
			astilog.Debugf("main: received signal %s", s)
			if s == syscall.SIGABRT || s == syscall.SIGKILL || s == syscall.SIGINT || s == syscall.SIGQUIT || s == syscall.SIGTERM {
				cancel()
			}
		}
	}()
}
