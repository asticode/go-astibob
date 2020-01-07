package portaudio

import (
	"fmt"

	"github.com/asticode/go-astikit"
	"github.com/gordonklaus/portaudio"
)

type PortAudio struct {
	l astikit.SeverityLogger
}

func New(l astikit.StdLogger) *PortAudio {
	return &PortAudio{l: astikit.AdaptStdLogger(l)}
}

func (p *PortAudio) Initialize() (err error) {
	// Log
	p.l.Debug("portaudio: initializing portaudio")

	// Initialize
	if err = portaudio.Initialize(); err != nil {
		err = fmt.Errorf("portaudio: initializing portaudio failed: %w", err)
		return
	}
	return
}

func (p *PortAudio) Close() (err error) {
	// Log
	p.l.Debug("astiportaudio: terminating portaudio")

	// Terminate
	if err = portaudio.Terminate(); err != nil {
		err = fmt.Errorf("astiportaudio: terminating portaudio failed: %w", err)
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
