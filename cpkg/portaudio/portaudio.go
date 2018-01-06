package astiportaudio

import (
	"fmt"

	"github.com/asticode/go-astilog"
	"github.com/gordonklaus/portaudio"
	"github.com/pkg/errors"
)

// PortAudio represents a portaudio wrapper
type PortAudio struct{}

// New creates a new portaudio wrapper and initializes it
func New() (p *PortAudio, err error) {
	astilog.Debug("astiportaudio: initializing portaudio")
	if err = portaudio.Initialize(); err != nil {
		err = errors.Wrap(err, "astiportaudio: initializing portaudio failed")
		return
	}
	return &PortAudio{}, nil
}

// Close implements the io.Closer interface
func (p *PortAudio) Close() (err error) {
	// Terminate portaudio
	astilog.Debug("astiportaudio: terminating portaudio")
	if err = portaudio.Terminate(); err != nil {
		err = errors.Wrap(err, "astiportaudio: terminating portaudio failed")
		return
	}
	return
}

// Info returns information about itself
func (p *PortAudio) Info() (s string) {
	// Get host APIs
	as, err := portaudio.HostApis()
	if err != nil {
		return "getting portaudio host apis failed"
	}

	// Loop through APIs
	s = "\n+ Portaudio\n"
	for idxAPI, a := range as {
		s += fmt.Sprintf("|\n+--+ Host API #%d: %s - %s\n", idxAPI, a.Name, a.Type)
		if a.DefaultInputDevice != nil {
			s += fmt.Sprintf("|  |\n|  +--+ Default input device: %s\n", a.DefaultInputDevice.Name)
		}
		if a.DefaultOutputDevice != nil {
			s += fmt.Sprintf("|  |\n|  +--+ Default output device: %s\n", a.DefaultOutputDevice.Name)
		}
		if len(a.Devices) > 0 {
			s += "|  |\n|  +--+ Devices:\n"
			for idxDevice, d := range a.Devices {
				s += fmt.Sprintf("|     |\n|     +--+ Device #%d: %s\n", idxDevice, d.Name)
			}
		}
	}
	return
}
