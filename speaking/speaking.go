package astispeaking

import (
	"context"
	"sync"

	"time"

	"github.com/asticode/go-astilog"
	"github.com/go-ole/go-ole"
	"github.com/pkg/errors"
)

// Speaking represents an object capable of saying words to an audio output.
type Speaking struct {
	isMuted bool
	o       Options
	m       sync.Mutex

	// Windows
	windowsIDispatch *ole.IDispatch
	windowsIUnknown  *ole.IUnknown
}

// Options represents speaking options.
type Options struct {
	BinaryPath string `toml:"binary_path"`
	Voice      string `toml:"voice"`
}

// New creates a new speaking
func New(o Options) *Speaking {
	return &Speaking{
		o: o,
	}
}

// Unmute unmutes the speaker
func (s *Speaking) Unmute() {
	s.m.Lock()
	defer s.m.Unlock()
	s.isMuted = false
}

// Mute mutes the speaker
func (s *Speaking) Mute() {
	s.m.Lock()
	defer s.m.Unlock()
	s.isMuted = true
}

// Run implements the astibob.Ability interface
func (s *Speaking) Run(ctx context.Context) (err error) {
	// Handle muted attributed
	s.Unmute()
	defer s.Mute()

	go func() {
		time.Sleep(time.Second)
		s.Say("Bonjour quentin, comment vas-tu aujourd'hui?")
	}()

	// Wait for context to be done
	<-ctx.Done()
	if ctx.Err() != nil {
		err = errors.Wrap(err, "astispeaking: context error")
		return
	}
	return
}

// Say says words
func (s *Speaking) Say(i string) (err error) {
	astilog.Debugf("astispeaking: saying \"%s\"", i)
	if err = s.say(i); err != nil {
		err = errors.Wrapf(err, "saying \"%s\" failed", i)
		return
	}
	return
}
