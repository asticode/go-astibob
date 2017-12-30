package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/asticode/go-astibob"
	"github.com/asticode/go-astibob/hearing"
	"github.com/asticode/go-astibob/portaudio"
	"github.com/asticode/go-astibob/speaking"
	"github.com/asticode/go-astilog"
	"github.com/asticode/go-astitools/config"
	"github.com/pkg/errors"
)

// Flags
var (
	autoStart          = flag.Bool("as", false, "triggers the auto start")
	ctx, cancel        = context.WithCancel(context.Background())
	config             = flag.String("c", "", "the config path")
	resourcesDirectory = flag.String("r", "", "the resources directory path")
	serverAddr         = flag.String("a", "", "the server addr")
	serverPassword     = flag.String("p", "", "the server password")
	serverTimeout      = flag.Duration("t", 0, "the server timeout")
	serverUsername     = flag.String("u", "", "the server username")
)

func main() {
	// Parse flags
	flag.Parse()
	astilog.FlagInit()

	// Init configuration
	c := newConfiguration()

	// Handle signals
	handleSignals()

	// Init portaudio
	p, err := astiportaudio.New()
	if err != nil {
		astilog.Fatal(errors.Wrap(err, "astibob: creating portaudio failed"))
	}
	defer p.Close()

	// Init portaudio stream
	s, err := p.NewDefaultStream(make([]int32, 192), c.PortAudio)
	if err != nil {
		astilog.Fatal(errors.Wrap(err, "astibob: creating portaudio default stream failed"))
	}
	defer s.Close()

	// Init hearing
	hearing := astihearing.New(s, c.Hearing)

	// Init speaking
	speaking := astispeaking.New(c.Speaking)

	// Init bob
	bob := astibob.New(c.Bob)
	defer bob.Close()

	// Learn abilities
	bob.Learn("Hearing", hearing, astibob.AbilityOptions{AutoStart: c.AutoStart})
	bob.Learn("Speaking", speaking, astibob.AbilityOptions{AutoStart: c.AutoStart})

	// Run Bob
	if err = bob.Run(ctx); err != nil {
		astilog.Fatal(errors.Wrap(err, "astibob: running bob failed"))
	}
}

// Configuration represents a configuration
type Configuration struct {
	AutoStart bool                        `toml:"auto_start"`
	Bob       astibob.Options             `toml:"bob"`
	Hearing   astihearing.Options         `toml:"hearing"`
	PortAudio astiportaudio.StreamOptions `toml:"portaudio"`
	Speaking  astispeaking.Options        `toml:"speaking"`
}

// newConfiguration creates a new configuration
func newConfiguration() *Configuration {
	// Global config
	gc := &Configuration{
		Bob: astibob.Options{
			ServerAddr:         "127.0.0.1:6969",
			ServerPassword:     "admin",
			ServerTimeout:      5 * time.Second,
			ServerUsername:     "admin",
			ResourcesDirectory: "resources",
		},
		Hearing: astihearing.Options{
			WorkingDirectory: filepath.Join(os.TempDir(), "bob", "hearing"),
		},
		PortAudio: astiportaudio.StreamOptions{
			NumInputChannels: 1,
			SampleRate:       16000,
		},
		Speaking: astispeaking.Options{
			BinaryPath: "espeak",
		},
	}

	// Flag config
	fc := &Configuration{
		AutoStart: *autoStart,
		Bob: astibob.Options{
			ServerAddr:         *serverAddr,
			ServerPassword:     *serverPassword,
			ServerTimeout:      *serverTimeout,
			ServerUsername:     *serverUsername,
			ResourcesDirectory: *resourcesDirectory,
		},
	}

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
