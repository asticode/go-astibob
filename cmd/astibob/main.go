package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/asticode/go-astibob"
	"github.com/asticode/go-astilog"
	"github.com/asticode/go-astitools/config"
	"github.com/pkg/errors"
)

// Flags
var (
	ctx, cancel = context.WithCancel(context.Background())
	config      = flag.String("c", "", "the config path")
)

func main() {
	// Parse flags
	flag.Parse()
	astilog.FlagInit()

	// Create configuration
	c := newConfiguration()

	// Handle signals
	handleSignals()

	// Create bob
	bob, err := astibob.New(c.Bob)
	if err != nil {
		astilog.Fatal(errors.Wrap(err, "astibob: creating bob failed"))
	}
	defer bob.Close()

	// Run Bob
	if err = bob.Run(ctx); err != nil {
		astilog.Fatal(errors.Wrap(err, "astibob: running bob failed"))
	}
}

// Configuration represents a configuration
type Configuration struct {
	Bob astibob.Options `toml:"bob"`
}

// newConfiguration creates a new configuration
func newConfiguration() *Configuration {
	// Global config
	gc := &Configuration{
		Bob: astibob.Options{
			BrainsServer: astibob.ServerOptions{
				ListenAddr: "127.0.0.1:6970",
				Password:   "admin",
				PublicAddr: "127.0.0.1:6970",
				Timeout:    5 * time.Second,
				Username:   "admin",
			},
			ClientsServer: astibob.ServerOptions{
				ListenAddr: "127.0.0.1:6969",
				Password:   "admin",
				PublicAddr: "127.0.0.1:6969",
				Timeout:    5 * time.Second,
				Username:   "admin",
			},
			ResourcesDirectory: "resources",
		},
	}

	// Flag config
	fc := &Configuration{}

	// Build configuration
	c, err := asticonfig.New(gc, *config, fc)
	if err != nil {
		astilog.Fatal(err)
	}
	return c.(*Configuration)
}

func handleSignals() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch)
	go func() {
		for s := range ch {
			astilog.Debugf("astibob: received signal %s", s)
			if s == syscall.SIGABRT || s == syscall.SIGKILL || s == syscall.SIGINT || s == syscall.SIGQUIT || s == syscall.SIGTERM {
				cancel()
			}
		}
	}()
}
