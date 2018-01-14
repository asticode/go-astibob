package astihearing

import (
	"encoding/json"

	"sync"

	"time"

	"math"

	"github.com/asticode/go-astibob"
	"github.com/asticode/go-astilog"
	"github.com/asticode/go-astitools/audio"
	"github.com/asticode/go-astiws"
	"github.com/pkg/errors"
)

// Interface is the interface of the ability
// TODO Add default options to all project options
type Interface struct {
	calibrateBuf        *[]int32
	calibrateSampleRate int
	dispatchFunc        astibob.DispatchFunc
	mc                  sync.Mutex // Lock calibrateBuf
	o                   InterfaceOptions
	onSamples           []SamplesFunc
}

// InterfaceOptions represents interface options
type InterfaceOptions struct {
	CalibrateMaxDuration  time.Duration `toml:"calibrate_max_duration"`
	CalibrateStepDuration time.Duration `toml:"calibrate_step_duration"`
}

// SamplesFunc represents the callback executed upon receiving samples
type SamplesFunc func(samples []int32, sampleRate, significantBits int, silenceMaxAudioLevel float64) error

// NewInterface creates a new interface
func NewInterface(o InterfaceOptions) (i *Interface) {
	i = &Interface{o: o}
	i.onSamples = append(i.onSamples, i.onSamplesCalibrate)
	return
}

// Name implements the astibob.Interface interface
func (i *Interface) Name() string {
	return Name
}

// SetDispatchFunc implements the astibob.Dispatcher interface
func (i *Interface) SetDispatchFunc(fn astibob.DispatchFunc) {
	i.dispatchFunc = fn
}

// OnSamples adds a callback executed upon receiving samples
func (i *Interface) OnSamples(fn SamplesFunc) {
	i.onSamples = append(i.onSamples, fn)
}

// onSamplesCalibrate is the samples callback for the calibration
func (i *Interface) onSamplesCalibrate(samples []int32, sampleRate, significantBits int, silenceMaxAudioLevel float64) error {
	// Lock
	i.mc.Lock()
	defer i.mc.Unlock()

	// Interface is not calibrating
	if i.calibrateBuf == nil {
		return nil
	}

	// Set sample rate
	i.calibrateSampleRate = sampleRate

	// Add samples
	*i.calibrateBuf = append(*i.calibrateBuf, samples...)

	// Check calibrate max duration
	if float64(len(*i.calibrateBuf))/float64(i.calibrateSampleRate) > i.o.CalibrateMaxDuration.Seconds() {
		i.calibrate()
	}
	return nil
}

// BrainWebsocketListeners implements the astibob.BrainWebsocketListener interface
func (i *Interface) BrainWebsocketListeners() map[string]astiws.ListenerFunc {
	return map[string]astiws.ListenerFunc{
		websocketEventNameSamples: i.brainWebsocketListenerSamples,
	}
}

// brainWebsocketListenerSamples listens to the samples brain websocket event
func (i *Interface) brainWebsocketListenerSamples(c *astiws.Client, eventName string, payload json.RawMessage) error {
	// Unmarshal payload
	var p PayloadSamples
	if err := json.Unmarshal(payload, &p); err != nil {
		astilog.Error(errors.Wrapf(err, "astihearing: json unmarshaling %s into %#v failed", payload, p))
		return nil
	}

	// No callback
	if i.onSamples == nil {
		astilog.Error("astihearing: onSamples is undefined")
		return nil
	}

	// Execute callbacks
	for _, fn := range i.onSamples {
		if err := fn(p.Samples, p.SampleRate, p.SignificantBits, p.SilenceMaxAudioLevel); err != nil {
			astilog.Error(errors.Wrap(err, "astihearing: executing samples callback failed"))
		}
	}
	return nil
}

// ClientWebsocketListeners implements the astibob.ClientWebsocketListener interface
func (i *Interface) ClientWebsocketListeners() map[string]astiws.ListenerFunc {
	return map[string]astiws.ListenerFunc{
		"calibrate.start": i.clientWebsocketListenerCalibrateStart,
		"calibrate.stop":  i.clientWebsocketListenerCalibrateStop,
	}
}

// clientWebsocketListenerCalibrateStart listens to the calibrate.start client websocket event
// TODO Make sure the ability is on
func (i *Interface) clientWebsocketListenerCalibrateStart(c *astiws.Client, eventName string, payload json.RawMessage) error {
	// Lock
	i.mc.Lock()
	defer i.mc.Unlock()

	// Already calibrating
	if i.calibrateBuf != nil {
		return nil
	}

	// Reset buf
	i.calibrateBuf = &[]int32{}
	return nil
}

// CalibrateResults represents calibrate results
// TODO Add audio level graph
type CalibrateResults struct {
	MaxAudioLevel        float64 `json:"max_audio_level"`
	SilenceMaxAudioLevel float64 `json:"silence_max_audio_level"`
}

// clientWebsocketListenerCalibrateStop listens to the calibrate.stop client websocket event
func (i *Interface) clientWebsocketListenerCalibrateStop(c *astiws.Client, eventName string, payload json.RawMessage) error {
	// Lock
	i.mc.Lock()
	i.mc.Unlock()

	// Not calibrating
	if i.calibrateBuf == nil {
		return nil
	}

	// Calibrate
	i.calibrate()
	return nil
}

// calibrate processes the calibrate buffer.
// Assumption is made that mc is locked
func (i *Interface) calibrate() {
	// Create payload
	p := CalibrateResults{}

	// Get number of samples per steps
	numberOfSamplesPerStep := int(math.Ceil(float64(i.calibrateSampleRate) * i.o.CalibrateStepDuration.Seconds()))

	// Get number of steps
	numberOfSteps := int(math.Ceil(float64(len(*i.calibrateBuf)) / float64(numberOfSamplesPerStep)))

	// Process buffer
	for idx := 0; idx < numberOfSteps; idx++ {
		// Offsets
		start := idx * numberOfSamplesPerStep
		end := start + numberOfSamplesPerStep

		// Get samples
		var samples []int32
		if len(*i.calibrateBuf) >= end {
			samples = (*i.calibrateBuf)[start:end]
		} else {
			samples = (*i.calibrateBuf)[start:]
		}

		// Compute audio level
		audioLevel := astiaudio.AudioLevel(samples)

		// Get max audio level
		p.MaxAudioLevel = math.Max(p.MaxAudioLevel, audioLevel)
	}

	// Get silence max audio level
	p.SilenceMaxAudioLevel = float64(1) * p.MaxAudioLevel / float64(3)

	// Reset buffer
	i.calibrateBuf = nil

	// Dispatch to clients
	i.dispatchFunc(astibob.ClientEvent{
		Name:    "calibrate.results",
		Payload: p,
	})
}

// WebTemplates implements the astibob.WebTemplater interface
func (i *Interface) WebTemplates() map[string]string {
	return map[string]string{
		"/index": i.webTemplateIndex(),
	}
}

// webTemplateIndex returns the index web template
func (i *Interface) webTemplateIndex() string {
	return `{{ define "title" }}Hearing{{ end }}
{{ define "css" }}{{ end }}
{{ define "html" }}
<div class='header'>Calibrate</div>
<p>Click "Calibrate" to retrieve the max audio level as well as the deduced silence max audio level appropriate to your audio device.</p>
<button class="default" id="btn-calibrate">Calibrate</button>
<p id="calibrate-results"></p>
{{ end }}
{{ define "js" }}
<script type="text/javascript">
	let hearing = {
		init: function() {
			base.init(hearing.websocketFunc, function(data) {
				// Handle calibrate
				$("#btn-calibrate").click(hearing.handleClickCalibrate);

				// Finish
				base.finish();
			});
		},
		handleClickCalibrate: function() {
			// Create stop
			let stop = document.createElement("button");
			stop.className = "default";
			stop.innerText = "Stop";
			stop.onclick = function() {
				// Send ws event
				base.sendWs(base.abilityWebsocketEventName("calibrate.stop"))

				// Hide modal
				asticode.modaler.hide();
			};

			// Create div
			let div = document.createElement("div");
			div.innerHTML = '<div style="margin-bottom:15px">Say something...</div>'
			div.appendChild(stop);

			// Show modal
			asticode.modaler.setWidth("300px");
			asticode.modaler.setContent(div);
			asticode.modaler.show();

			// Send ws event
			base.sendWs(base.abilityWebsocketEventName("calibrate.start"))
		},
		addCalibrateResults: function(results) {
			let html = "<table><tbody>";
			html += "<tr><td style='font-weight: bold; padding-right: 10px; text-align: left'>Max audio level</td><td style='text-align: right'>" + Math.round(results.max_audio_level) + "</td></tr>";
			html += "<tr><td style='font-weight: bold; padding-right: 10px; text-align: left'>Silence max audio level</td><td style='text-align: right'>" + Math.round(results.silence_max_audio_level) + "</td></tr>";
			html += "</tbody></table>";
			$("#calibrate-results").html(html);
		},
    	websocketFunc: function(event_name, payload) {
			switch (event_name) {
				case base.abilityWebsocketEventName("calibrate.results"):
					// Close modal
					asticode.modaler.hide();

					// Add results
					hearing.addCalibrateResults(payload);
					break;
			}
		}
	}
	hearing.init();
</script>
{{ end }}
{{ template "base" . }}`
}
