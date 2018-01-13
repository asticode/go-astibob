package astiunderstanding

import (
	"encoding/json"

	"github.com/asticode/go-astibob"
	"github.com/asticode/go-astilog"
	"github.com/asticode/go-astiws"
	"github.com/pkg/errors"
)

// Interface is the interface of the ability
type Interface struct {
	onAnalysis []AnalysisFunc
}

// AnalysisFunc represents the callback executed upon receiving results of an analysis
type AnalysisFunc func(text string) error

// PayloadSamples represents the samples payload
type PayloadSamples struct {
	SampleRate           int     `json:"sample_rate"`
	Samples              []int32 `json:"samples"`
	SignificantBits      int     `json:"significant_bits"`
	SilenceMaxAudioLevel float64 `json:"silence_max_audio_level"`
}

// NewInterface creates a new interface
func NewInterface() *Interface {
	return &Interface{}
}

// Name implements the astibob.Interface interface
func (i *Interface) Name() string {
	return Name
}

// Samples creates a samples cmd
func (i *Interface) Samples(samples []int32, sampleRate, significantBits int, silenceMaxAudioLevel float64) *astibob.Cmd {
	return &astibob.Cmd{
		AbilityName: Name,
		EventName:   websocketEventNameSamples,
		Payload: PayloadSamples{
			SampleRate:           sampleRate,
			Samples:              samples,
			SignificantBits:      significantBits,
			SilenceMaxAudioLevel: silenceMaxAudioLevel,
		},
	}
}

// OnAnalysis adds a callback executed upon receiving an analysis
func (i *Interface) OnAnalysis(fn AnalysisFunc) {
	i.onAnalysis = append(i.onAnalysis, fn)
}

// WebsocketListeners implements the astibob.WebsocketListener interface
func (i *Interface) WebsocketListeners() map[string]astiws.ListenerFunc {
	return map[string]astiws.ListenerFunc{
		websocketEventNameAnalysis: i.websocketListenerAnalysis,
	}
}

// websocketListenerAnalysis listens to the analysis websocket event
func (i *Interface) websocketListenerAnalysis(c *astiws.Client, eventName string, payload json.RawMessage) error {
	// Unmarshal payload
	var p string
	if err := json.Unmarshal(payload, &p); err != nil {
		astilog.Error(errors.Wrapf(err, "astiunderstanding: json unmarshaling %s into %#v failed", payload, p))
		return nil
	}

	// No callback
	if i.onAnalysis == nil {
		astilog.Error("astiunderstanding: onAnalysis is undefined")
		return nil
	}

	// Execute callbacks
	for _, fn := range i.onAnalysis {
		if err := fn(p); err != nil {
			astilog.Error(errors.Wrap(err, "astiunderstanding: executing analysis callback failed"))
		}
	}
	return nil
}
