package astispeechtotext

import (
	"github.com/asticode/go-astideepspeech"
	"github.com/asticode/go-astilog"
	"github.com/asticode/go-astitools/audio"
	"github.com/pkg/errors"
	"os"
)

// SpeechToText represents an object capable of doing speech to text operations
type SpeechToText struct {
	m *astideepspeech.Model
	c Configuration
}

// Configuration represents a configuration
type Configuration struct {
	AlphabetConfigPath   string  `toml:"alphabet_config_path"`
	BeamWidth            int     `toml:"beam_width"`
	LMPath               string  `toml:"lm_path"`
	LMWeight             float64 `toml:"lm_weight"`
	ModelPath            string  `toml:"model_path"`
	NCep                 int     `toml:"ncep"`
	NContext             int     `toml:"ncontext"`
	TriePath             string  `toml:"trie_path"`
	ValidWordCountWeight float64 `toml:"valid_word_count_weight"`
	WordCountWeight      float64 `toml:"word_count_weight"`
}

// New creates a new speech to text parser
func New(c Configuration) (s *SpeechToText) {
	// Create speech to text
	s = &SpeechToText{c: c}

	// Only do the following if the model exists
	if _, err := os.Stat(c.ModelPath); err == nil {
		// Create model
		s.m = astideepspeech.New(c.ModelPath, c.NCep, c.NContext, c.AlphabetConfigPath, c.BeamWidth)

		// Enable decoder with lm
		if len(c.LMPath) > 0 {
			s.m.EnableDecoderWithLM(c.AlphabetConfigPath, c.LMPath, c.TriePath, c.LMWeight, c.WordCountWeight, c.ValidWordCountWeight)
		}
	} else {
		astilog.Debugf("astispeechtotext: %s doesn't exist, skipping model creation", c.ModelPath)
	}
	return
}

// Close implements the io.Closer interface
func (s *SpeechToText) Close() error {
	// Close model
	if s.m != nil {
		astilog.Debugf("astispeechtotext: closing model")
		if err := s.m.Close(); err != nil {
			astilog.Error(errors.Wrap(err, "astispeechtotext: closing model failed"))
		}
	}
	return nil
}

// SpeechToText implements the astiunderstanding.SpeechToText interface
func (s *SpeechToText) SpeechToText(samples []int32, sampleRate, significantBits int) (text string, err error) {
	// Model has not been set
	if s.m == nil {
		return
	}

	// Convert sample rate
	if samples, err = astiaudio.ConvertSampleRate(samples, sampleRate, 16000); err != nil {
		err = errors.Wrap(err, "astispeechtotext: converting sample rate failed")
		return
	}

	// Loop through samples
	var samples16 = make([]int16, len(samples))
	for idx := 0; idx < len(samples); idx++ {
		// Convert bit depth
		var sample int32
		if sample, err = astiaudio.ConvertBitDepth(samples[idx], significantBits, 16); err != nil {
			err = errors.Wrap(err, "astispeechtotext: converting bit depth failed")
			return
		}

		// Append sample
		samples16[idx] = int16(sample)
	}

	// Speech to text
	text = s.m.SpeechToText(samples16, len(samples16), 16000)
	return
}
