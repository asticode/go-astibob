package audio_input

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/asticode/go-astibob"
	"github.com/asticode/go-astichartjs"
	"github.com/asticode/go-astikit"
	"github.com/julienschmidt/httprouter"
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
	MaxSilenceLevel() float64
	NumChannels() int
	Read() ([]int, error)
	SampleRate() int
	Start() error
	Stop() error
}

type Runnable struct {
	*astibob.BaseOperatable
	*astibob.BaseRunnable
	cs []*calibration
	l  *Listenable
	lg astikit.SeverityLogger
	mc *sync.Mutex // Locks cs
	s  Stream
}

func NewRunnable(name string, s Stream, l astikit.StdLogger) *Runnable {
	// Create runnable
	r := &Runnable{
		lg: astikit.AdaptStdLogger(l),
		mc: &sync.Mutex{},
		s:  s,
	}

	// Add base operatable
	r.BaseOperatable = newBaseOperatable(r.lg)

	// Add routes
	r.AddRoute("/calibrate", http.MethodGet, r.calibrate)

	// Set listenable
	r.l = newListenable(ListenableOptions{OnSamples: r.onSamples})

	// Set base runnable
	r.BaseRunnable = astibob.NewBaseRunnable(astibob.BaseRunnableOptions{
		Logger: l,
		Metadata: astibob.Metadata{
			Description: "Reads an audio input and dispatches audio samples",
			Name:        name,
		},
		OnMessage: r.l.OnMessage,
		OnStart:   r.onStart,
	})
	return r
}

func (r *Runnable) MessageNames() []string {
	return r.l.MessageNames()
}

func (r *Runnable) onStart(ctx context.Context) (err error) {
	// Start stream
	if err = r.s.Start(); err != nil {
		err = fmt.Errorf("audio_input: starting stream failed: %w", err)
		return
	}

	// Make sure to stop stream
	defer func() {
		if err := r.s.Stop(); err != nil {
			r.lg.Error(fmt.Errorf("audio_input: stopping stream failed: %w", err))
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
		var b []int
		if b, err = r.s.Read(); err != nil {
			err = fmt.Errorf("audio_input: reading failed: %w", err)
			return
		}

		// Create message
		var m *astibob.Message
		if m, err = r.newSamplesMessage(b); err != nil {
			err = fmt.Errorf("audio_input: creating samples message failed: %w", err)
			return
		}

		// Dispatch
		r.Dispatch(m)
	}
}

type Samples struct {
	BitDepth        int     `json:"bit_depth"`
	MaxSilenceLevel float64 `json:"max_silence_level"`
	NumChannels     int     `json:"num_channels"`
	SampleRate      int     `json:"sample_rate"`
	Samples         []int   `json:"samples"`
}

func (r *Runnable) newSamplesMessage(b []int) (m *astibob.Message, err error) {
	// Create message
	m = astibob.NewMessage()

	// Set name
	m.Name = samplesMessage

	// Marshal
	if m.Payload, err = json.Marshal(Samples{
		BitDepth:        r.s.BitDepth(),
		MaxSilenceLevel: r.s.MaxSilenceLevel(),
		NumChannels:     r.s.NumChannels(),
		Samples:         b,
		SampleRate:      r.s.SampleRate(),
	}); err != nil {
		err = fmt.Errorf("audio_input: marshaling payload failed: %w", err)
		return
	}
	return
}

func parseSamplesPayload(m *astibob.Message) (ss Samples, err error) {
	if err = json.Unmarshal(m.Payload, &ss); err != nil {
		err = fmt.Errorf("audio_input: unmarshaling failed: %w", err)
		return
	}
	return
}

func (r *Runnable) onSamples(_ astibob.Identifier, samples []int, _, _, _ int, _ float64) (err error) {
	// Lock
	r.mc.Lock()

	// Loop through calibrations
	for idx := 0; idx < len(r.cs); idx++ {
		// Add samples
		var done bool
		if r.cs[idx].ctx.Err() == nil {
			done = r.cs[idx].add(samples)
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
		astibob.WriteHTTPError(r.lg, rw, http.StatusBadRequest, fmt.Errorf("audio_input: status is %s", r.Status()))
		return
	}

	// Create new calibration
	c := r.newCalibration()
	defer c.close()

	// Wait
	if err := c.wait(); err != nil {
		astibob.WriteHTTPError(r.lg, rw, http.StatusInternalServerError, fmt.Errorf("audio_input: waiting failed: %w", err))
		return
	}

	// Write results
	astibob.WriteHTTPData(r.lg, rw, c.results())
}

type calibration struct {
	b      []int
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

func (c *calibration) add(s []int) (done bool) {
	// Get required number of samples
	// We take one more step than requested for the chart to be fully drawn
	n := int(float64(c.s.SampleRate())*calibrationDuration.Seconds()) + int(float64(c.s.SampleRate())*calibrationStepDuration.Seconds())

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
	Chart                    astichartjs.Chart `json:"chart"`
	CurrentMaxSilenceLevel   float64           `json:"current_max_silence_level"`
	MaxLevel                 float64           `json:"max_level"`
	SuggestedMaxSilenceLevel float64           `json:"suggested_max_silence_level"`
}

func (c *calibration) results() (o Calibration) {
	// Create calibration
	o = Calibration{
		Chart: astichartjs.Chart{
			Data: &astichartjs.Data{
				Datasets: []astichartjs.Dataset{{
					BackgroundColor: astichartjs.ChartBackgroundColorGreen,
					BorderColor:     astichartjs.ChartBorderColorGreen,
					Label:           "Level",
				}},
			},
			Options: &astichartjs.Options{
				Scales: &astichartjs.Scales{
					XAxes: []astichartjs.Axis{
						{
							Position: astichartjs.ChartAxisPositionsBottom,
							ScaleLabel: &astichartjs.ScaleLabel{
								Display:     astikit.BoolPtr(true),
								LabelString: "Duration (s)",
							},
							Type: astichartjs.ChartAxisTypesLinear,
						},
					},
					YAxes: []astichartjs.Axis{
						{
							ScaleLabel: &astichartjs.ScaleLabel{
								Display:     astikit.BoolPtr(true),
								LabelString: "Level",
							},
						},
					},
				},
				Title: &astichartjs.Title{Display: astikit.BoolPtr(true)},
			},
			Type: astichartjs.ChartTypeLine,
		},
	}

	// Get number of samples per steps
	numberOfSamplesPerStep := int(math.Ceil(float64(c.s.SampleRate()) * calibrationStepDuration.Seconds()))

	// Get number of steps
	numberOfSteps := int(math.Ceil(float64(len(c.b)) / float64(numberOfSamplesPerStep)))

	// Process buffer
	var maxX float64
	for idx := 0; idx < numberOfSteps; idx++ {
		// Offsets
		start := idx * numberOfSamplesPerStep
		end := start + numberOfSamplesPerStep

		// Get samples
		var samples []int
		if len(c.b) >= end {
			samples = c.b[start:end]
		} else {
			samples = c.b[start:]
		}

		// Compute level
		level := astikit.PCMLevel(samples)

		// Get max level
		o.MaxLevel = math.Max(o.MaxLevel, level)

		// Add data to chart
		maxX = float64(numberOfSamplesPerStep) / float64(c.s.SampleRate()) * float64(idx)
		o.Chart.Data.Datasets[0].Data = append(o.Chart.Data.Datasets[0].Data, astichartjs.DataPoint{
			X: maxX,
			Y: level,
		})
	}

	// Get current max silence level
	o.CurrentMaxSilenceLevel = c.s.MaxSilenceLevel()

	// Add current max silence level to chart
	o.Chart.Data.Datasets = append(o.Chart.Data.Datasets, astichartjs.Dataset{
		BackgroundColor: astichartjs.ChartBackgroundColorBlue,
		BorderColor:     astichartjs.ChartBorderColorBlue,
		Label:           "Current max silence level",
	})
	o.Chart.Data.Datasets[1].Data = append(o.Chart.Data.Datasets[1].Data, astichartjs.DataPoint{X: 0, Y: o.CurrentMaxSilenceLevel})
	o.Chart.Data.Datasets[1].Data = append(o.Chart.Data.Datasets[1].Data, astichartjs.DataPoint{X: maxX, Y: o.CurrentMaxSilenceLevel})

	// Get suggested max silence level
	o.SuggestedMaxSilenceLevel = 0.3 * o.MaxLevel

	// Add suggested max silence level to chart
	o.Chart.Data.Datasets = append(o.Chart.Data.Datasets, astichartjs.Dataset{
		BackgroundColor: astichartjs.ChartBackgroundColorRed,
		BorderColor:     astichartjs.ChartBorderColorRed,
		Label:           "Suggested max silence level",
	})
	o.Chart.Data.Datasets[2].Data = append(o.Chart.Data.Datasets[2].Data, astichartjs.DataPoint{X: 0, Y: o.SuggestedMaxSilenceLevel})
	o.Chart.Data.Datasets[2].Data = append(o.Chart.Data.Datasets[2].Data, astichartjs.DataPoint{X: maxX, Y: o.SuggestedMaxSilenceLevel})
	return
}
