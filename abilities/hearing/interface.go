package astihearing

import (
	"encoding/json"

	"sync"

	"time"

	"math"

	"github.com/asticode/go-astibob"
	"github.com/asticode/go-astichartjs"
	"github.com/asticode/go-astilog"
	"github.com/asticode/go-astitools/audio"
	"github.com/asticode/go-astitools/ptr"
	"github.com/asticode/go-astiws"
	"github.com/pkg/errors"
)

// Interface is the interface of the ability
type Interface struct {
	c                     InterfaceConfiguration
	calibrationBuf        *[]int32
	calibrationSampleRate int
	dispatchFunc          astibob.DispatchFunc
	mc                    sync.Mutex // Lock calibrationBuf
	onSamples             []SamplesFunc
}

// InterfaceConfiguration represents an interface configuration
type InterfaceConfiguration struct {
	CalibrationDuration     time.Duration `toml:"calibration_duration"`
	CalibrationStepDuration time.Duration `toml:"calibration_step_duration"`
}

// SamplesFunc represents the callback executed upon receiving samples
type SamplesFunc func(samples []int32, sampleRate, significantBits int, silenceMaxAudioLevel float64) error

// NewInterface creates a new interface
func NewInterface(c InterfaceConfiguration) (i *Interface) {
	// Create
	i = &Interface{c: c}

	// Add default callbacks
	i.onSamples = append(i.onSamples, i.onSamplesCalibration)

	// Default configuration values
	if i.c.CalibrationDuration == 0 {
		i.c.CalibrationDuration = 5 * time.Second
	}
	if i.c.CalibrationStepDuration == 0 {
		i.c.CalibrationStepDuration = 40 * time.Millisecond
	}
	return
}

// Name implements the astibob.Interface interface
func (i *Interface) Name() string {
	return name
}

// SetDispatchFunc implements the astibob.Dispatcher interface
func (i *Interface) SetDispatchFunc(fn astibob.DispatchFunc) {
	i.dispatchFunc = fn
}

// OnSamples adds a callback executed upon receiving samples
func (i *Interface) OnSamples(fn SamplesFunc) {
	i.onSamples = append(i.onSamples, fn)
}

// onSamplesCalibration is the samples callback for the calibration
func (i *Interface) onSamplesCalibration(samples []int32, sampleRate, significantBits int, silenceMaxAudioLevel float64) error {
	// Lock
	i.mc.Lock()
	defer i.mc.Unlock()

	// Interface is not calibrating
	if i.calibrationBuf == nil {
		return nil
	}

	// Set sample rate
	i.calibrationSampleRate = sampleRate

	// Get max number of samples
	// We take one more step than requested
	maxNumberOfSamples := int(float64(i.calibrationSampleRate)*i.c.CalibrationDuration.Seconds()) + int(float64(i.calibrationSampleRate)*float64(i.c.CalibrationStepDuration.Seconds()))

	// Add samples
	if len(*i.calibrationBuf)+len(samples) <= maxNumberOfSamples {
		*i.calibrationBuf = append(*i.calibrationBuf, samples...)
	} else {
		*i.calibrationBuf = append(*i.calibrationBuf, samples[:maxNumberOfSamples-len(*i.calibrationBuf)]...)
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
		"calibration.start": i.clientWebsocketListenerCalibrationStart,
	}
}

// clientWebsocketListenerCalibrationStart listens to the calibration.start client websocket event
// TODO Make sure the ability is on
func (i *Interface) clientWebsocketListenerCalibrationStart(c *astiws.Client, eventName string, payload json.RawMessage) error {
	// Lock
	i.mc.Lock()
	defer i.mc.Unlock()

	// Already calibrating
	if i.calibrationBuf != nil {
		return nil
	}

	// Reset buf
	i.calibrationBuf = &[]int32{}
	return nil
}

// CalibrationResults represents calibration results
type CalibrationResults struct {
	Chart                astichartjs.Chart `json:"chart"`
	MaxAudioLevel        float64           `json:"max_audio_level"`
	SilenceMaxAudioLevel float64           `json:"silence_max_audio_level"`
}

// calibrate processes the calibration buffer.
// Assumption is made that mc is locked
func (i *Interface) calibrate() {
	// Create payload
	p := CalibrationResults{
		Chart: astichartjs.Chart{
			Data: &astichartjs.Data{
				Datasets: []astichartjs.Dataset{{
					BackgroundColor: astichartjs.ChartBackgroundColorGreen,
					BorderColor:     astichartjs.ChartBorderColorGreen,
					Label:           "Audio level",
				}},
			},
			Options: &astichartjs.Options{
				Scales: &astichartjs.Scales{
					XAxes: []astichartjs.Axis{
						{
							Position: astichartjs.ChartAxisPositionsBottom,
							ScaleLabel: &astichartjs.ScaleLabel{
								Display:     astiptr.Bool(true),
								LabelString: "Duration (s)",
							},
							Type: astichartjs.ChartAxisTypesLinear,
						},
					},
					YAxes: []astichartjs.Axis{
						{
							ScaleLabel: &astichartjs.ScaleLabel{
								Display:     astiptr.Bool(true),
								LabelString: "Audio level",
							},
						},
					},
				},
				Title: &astichartjs.Title{Display: astiptr.Bool(true)},
			},
			Type: astichartjs.ChartTypeLine,
		},
	}

	// Get number of samples per steps
	numberOfSamplesPerStep := int(math.Ceil(float64(i.calibrationSampleRate) * i.c.CalibrationStepDuration.Seconds()))

	// Get number of steps
	numberOfSteps := int(math.Ceil(float64(len(*i.calibrationBuf)) / float64(numberOfSamplesPerStep)))

	// Process buffer
	var maxX float64
	for idx := 0; idx < numberOfSteps; idx++ {
		// Offsets
		start := idx * numberOfSamplesPerStep
		end := start + numberOfSamplesPerStep

		// Get samples
		var samples []int32
		if len(*i.calibrationBuf) >= end {
			samples = (*i.calibrationBuf)[start:end]
		} else {
			samples = (*i.calibrationBuf)[start:]
		}

		// Compute audio level
		audioLevel := astiaudio.AudioLevel(samples)

		// Get max audio level
		p.MaxAudioLevel = math.Max(p.MaxAudioLevel, audioLevel)

		// Add data to chart
		maxX = float64(numberOfSamplesPerStep) / float64(i.calibrationSampleRate) * float64(idx)
		p.Chart.Data.Datasets[0].Data = append(p.Chart.Data.Datasets[0].Data, astichartjs.DataPoint{
			X: maxX,
			Y: audioLevel,
		})
	}

	// Get silence max audio level
	p.SilenceMaxAudioLevel = float64(1) * p.MaxAudioLevel / float64(3)

	// Add data to chart
	p.Chart.Data.Datasets = append(p.Chart.Data.Datasets, astichartjs.Dataset{
		BackgroundColor: astichartjs.ChartBackgroundColorRed,
		BorderColor:     astichartjs.ChartBorderColorRed,
		Label:           "Silence max audio level",
	})
	p.Chart.Data.Datasets[1].Data = append(p.Chart.Data.Datasets[1].Data, astichartjs.DataPoint{X: 0, Y: p.SilenceMaxAudioLevel})
	p.Chart.Data.Datasets[1].Data = append(p.Chart.Data.Datasets[1].Data, astichartjs.DataPoint{X: maxX, Y: p.SilenceMaxAudioLevel})

	// Reset buffer
	i.calibrationBuf = nil

	// Dispatch to clients
	i.dispatchFunc(astibob.ClientEvent{
		Name:    "calibration.results",
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
<div class='header'>Calibration</div>
<p>Click "Calibrate" to retrieve the max audio level as well as the deduced silence max audio level appropriate to your audio device.</p>
<button class="default" id="btn-calibrate">Calibrate</button>
<p id="calibration-results"></p>
{{ end }}
{{ define "js" }}
<script type="text/javascript">
	let hearing = {
		init: function() {
			base.init(hearing.websocketFunc, function(data) {
				// Handle calibration
				$("#btn-calibrate").click(hearing.handleClickCalibrate);

				// Finish
				base.finish();
			});
		},
		handleClickCalibrate: function() {
			// Create div
			let div = document.createElement("div");
			div.innerHTML = 'Say something...';

			// Show modal
			asticode.modaler.setWidth("300px");
			asticode.modaler.setContent(div);
			asticode.modaler.show();

			// Send ws event
			base.sendWs(base.abilityWebsocketEventName("calibration.start"))
		},
		addCalibrationResults: function(results) {
			// Create html
			let html = "<table><tbody>";
			html += "<tr><td style='font-weight: bold; padding-right: 10px; text-align: left'>Max audio level</td><td style='text-align: right'>" + Math.round(results.max_audio_level) + "</td></tr>";
			html += "<tr><td style='font-weight: bold; padding-right: 10px; text-align: left'>Silence max audio level</td><td style='text-align: right'>" + Math.round(results.silence_max_audio_level) + "</td></tr>";
			html += "</tbody></table>";
			html += "<canvas id='chart'></canvas>";

			// Set html
			$("#calibration-results").html(html);

			// Add chart
			new Chart(document.getElementById("chart"), results.chart);
		},
    	websocketFunc: function(event_name, payload) {
			switch (event_name) {
				case base.abilityWebsocketEventName("calibration.results"):
					// Close modal
					asticode.modaler.hide();

					// Add results
					hearing.addCalibrationResults(payload);
					break;
			}
		}
	}
	hearing.init();
</script>
{{ end }}
{{ template "base" . }}`
}
