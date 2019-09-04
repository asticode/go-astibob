package audio_input

import (
	"context"
	"encoding/json"

	"net/http"

	"sync"

	"time"

	"math"

	"fmt"

	"github.com/asticode/go-astibob"
	"github.com/asticode/go-astichartjs"
	"github.com/asticode/go-astilog"
	astiaudio "github.com/asticode/go-astitools/audio"
	astiptr "github.com/asticode/go-astitools/ptr"
	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
)

// Message names
const (
	samplesMessage = "audio_input.samples"
)

// Vars
var (
	calibrationDuration     = 5 * time.Second
	calibrationStepDuration = 100 * time.Millisecond
)

type Stream interface {
	BitDepth() int
	MaxSilenceAudioLevel() float64
	Read() ([]int32, error)
	SampleRate() float64
	Start() error
	Stop() error
}

type Runnable struct {
	*astibob.BaseOperatable
	*astibob.BaseRunnable
	cs []*calibration
	l  *Listenable
	mc *sync.Mutex // Locks cs
	s  Stream
}

func NewRunnable(name string, s Stream) *Runnable {
	// Create runnable
	r := &Runnable{
		BaseOperatable: newBaseOperatable(),
		mc:             &sync.Mutex{},
		s:              s,
	}

	// Add routes
	r.AddRoute("/calibrate", http.MethodGet, r.calibrate)

	// Set base runnable
	r.BaseRunnable = astibob.NewBaseRunnable(astibob.BaseRunnableOptions{
		Metadata: astibob.Metadata{
			Description: "Reads an audio input and dispatches audio samples",
			Name:        name,
		},
		OnStart: r.onStart,
	})

	// Set listenable
	r.l = newListenable(ListenableOptions{OnSamples: r.onSamples})
	return r
}

func (r *Runnable) MessageNames() []string {
	return r.l.MessageNames()
}

func (r *Runnable) OnMessage(m *astibob.Message) error {
	return r.l.OnMessage(m)
}

func (r *Runnable) onStart(ctx context.Context) (err error) {
	// Start stream
	if err = r.s.Start(); err != nil {
		err = errors.Wrap(err, "audio_input: starting stream failed")
		return
	}

	// Make sure to stop stream
	defer func() {
		if err := r.s.Stop(); err != nil {
			astilog.Error(errors.Wrap(err, "audio_input: stopping stream failed"))
			return
		}
	}()

	// Read
	for {
		// Check context
		if ctx.Err() != nil {
			return
		}

		// Read
		var b []int32
		if b, err = r.s.Read(); err != nil {
			err = errors.Wrap(err, "audio_input: reading failed")
			return
		}

		// Create message
		var m *astibob.Message
		if m, err = r.newSamplesMessage(b); err != nil {
			err = errors.Wrap(err, "audio_input: creating samples message failed")
			return
		}

		// Dispatch
		r.Dispatch(m)
	}
	return
}

type Samples struct {
	BitDepth             int     `json:"bit_depth"`
	MaxSilenceAudioLevel float64 `json:"max_silence_audio_level"`
	SampleRate           float64 `json:"sample_rate"`
	Samples              []int32 `json:"samples"`
}

func (r *Runnable) newSamplesMessage(b []int32) (m *astibob.Message, err error) {
	// Create message
	m = astibob.NewMessage()

	// Set name
	m.Name = samplesMessage

	// Marshal
	if m.Payload, err = json.Marshal(Samples{
		BitDepth:             r.s.BitDepth(),
		MaxSilenceAudioLevel: r.s.MaxSilenceAudioLevel(),
		Samples:              b,
		SampleRate:           r.s.SampleRate(),
	}); err != nil {
		err = errors.Wrap(err, "audio_input: marshaling payload failed")
		return
	}
	return
}

func parseSamplesPayload(m *astibob.Message) (ss Samples, err error) {
	if err = json.Unmarshal(m.Payload, &ss); err != nil {
		err = errors.Wrap(err, "audio_input: unmarshaling failed")
		return
	}
	return
}

func (r *Runnable) onSamples(_ astibob.Identifier, samples []int32, _ int, _, _ float64) (err error) {
	// Lock
	r.mc.Lock()

	// Loop through calibrations
	for idx, c := range r.cs {
		// Add samples
		var done bool
		if c.ctx.Err() == nil {
			done = c.add(samples)
		} else {
			done = true
		}

		// Remove calibration
		if done {
			r.cs = append(r.cs[:idx], r.cs[idx+1:]...)
			idx--
		}
	}

	// Unlock
	r.mc.Unlock()
	return
}

func (r *Runnable) calibrate(rw http.ResponseWriter, req *http.Request, p httprouter.Params) {
	// Set content type
	rw.Header().Set("Content-Type", "application/json")

	// Check status
	if r.Status() != astibob.RunningStatus {
		astibob.WriteHTTPError(rw, http.StatusBadRequest, fmt.Errorf("audio_input: status is %s", r.Status()))
		return
	}

	// Create new calibration
	c := r.newCalibration()
	defer c.close()

	// Wait
	if err := c.wait(); err != nil {
		astibob.WriteHTTPError(rw, http.StatusInternalServerError, errors.Wrap(err, "audio_input: waiting failed"))
		return
	}

	// Write results
	astibob.WriteHTTPData(rw, c.results())
}

type calibration struct {
	b      []int32
	c      *sync.Cond
	cancel context.CancelFunc
	ctx    context.Context
	mb     *sync.Mutex // Locks b
	s      Stream
}

func (r *Runnable) newCalibration() (c *calibration) {
	// Create calibration
	c = &calibration{
		c:  sync.NewCond(&sync.Mutex{}),
		mb: &sync.Mutex{},
		s:  r.s,
	}

	// Create context
	c.ctx, c.cancel = context.WithTimeout(context.Background(), 2*calibrationDuration)

	// Append
	r.mc.Lock()
	r.cs = append(r.cs, c)
	r.mc.Unlock()
	return
}

func (c *calibration) close() {
	c.cancel()
}

func (c *calibration) add(s []int32) (done bool) {
	// Get required number of samples
	// We take one more step than requested for the chart to be fully drawn
	n := int(c.s.SampleRate()*calibrationDuration.Seconds()) + int(c.s.SampleRate()*calibrationStepDuration.Seconds())

	// Lock
	c.mb.Lock()

	// Add samples
	if len(c.b)+len(s) <= n {
		c.b = append(c.b, s...)
	} else {
		// Append
		c.b = append(c.b, s[:n-len(c.b)]...)

		// Signal
		c.c.L.Lock()
		c.c.Signal()
		c.c.L.Unlock()

		// Update done
		done = true
	}

	// Unlock
	c.mb.Unlock()
	return
}

func (c *calibration) wait() (err error) {
	// Handle context
	go func() {
		// Wait for context to be done
		<-c.ctx.Done()

		// Signal
		c.c.L.Lock()
		err = c.ctx.Err()
		c.c.Signal()
		c.c.L.Unlock()
	}()

	// Wait
	c.c.L.Lock()
	c.c.Wait()
	c.c.L.Unlock()
	return
}

type Calibration struct {
	Chart                         astichartjs.Chart `json:"chart"`
	CurrentMaxSilenceAudioLevel   float64           `json:"current_max_silence_audio_level"`
	MaxAudioLevel                 float64           `json:"max_audio_level"`
	SuggestedMaxSilenceAudioLevel float64           `json:"suggested_max_silence_audio_level"`
}

func (c *calibration) results() (o Calibration) {
	// Create calibration
	o = Calibration{
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
	numberOfSamplesPerStep := int(math.Ceil(c.s.SampleRate() * calibrationStepDuration.Seconds()))

	// Get number of steps
	numberOfSteps := int(math.Ceil(float64(len(c.b)) / float64(numberOfSamplesPerStep)))

	// Process buffer
	var maxX float64
	for idx := 0; idx < numberOfSteps; idx++ {
		// Offsets
		start := idx * numberOfSamplesPerStep
		end := start + numberOfSamplesPerStep

		// Get samples
		var samples []int32
		if len(c.b) >= end {
			samples = c.b[start:end]
		} else {
			samples = c.b[start:]
		}

		// Compute audio level
		audioLevel := astiaudio.AudioLevel(samples)

		// Get max audio level
		o.MaxAudioLevel = math.Max(o.MaxAudioLevel, audioLevel)

		// Add data to chart
		maxX = float64(numberOfSamplesPerStep) / c.s.SampleRate() * float64(idx)
		o.Chart.Data.Datasets[0].Data = append(o.Chart.Data.Datasets[0].Data, astichartjs.DataPoint{
			X: maxX,
			Y: audioLevel,
		})
	}

	// Get current max silence audio level
	o.CurrentMaxSilenceAudioLevel = c.s.MaxSilenceAudioLevel()

	// Add current max silence audio level to chart
	o.Chart.Data.Datasets = append(o.Chart.Data.Datasets, astichartjs.Dataset{
		BackgroundColor: astichartjs.ChartBackgroundColorBlue,
		BorderColor:     astichartjs.ChartBorderColorBlue,
		Label:           "Current max silence audio level",
	})
	o.Chart.Data.Datasets[1].Data = append(o.Chart.Data.Datasets[1].Data, astichartjs.DataPoint{X: 0, Y: o.CurrentMaxSilenceAudioLevel})
	o.Chart.Data.Datasets[1].Data = append(o.Chart.Data.Datasets[1].Data, astichartjs.DataPoint{X: maxX, Y: o.CurrentMaxSilenceAudioLevel})

	// Get suggested max silence audio level
	o.SuggestedMaxSilenceAudioLevel = 0.3 * o.MaxAudioLevel

	// Add suggested max silence audio level to chart
	o.Chart.Data.Datasets = append(o.Chart.Data.Datasets, astichartjs.Dataset{
		BackgroundColor: astichartjs.ChartBackgroundColorRed,
		BorderColor:     astichartjs.ChartBorderColorRed,
		Label:           "Suggested max silence audio level",
	})
	o.Chart.Data.Datasets[2].Data = append(o.Chart.Data.Datasets[2].Data, astichartjs.DataPoint{X: 0, Y: o.SuggestedMaxSilenceAudioLevel})
	o.Chart.Data.Datasets[2].Data = append(o.Chart.Data.Datasets[2].Data, astichartjs.DataPoint{X: maxX, Y: o.SuggestedMaxSilenceAudioLevel})
	return
}
