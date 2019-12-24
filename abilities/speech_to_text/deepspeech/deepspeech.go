package deepspeech

import (
	"context"
	"os"
	"path/filepath"

	"github.com/asticode/go-astibob/abilities/speech_to_text"
	"github.com/asticode/go-astideepspeech"
	"github.com/asticode/go-astilog"
	"github.com/pkg/errors"
)

// Steps
const (
	preparingStep = "Preparing"
	trainingStep  = "Training"
)

// Deepspeech
const (
	deepSpeechBitDepth    = 16
	deepSpeechNumChannels = 1
	deepSpeechSampleRate  = 16000
)

type DeepSpeech struct {
	m *astideepspeech.Model
	o Options
}

type Options struct {
	AlphabetPath         string            `toml:"alphabet_path"`
	BeamWidth            int               `toml:"beam_width"`
	ClientPath           string            `toml:"client_path"`
	LMPath               string            `toml:"lm_path"`
	LMWeight             float64           `toml:"lm_weight"`
	ModelPath            string            `toml:"model_path"`
	PrepareDirPath       string            `toml:"prepare_dir_path"`
	PythonBinaryPath     string            `toml:"python_binary_path"`
	TrainingArgs         map[string]string `toml:"training_args"`
	TriePath             string            `toml:"trie_path"`
	ValidWordCountWeight float64           `toml:"1.85"`
}

func New(o Options) (d *DeepSpeech) {
	// Create deepspeech
	d = &DeepSpeech{o: o}

	// Only do the following if the model path exists
	if _, err := os.Stat(d.o.ModelPath); err == nil {
		// Create model
		d.m = astideepspeech.New(o.ModelPath, o.BeamWidth)

		// Enable LM
		if o.LMPath != "" {
			d.m.EnableDecoderWithLM(o.LMPath, o.TriePath, o.LMWeight, o.ValidWordCountWeight)
		}
	}
	return
}

func (d *DeepSpeech) Init() (err error) {
	// Get absolute path
	if d.o.PrepareDirPath, err = filepath.Abs(d.o.PrepareDirPath); err != nil {
		err = errors.Wrapf(err, "deepspeech: getting absolute path of %s failed", d.o.PrepareDirPath)
		return
	}
	return
}

func (d *DeepSpeech) Close() {
	// Close the model
	if d.m != nil {
		astilog.Debug("deepspeech: closing model")
		if err := d.m.Close(); err != nil {
			astilog.Error(errors.Wrap(err, "deepspeech: closing model failed"))
		}
	}
}

func (d *DeepSpeech) Parse(samples []int, bitDepth, numChannels, sampleRate int) (t string, err error) {
	// No model
	if d.m == nil {
		return
	}

	// Create audio converter
	var ss []int16
	c := newAudioConverter(bitDepth, numChannels, sampleRate, func(s int) (err error) {
		ss = append(ss, int16(s))
		return
	})

	// Loop through samples
	for _, s := range samples {
		// Add to audio converter
		if err = c.add(s); err != nil {
			err = errors.Wrap(err, "deepspeech: adding to audio converter failed")
			return
		}
	}

	// Parse
	t = d.m.SpeechToText(ss, uint(len(ss)))
	return
}

func (d *DeepSpeech) Train(ctx context.Context, speeches []speech_to_text.SpeechFile, progressFunc func(speech_to_text.Progress)) {
	// Train
	p, err := d.handleError(ctx, speeches, progressFunc)

	// Handle error
	if err != nil {
		// Update error
		p.Error = err

		// Dispatch progress
		progressFunc(p)
		return
	}
}

func (d *DeepSpeech) handleError(ctx context.Context, speeches []speech_to_text.SpeechFile, progressFunc func(speech_to_text.Progress)) (p speech_to_text.Progress, err error) {
	// Create progress
	p = speech_to_text.Progress{
		Steps: []string{
			preparingStep,
			trainingStep,
		},
	}

	// Prepare
	if err = d.prepare(ctx, speeches, progressFunc, &p); err != nil {
		err = errors.Wrap(err, "deepspeech: preparing failed")
		return
	}

	// Train
	if err = d.train(ctx, speeches, progressFunc, &p); err != nil {
		err = errors.Wrap(err, "deepspeech: training failed")
		return
	}
	return
}
