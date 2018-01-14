package astiunderstanding

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"fmt"

	"github.com/asticode/go-astibob/brain"
	"github.com/asticode/go-astilog"
	"github.com/asticode/go-astitools/sync"
	"github.com/asticode/go-astiws"
	"github.com/cryptix/wav"
	"github.com/pkg/errors"
	"github.com/rs/xid"
)

// Ability represents an object capable of doing speech to text analysis
type Ability struct {
	c            AbilityConfiguration
	ch           chan PayloadSamples
	d            *astisync.Do
	dispatchFunc astibrain.DispatchFunc
	p            SpeechParser
	sd           SilenceDetector
}

// AbilityConfiguration represents an ability configuration
type AbilityConfiguration struct {
	SamplesDirectory string `toml:"samples_directory"`
	StoreSamples     bool   `toml:"store_samples"`
}

// NewAbility creates a new ability
func NewAbility(p SpeechParser, sd SilenceDetector, c AbilityConfiguration) (a *Ability, err error) {
	// Create
	a = &Ability{
		c:  c,
		d:  astisync.NewDo(),
		p:  p,
		sd: sd,
	}

	// Absolute paths
	if len(a.c.SamplesDirectory) > 0 {
		if a.c.SamplesDirectory, err = filepath.Abs(a.c.SamplesDirectory); err != nil {
			err = errors.Wrapf(err, "astiunderstanding: filepath abs of %s failed", a.c.SamplesDirectory)
			return
		}
	}
	return
}

// SetDispatchFunc implements the astibrain.Dispatcher interface
func (a *Ability) SetDispatchFunc(fn astibrain.DispatchFunc) {
	a.dispatchFunc = fn
}

// Name implements the astibrain.Ability interface
func (a *Ability) Name() string {
	return name
}

// Description implements the astibrain.Ability interface
func (a *Ability) Description() string {
	return "Executes a speech to text analysis on audio samples"
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
		text := a.p.SpeechToText(samples, len(samples), sampleRate, significantBits)
		astilog.Debugf("astiunderstanding: speech to text analysis done in %s", time.Now().Sub(start))

		// Dispatch analysis
		if len(text) > 0 {
			a.dispatchFunc(astibrain.Event{
				AbilityName: name,
				Name:        websocketEventNameAnalysis,
				Payload:     text,
			})
		}

		// Check if samples have to be stored
		if a.c.StoreSamples && len(a.c.SamplesDirectory) > 0 {
			// Store samples
			id, err := a.storeSamples(text, samples, sampleRate, significantBits)
			if err != nil {
				astilog.Error(errors.Wrap(err, "astiunderstanding: storing samples failed"))
			} else {
				a.dispatchFunc(astibrain.Event{
					AbilityName: name,
					Name:        websocketEventNameSamplesStored,
					Payload:     newPayloadStoredSamples(id, text),
				})
			}
		}
	})
}

// PayloadStoredSamples represents stored samples payload
type PayloadStoredSamples struct {
	ID            string `json:"id"`
	Text          string `json:"text"`
	WavStaticPath string `json:"wav_static_path"`
}

// newPayloadStoredSamples creates a new stored samples payload
func newPayloadStoredSamples(id, text string) PayloadStoredSamples {
	return PayloadStoredSamples{
		ID:            id,
		Text:          text,
		WavStaticPath: fmt.Sprintf("/samples%s.wav", id),
	}
}

// samplesToBeValidatedDirectory returns the directory containing samples to be validated
func samplesToBeValidatedDirectory(samplesDirectory string) string {
	return filepath.Join(samplesDirectory, "to_be_validated")
}

// samplesValidatedDirectory returns the directory containing validated samples
func samplesValidatedDirectory(samplesDirectory string) string {
	return filepath.Join(samplesDirectory, "validated")
}

// storeSamples stores the samples for later validation
func (a *Ability) storeSamples(text string, samples []int32, sampleRate, significantBits int) (id string, err error) {
	// Create id
	id = filepath.Join(time.Now().Format("2006-01-02"), xid.New().String())

	// Store samples wav
	if err = a.storeSamplesWav(id, samples, sampleRate, significantBits); err != nil {
		err = errors.Wrap(err, "astiunderstanding: storing samples wav failed")
		return
	}

	// Store samples txt
	if err = a.storeSamplesTxt(id, text); err != nil {
		err = errors.Wrap(err, "astiunderstanding: storing samples txt failed")
		return
	}
	return
}

// storeSamplesWav stores the samples as a wav file
func (a *Ability) storeSamplesWav(id string, samples []int32, sampleRate, significantBits int) (err error) {
	// Create wav path
	wavPath := filepath.Join(samplesToBeValidatedDirectory(a.c.SamplesDirectory), id+".wav")

	// Create dir
	if err = os.MkdirAll(filepath.Dir(wavPath), 0755); err != nil {
		err = errors.Wrapf(err, "astiunderstanding: mkdirall %s failed", filepath.Dir(wavPath))
		return
	}

	// Create wav file
	var f *os.File
	if f, err = os.Create(wavPath); err != nil {
		err = errors.Wrapf(err, "astiunderstanding: creating %s failed", wavPath)
		return
	}
	defer f.Close()

	// Create wav writer
	wf := wav.File{
		Channels:        1,
		SampleRate:      uint32(sampleRate),
		SignificantBits: uint16(significantBits),
	}
	var r *wav.Writer
	if r, err = wf.NewWriter(f); err != nil {
		err = errors.Wrap(err, "astiunderstanding: creating wav writer failed")
		return
	}
	defer r.Close()

	// Write wav samples
	for _, sample := range samples {
		if err = r.WriteInt32(sample); err != nil {
			err = errors.Wrap(err, "astiunderstanding: writing wav sample failed")
			return
		}
	}
	return
}

// storeSamplesTxt stores the samples information in a txt file
func (a *Ability) storeSamplesTxt(id, text string) (err error) {
	// Create txt path
	txtPath := filepath.Join(samplesToBeValidatedDirectory(a.c.SamplesDirectory), id+".txt")

	// Create txt file
	var f *os.File
	if f, err = os.Create(txtPath); err != nil {
		err = errors.Wrapf(err, "astiunderstanding: creating %s failed", txtPath)
		return
	}
	defer f.Close()

	// Write data
	if _, err = f.Write([]byte(text)); err != nil {
		err = errors.Wrap(err, "astiunderstanding: writing text failed")
		return
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
