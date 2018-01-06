package astispeaking

import (
	"sync"

	"github.com/asticode/go-astilog"
	"github.com/go-ole/go-ole"
	"github.com/pkg/errors"
)

// Speaking represents an object capable of saying words to an audio output.
type Speaking struct {
	activated bool
	o         Options
	m         sync.Mutex

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

// Activate implements the astibrain.Activable interface
func (s *Speaking) Activate(a bool) {
	s.m.Lock()
	defer s.m.Unlock()
	s.activated = a
}

// Say says words
func (s *Speaking) Say(i string) (err error) {
	// Not activated
	s.m.Lock()
	activated := s.activated
	s.m.Unlock()
	if !activated {
		return
	}

	// Say
	astilog.Debugf("astispeaking: saying \"%s\"", i)
	if err = s.say(i); err != nil {
		err = errors.Wrapf(err, "saying \"%s\" failed", i)
		return
	}
	return
}
