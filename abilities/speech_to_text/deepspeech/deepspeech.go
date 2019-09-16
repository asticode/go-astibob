package deepspeech

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/asticode/go-astibob/abilities/speech_to_text"
	"github.com/asticode/go-astideepspeech"
	"github.com/asticode/go-astilog"
	astiaudio "github.com/asticode/go-astitools/audio"
	"github.com/pkg/errors"
)

// Steps
const (
	preparingStep = "Preparing"
	trainingStep  = "Training"
)

// Deepspeech constants
const (
	deepSpeechBitDepth   = 16
	deepSpeechSampleRate = 16000
)

type DeepSpeech struct {
	m *astideepspeech.Model
	o Options
}

type Options struct {
	AlphabetPath         string  `toml:"alphabet_path"`
	BeamWidth            int     `toml:"beam_width"`
	LMPath               string  `toml:"lm_path"`
	LMWeight             float64 `toml:"lm_weight"`
	ModelPath            string  `toml:"model_path"`
	NCep                 int     `toml:"ncep"`
	NContext             int     `toml:"ncontext"`
	SpeechesDirPath      string  `toml:"speeches_dir_path"`
	TriePath             string  `toml:"trie_path"`
	ValidWordCountWeight float64 `toml:"1.85"`
}

func New(o Options) (d *DeepSpeech) {
	// Create deepspeech
	d = &DeepSpeech{o: o}

	// Only do the following if the model path exists
	if _, err := os.Stat(d.o.ModelPath); err == nil {
		// Create model
		d.m = astideepspeech.New(o.ModelPath, o.NCep, o.NContext, o.AlphabetPath, o.BeamWidth)

		// Enable LM
		if o.LMPath != "" {
			d.m.EnableDecoderWithLM(o.AlphabetPath, o.LMPath, o.TriePath, o.LMWeight, o.ValidWordCountWeight)
		}
	}
	return
}

func (d *DeepSpeech) Init() (err error) {
	// Get absolute path
	if d.o.SpeechesDirPath, err = filepath.Abs(d.o.SpeechesDirPath); err != nil {
		err = errors.Wrapf(err, "deepspeech: getting absolute path of %s failed", d.o.SpeechesDirPath)
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

func (d *DeepSpeech) Parse(samples []int32, bitDepth int, sampleRate float64) (t string, err error) {
	// No model
	if d.m == nil {
		return
	}

	// Create sample rate converter
	var ss []int16
	c := astiaudio.NewSampleRateConverter(sampleRate, deepSpeechSampleRate, func(s int32) (err error) {
		// Convert bit depth
		if s, err = astiaudio.ConvertBitDepth(s, bitDepth, deepSpeechBitDepth); err != nil {
			err = errors.Wrap(err, "deepspeech: converting bit depth failed")
			return
		}

		// Append sample
		ss = append(ss, int16(s))
		return
	})

	// Loop through samples
	for _, s := range samples {
		// Add to sample rate converter
		if err = c.Add(s); err != nil {
			err = errors.Wrap(err, "deepspeech: adding sample to sample rate converter failed")
			return
		}
	}

	// Parse
	t = d.m.SpeechToText(ss, uint(len(ss)), deepSpeechSampleRate)
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

	// Get current speeches hash
	var h []byte
	if h, err = d.speechesHash(speeches); err != nil {
		err = errors.Wrap(err, "deepspeech: getting current speeches hash failed")
		return
	}

	// Prepare
	if err = d.prepare(ctx, h, speeches, progressFunc, &p); err != nil {
		err = errors.Wrap(err, "deepspeech: preparing failed")
		return
	}

	// Train
	if err = d.train(ctx, h, speeches, progressFunc, &p); err != nil {
		err = errors.Wrap(err, "deepspeech: training failed")
		return
	}
	return
}

func (d *DeepSpeech) speechesHash(speeches []speech_to_text.SpeechFile) (h []byte, err error) {
	// Marshal
	var b []byte
	if b, err = json.Marshal(speeches); err != nil {
		err = errors.Wrap(err, "deepspeech: marshaling failed")
	}

	// Create hasher
	hh := sha1.New()

	// Write
	if _, err = hh.Write(b); err != nil {
		err = errors.Wrap(err, "deepspeech: writing in hasher failed")
		return
	}

	// Sum
	h = hh.Sum(nil)
	return
}

func (d *DeepSpeech) sameHashes(h []byte, path string) (same bool, err error) {
	// Get previous hash
	var ph []byte
	if ph, err = ioutil.ReadFile(path); err != nil && !os.IsNotExist(err) {
		err = errors.Wrapf(err, "deepspeech: reading %s failed", path)
		return
	}
	err = nil

	// Hashes are the same
	if err == nil && bytes.Equal(ph, h) {
		same = true
		return
	}

	// Reset error
	err = nil
	return
}
