package portaudio

import (
	"fmt"

	"github.com/asticode/go-astilog"
	"github.com/gordonklaus/portaudio"
	"github.com/pkg/errors"
)

type PortAudio struct{}

func New() *PortAudio {
	return &PortAudio{}
}

func (p *PortAudio) Initialize() (err error) {
	// Log
	astilog.Debug("portaudio: initializing portaudio")

	// Initialize
	if err = portaudio.Initialize(); err != nil {
		err = errors.Wrap(err, "portaudio: initializing portaudio failed")
		return
	}
	return
}

func (p *PortAudio) Close() (err error) {
	// Log
	astilog.Debug("astiportaudio: terminating portaudio")

	// Terminate
	if err = portaudio.Terminate(); err != nil {
		err = errors.Wrap(err, "astiportaudio: terminating portaudio failed")
		return
	}
	return
}
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
				s += fmt.Sprintf("|     |\n|     +--+ Device #%d: %s (sample rate: %.0fkHz - max input channels: %v)\n", idxDevice, d.Name, d.DefaultSampleRate, d.MaxInputChannels)
			}
		}
	}
	return
}
