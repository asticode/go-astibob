package astiunderstanding

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"strings"

	"io/ioutil"

	"context"

	"github.com/asticode/go-astibob"
	"github.com/asticode/go-astilog"
	"github.com/asticode/go-astitools/os"
	"github.com/asticode/go-astiws"
	"github.com/pkg/errors"
)

// Interface is the interface of the ability
type Interface struct {
	c               InterfaceConfiguration
	dispatchFunc    astibob.DispatchFunc
	onAnalysis      []AnalysisFunc
	onSamplesStored []SamplesStoredFunc
}

// InterfaceConfiguration represents an interface configuration
type InterfaceConfiguration struct {
	SamplesDirectory string `toml:"samples_directory"`
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

// SamplesStoredFunc represents the callback executed when samples have been stored
type SamplesStoredFunc func(id, text string) error

// NewInterface creates a new interface
func NewInterface(c InterfaceConfiguration) (i *Interface, err error) {
	// Create
	i = &Interface{c: c}

	// Add default callbacks
	i.onSamplesStored = append(i.onSamplesStored, i.onSamplesStoredDispatch)

	// Absolute paths
	if len(i.c.SamplesDirectory) > 0 {
		if i.c.SamplesDirectory, err = filepath.Abs(i.c.SamplesDirectory); err != nil {
			err = errors.Wrapf(err, "astiunderstanding: filepath abs of %s failed", i.c.SamplesDirectory)
			return
		}
	}
	return
}

// SetDispatchFunc implements the astibob.Dispatcher interface
func (i *Interface) SetDispatchFunc(fn astibob.DispatchFunc) {
	i.dispatchFunc = fn
}

// Name implements the astibob.Interface interface
func (i *Interface) Name() string {
	return name
}

// Samples creates a samples cmd
func (i *Interface) Samples(samples []int32, sampleRate, significantBits int, silenceMaxAudioLevel float64) *astibob.Cmd {
	return &astibob.Cmd{
		AbilityName: name,
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

// OnSamplesStored adds a callback executed upon receiving notification that samples have been stored
func (i *Interface) OnSamplesStored(fn SamplesStoredFunc) {
	i.onSamplesStored = append(i.onSamplesStored, fn)
}

// onSamplesStoredDispatch is the samples stored callback for the dispatch
func (i *Interface) onSamplesStoredDispatch(id, text string) error {
	if i.dispatchFunc != nil {
		i.dispatchFunc(astibob.ClientEvent{Name: "samples.stored", Payload: newPayloadStoredSamples(id, text)})
	}
	return nil
}

// BrainWebsocketListeners implements the astibob.BrainWebsocketListener interface
func (i *Interface) BrainWebsocketListeners() map[string]astiws.ListenerFunc {
	return map[string]astiws.ListenerFunc{
		websocketEventNameAnalysis:      i.brainWebsocketListenerAnalysis,
		websocketEventNameSamplesStored: i.brainWebsocketListenerSamplesStored,
	}
}

// brainWebsocketListenerAnalysis listens to the analysis brain websocket event
func (i *Interface) brainWebsocketListenerAnalysis(c *astiws.Client, eventName string, payload json.RawMessage) error {
	// Unmarshal payload
	var p string
	if err := json.Unmarshal(payload, &p); err != nil {
		astilog.Error(errors.Wrapf(err, "astiunderstanding: json unmarshaling %s into %#v failed", payload, p))
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

// brainWebsocketListenerSamplesStored listens to the samples.stored brain websocket event
func (i *Interface) brainWebsocketListenerSamplesStored(c *astiws.Client, eventName string, payload json.RawMessage) error {
	// Unmarshal payload
	var p PayloadStoredSamples
	if err := json.Unmarshal(payload, &p); err != nil {
		astilog.Error(errors.Wrapf(err, "astiunderstanding: json unmarshaling %s into %#v failed", payload, p))
		return nil
	}

	// Execute callbacks
	for _, fn := range i.onSamplesStored {
		if err := fn(p.ID, p.Text); err != nil {
			astilog.Error(errors.Wrap(err, "astiunderstanding: executing samples stored callback failed"))
		}
	}
	return nil
}

// APIHandlers implements the astibob.APIHandle interface
func (i *Interface) APIHandlers() map[string]http.Handler {
	return map[string]http.Handler{
		"/": i.apiHandlerIndex(),
	}
}

// apiHandlerIndex handles the index api request
func (i *Interface) apiHandlerIndex() http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		// Walk folder
		ss := []PayloadStoredSamples{}
		root := samplesToBeValidatedDirectory(i.c.SamplesDirectory)
		filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			// Process error
			if err != nil {
				return err
			}

			// Only process wav files
			if info.IsDir() || !strings.HasSuffix(path, ".wav") {
				return nil
			}

			// Get id
			id := strings.TrimSuffix(strings.TrimPrefix(path, root), ".wav")

			// Read txt file
			txtPath := strings.TrimSuffix(path, ".wav") + ".txt"
			var text []byte
			if text, err = ioutil.ReadFile(txtPath); err != nil {
				return errors.Wrapf(err, "reading %s failed", txtPath)
			}

			// Append
			ss = append(ss, newPayloadStoredSamples(id, string(text)))
			return nil
		})

		// Write
		astibob.APIWrite(rw, ss)
	})
}

// ClientWebsocketListeners implements the astibob.ClientWebsocketListener interface
func (i *Interface) ClientWebsocketListeners() map[string]astiws.ListenerFunc {
	return map[string]astiws.ListenerFunc{
		"samples.remove":   i.ClientWebsocketListenerHandleSamples("removed"),
		"samples.validate": i.ClientWebsocketListenerHandleSamples("validated"),
	}
}

// ClientWebsocketListenerRemoveSamples listens to the samples.remove client websocket event
func (i *Interface) ClientWebsocketListenerHandleSamples(reward string) astiws.ListenerFunc {
	return func(c *astiws.Client, eventName string, payload json.RawMessage) error {
		// Handle error
		var err error
		defer func(err *error) {
			if *err != nil {
				astilog.Error(*err)
				if i.dispatchFunc != nil {
					i.dispatchFunc(astibob.ClientEvent{Name: "error", Payload: (*err).Error()})
				}
			}
		}(&err)

		// Unmarshal payload
		var p PayloadStoredSamples
		if err = json.Unmarshal(payload, &p); err != nil {
			err = errors.Wrapf(err, "astiunderstanding: json unmarshaling %s into %#v failed", payload, p)
			return nil
		}

		// Validate
		if reward == "validated" {
			// No text
			if len(p.Text) == 0 {
				err = errors.New("astiunderstanding: no text provided")
				return nil
			}

			// Copy wav file
			var src, dst = filepath.Join(samplesToBeValidatedDirectory(i.c.SamplesDirectory), p.ID+".wav"), filepath.Join(samplesValidatedDirectory(i.c.SamplesDirectory), p.ID+".wav")
			if err = astios.Copy(context.Background(), src, dst); err != nil {
				err = errors.Wrapf(err, "astiunderstanding: copying %s to %s failed", src, dst)
				return nil
			}

			// Write txt file
			dst = filepath.Join(samplesValidatedDirectory(i.c.SamplesDirectory), p.ID+".txt")
			if err = ioutil.WriteFile(dst, []byte(p.Text), 0755); err != nil {
				err = errors.Wrapf(err, "astiunderstanding: writing into %s failed", dst)
				return nil
			}
		}

		// Remove files
		for _, p := range []string{
			filepath.Join(samplesToBeValidatedDirectory(i.c.SamplesDirectory), p.ID+".wav"),
			filepath.Join(samplesToBeValidatedDirectory(i.c.SamplesDirectory), p.ID+".txt"),
		} {
			if err = os.Remove(p); err != nil {
				err = errors.Wrapf(err, "astiunderstanding: removing %s failed", p)
				return nil
			}
		}

		// Dispatch to clients
		if i.dispatchFunc != nil {
			i.dispatchFunc(astibob.ClientEvent{Name: "samples." + reward, Payload: p})
		}
		return nil
	}
}

// StaticHandlers implements the astibob.StaticHandler interface
func (i *Interface) StaticHandlers() map[string]http.Handler {
	return map[string]http.Handler{
		"/samples": http.FileServer(http.Dir(samplesToBeValidatedDirectory(i.c.SamplesDirectory))),
	}
}

// WebTemplates implements the astibob.WebTemplater interface
func (i *Interface) WebTemplates() map[string]string {
	return map[string]string{
		"/index": i.webTemplateIndex(),
	}
}

// webTemplateIndex returns the index web template
func (i *Interface) webTemplateIndex() string {
	return `{{ define "title" }}Understanding{{ end }}
{{ define "css" }}{{ end }}
{{ define "html" }}
	<div class="header">Validate samples</div>
	<p>Listen to the audio, write the transcript and press "Enter" to validate or "Ctrl+Enter" to remove.</p>
	<div class="flex" id="samples-to-be-validated"></div>
{{ end }}
{{ define "js" }}
<script type="text/javascript">
	let understanding = {
		samplesToBeValidated: {},

		init: function() {
			base.init(understanding.websocketFunc, function(data) {
				// Fetch data
				base.sendHttp(base.abilityAPIPattern("/"), "GET", function(data) {
					// Display samples to be validated
					if (typeof data !== "undefined" && data !== null) {
						// Loop through data
						for (let idx = 0; idx < data.length; idx++) {
							understanding.addSampleToBeValidated(data[idx]);
						}

						// Make sure the first input is focused
						$("#samples-to-be-validated input").eq(0).focus();
					}

					// Finish
					base.finish();
				}, function() {
					asticode.loader.hide();
				});
			});
		},
		addSampleToBeValidated: function(samples) {
			// Create base
			let r = {};
			r.wrapper = $("<div class='panel-wrapper'></div>");
			r.wrapper.appendTo($("#samples-to-be-validated"));
			let panel = $("<div class='panel'></div>");
			panel.appendTo(r.wrapper);
			let table = $("<div class='table' style='width: 100%'></div>");
			table.appendTo(panel);
			let row = $("<div class='row'></div>");
			row.appendTo(table);

			// Create audio
			let audio = new Audio(base.abilityStaticPattern(samples.wav_static_path));

			// Play
			let playCell = $("<div class='cell' style='width: 28px'></div>");
			playCell.appendTo(row);
			let play = $("<i class='fa fa-play color-info-back' style='cursor: pointer'></i>");
			play.on("click", function() {
				understanding.stopAudio(audio);
				audio.play();
			});
			play.appendTo(playCell);

			// Input
			let inputCell = $("<div class='cell'></div>");
			inputCell.appendTo(row);
			let input = $("<input type='text' style='border: solid 1px #dedee0; width: 100%' value='" + samples.text + "'/>");
			input.on("focus", function() {
				understanding.stopAudio(audio);
				audio.play();
			});
			input.on("keyup", function(e) {
				if (e.keyCode == 13) {
					if (e.ctrlKey) {
						base.sendWs(base.abilityWebsocketEventName("samples.remove"), samples);
					} else {
						samples.text = input.val();
						base.sendWs(base.abilityWebsocketEventName("samples.validate"), samples);
					}
				}
			});
			input.appendTo(inputCell);

			// Append samples
			understanding.samplesToBeValidated[samples.id] = r
		},
		removeSamplesToBeValidated: function(samples) {
			// Fetch samples
			let s = understanding.samplesToBeValidated[samples.id];
	
			// Samples exists
			if (typeof s !== "undefined") {
				// Remove HTML
				s.wrapper.remove();
	
				// Remove from pool
				delete(understanding.samplesToBeValidated[samples.id]);
			}
		},
		stopAudio: function(audio) {
			audio.pause();
			audio.currentTime = 0;
		},
    	websocketFunc: function(event_name, payload) {
			switch (event_name) {
				case base.abilityWebsocketEventName("error"):
					// Display message
					asticode.notifier.error(payload);
					break;
				case base.abilityWebsocketEventName("samples.stored"):
					understanding.addSampleToBeValidated(payload);
					break;
				case base.abilityWebsocketEventName("samples.removed"):
				case base.abilityWebsocketEventName("samples.validated"):
					// Remove from UI
					understanding.removeSamplesToBeValidated(payload);

					// Display message
					if (event_name == base.abilityWebsocketEventName("samples.validated")) {
						asticode.notifier.success("Samples have been validated");
					} else {
						asticode.notifier.success("Samples have been removed");
					}

					// Make sure the first input is focused
					$("#samples-to-be-validated input").eq(0).focus();
					break;
			}
		}
	}
	understanding.init();
</script>
{{ end }}
{{ template "base" . }}`
}
