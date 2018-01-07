package astispeechtotext

import (
	"github.com/asticode/go-astideepspeech"
	"github.com/asticode/go-astilog"
	"github.com/pkg/errors"
)

// SpeechToText represents an object capable of doing speech to text operations
type SpeechToText struct {
	m *astideepspeech.Model
	o Options
}

// Options represents options
type Options struct {
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
func New(o Options) (s *SpeechToText) {
	// Create speech to text
	s = &SpeechToText{
		m: astideepspeech.New(o.ModelPath, o.NCep, o.NContext, o.AlphabetConfigPath, o.BeamWidth),
		o: o,
	}

	// Enable decoder with lm
	if len(o.LMPath) > 0 {
		s.m.EnableDecoderWithLM(o.AlphabetConfigPath, o.LMPath, o.TriePath, o.LMWeight, o.WordCountWeight, o.ValidWordCountWeight)
	}
	return
}

// Close implements the io.Closer interface
func (s *SpeechToText) Close() error {
	// Close model
	astilog.Debugf("astispeechtotext: closing model")
	if err := s.m.Close(); err != nil {
		astilog.Error(errors.Wrap(err, "astispeechtotext: closing model failed"))
	}
	return nil
}

// SpeechToText implements the astiunderstanding.SpeechToText interface
func (s *SpeechToText) SpeechToText(buffer []int32, bufferSize, sampleRate, significantBits int) string {
	// Convert to 16 bits
	// TODO Move to astiaudio
	var samples = make([]int16, bufferSize)
	var sample int16
	for idx := 0; idx < bufferSize; idx++ {
		if significantBits == 16 {
			sample = int16(buffer[idx])
		} else if significantBits > 16 {
			sample = int16(buffer[idx] >> uint((significantBits - 16)))
		} else {
			sample = int16(buffer[idx] << uint((16 - significantBits)))
		}
		samples[idx] = sample
	}

	// Speech to text
	return s.m.SpeechToText(samples, bufferSize, sampleRate)
}
