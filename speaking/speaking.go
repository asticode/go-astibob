package astispeaking

import (
	"context"
	"sync"

	"time"

	"github.com/asticode/go-astilog"
	"github.com/go-ole/go-ole"
	"github.com/pkg/errors"
)

// Speaking represents an object capable of saying words to an audio output
type Speaking struct {
	isMuted bool
	m       sync.Mutex

	// Windows
	windowsIDispatch *ole.IDispatch
	windowsIUnknown  *ole.IUnknown
}

// New creates a new speaking
func New() *Speaking {
	return &Speaking{}
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

// Say says words
func (s *Speaking) Say(i string) error {
	astilog.Debugf("astispeaking: saying \"%s\"", i)
	return s.say(i)
}

// Run implements the astibob.Ability interface
func (s *Speaking) Run(ctx context.Context) (err error) {
	// Handle muted attributed
	s.Unmute()
	defer s.Mute()

	go func() {
		time.Sleep(time.Second)
		s.Say("Hello quentin")
	}()

	// Wait for context to be done
	<-ctx.Done()
	if ctx.Err() != nil {
		err = errors.Wrap(err, "astispeaking: context error")
		return
	}
	return
}
