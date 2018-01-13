package astiunderstanding

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/asticode/go-astibob/brain"
	"github.com/asticode/go-astilog"
	"github.com/asticode/go-astitools/audio"
	"github.com/asticode/go-astitools/sync"
	"github.com/asticode/go-astiws"
	"github.com/cryptix/wav"
	"github.com/pkg/errors"
)

// Ability represents an object capable of doing speech to text analysis
type Ability struct {
	ch           chan PayloadSamples
	d            *astisync.Do
	dispatchFunc astibrain.DispatchFunc
	o            AbilityOptions
	p            SpeechParser
	sd           *astiaudio.SilenceDetector
}

// AbilityOptions represents ability options
type AbilityOptions struct {
	SamplesDirectoryPath *string                          `toml:"samples_directory_path"`
	SilenceDetector      astiaudio.SilenceDetectorOptions `toml:"silence_detector"`
}

// NewAbility creates a new ability
func NewAbility(p SpeechParser, o AbilityOptions) *Ability {
	return &Ability{
		d:  astisync.NewDo(),
		o:  o,
		p:  p,
		sd: astiaudio.NewSilenceDetector(o.SilenceDetector),
	}
}

// SetDispatchFunc implements the astibrain.Dispatcher interface
func (a *Ability) SetDispatchFunc(fn astibrain.DispatchFunc) {
	a.dispatchFunc = fn
}

// Name implements the astibrain.Ability interface
func (a *Ability) Name() string {
	return Name
}

// Run implements the astibrain.Runnable interface
func (a *Ability) Run(ctx context.Context) (err error) {
	// Reset
	a.ch = make(chan PayloadSamples)
	a.sd.Reset()

	// Listen
	for {
		select {
		case p := <-a.ch:
			// Add samples to silence detector and retrieve speech samples
			speechSamples := a.sd.Add(p.Samples, p.SampleRate, p.SilenceMaxAudioLevel)

			// No speech samples
			if len(speechSamples) <= 0 {
				continue
			}

			// Process samples
			for _, samples := range speechSamples {
				a.processSamples(samples, p.SampleRate, p.SignificantBits)
			}
		case <-ctx.Done():
			err = errors.Wrap(err, "astiunderstanding: context error")
			return
		}
	}
}

// processSamples processes samples
func (a *Ability) processSamples(samples []int32, sampleRate, significantBits int) {
	// Make sure the following is not blocking but still executed in FIFO order
	a.d.Do(func() {
		// Execute speech to text analysis
		start := time.Now()
		astilog.Debugf("astiunderstanding: starting speech to text analysis on %d samples", len(samples))
		s := a.p.SpeechToText(samples, len(samples), sampleRate, significantBits)
		astilog.Debugf("astiunderstanding: speech to text analysis done in %s", time.Now().Sub(start))

		// Store everything for later validation
		path, err := a.storeSamples(samples, sampleRate, significantBits)
		if err != nil {
			astilog.Error(errors.Wrap(err, "astiunderstanding: storing samples failed"))
		}
		if len(path) > 0 {
			astilog.Debugf("astiunderstanding: samples have been stored to %s", path)
		}

		// Dispatch
		a.dispatchFunc(astibrain.Event{
			AbilityName: Name,
			Name:        websocketEventNameAnalysis,
			Payload:     s,
		})
	})
}

// storeSamples stores the samples for later validation
// TODO Add option to stop storing samples from UI
// TODO Split samples in 2 folders => to validate, and validated
// TODO Add validation process in UI
// TODO Store max quality samples (more than 16 000 sample rate)
func (a *Ability) storeSamples(samples []int32, sampleRate, significantBits int) (path string, err error) {
	// No need to store samples
	if a.o.SamplesDirectoryPath == nil {
		return
	}

	// Create dir path
	now := time.Now()
	path = filepath.Join(*a.o.SamplesDirectoryPath, now.Format("2006-01-02"))

	// Make sure the dir exists
	if err = os.MkdirAll(path, 0777); err != nil {
		err = errors.Wrapf(err, "astiunderstanding: mkdirall %s failed", path)
		return
	}

	// Add filename to dir path
	path = filepath.Join(path, now.Format("15-04-05")+".wav")

	// Create file
	var f *os.File
	if f, err = os.Create(path); err != nil {
		err = errors.Wrapf(err, "astiunderstanding: creating %s failed", path)
		return
	}
	defer f.Close()

	// Create wav writer
	wf := wav.File{
		Channels:        1,
		SampleRate:      uint32(sampleRate),
		SignificantBits: uint16(significantBits),
	}
	var w *wav.Writer
	if w, err = wf.NewWriter(f); err != nil {
		err = errors.Wrap(err, "astiunderstanding: creating wav writer failed")
		return
	}
	defer w.Close()

	// Write
	for _, sample := range samples {
		if err = w.WriteInt32(sample); err != nil {
			err = errors.Wrap(err, "astiunderstanding: writing wav sample failed")
			return
		}
	}
	return
}

// WebsocketListeners implements the astibrain.WebsocketListener interface
func (a *Ability) WebsocketListeners() map[string]astiws.ListenerFunc {
	return map[string]astiws.ListenerFunc{
		websocketEventNameSamples: a.websocketListenerSamples,
	}
}

// websocketListenerSamples listens to the samples websocket event
func (a *Ability) websocketListenerSamples(c *astiws.Client, eventName string, payload json.RawMessage) error {
	// Unmarshal payload
	var p PayloadSamples
	if err := json.Unmarshal(payload, &p); err != nil {
		astilog.Error(errors.Wrapf(err, "astiunderstanding: json unmarshaling %s into %#v failed", payload, p))
		return nil
	}

	// Dispatch
	a.ch <- p
	return nil
}
