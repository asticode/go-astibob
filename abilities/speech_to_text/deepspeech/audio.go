package deepspeech

import (
	astipcm "github.com/asticode/go-astitools/pcm"
	"github.com/pkg/errors"
)

type audioConverter struct {
	cc *astipcm.ChannelsConverter
	fn astipcm.SampleFunc
	sc *astipcm.SampleRateConverter
}

func newAudioConverter(bitDepth, numChannels, sampleRate int, fn astipcm.SampleFunc) (c *audioConverter) {
	// Create converter
	c = &audioConverter{fn: fn}

	// Create channels converter
	c.cc = astipcm.NewChannelsConverter(numChannels, deepSpeechNumChannels, func(s int) (err error) {
		// Convert bit depth
		if s, err = astipcm.ConvertBitDepth(s, bitDepth, deepSpeechBitDepth); err != nil {
			err = errors.Wrap(err, "deepspeech: converting bit depth failed")
			return
		}

		// Custom
		if err = fn(s); err != nil {
			err = errors.Wrap(err, "deepspeech: custom sample func failed")
			return
		}
		return
	})

	// Create sample rate converter
	c.sc = astipcm.NewSampleRateConverter(sampleRate, deepSpeechSampleRate, numChannels, func(s int) (err error) {
		// Add to channels converter
		if err = c.cc.Add(s); err != nil {
			err = errors.Wrap(err, "deepspeech: adding to channels converter failed")
			return
		}
		return
	})
	return
}

func (c *audioConverter) add(s int) (err error) {
	// Add to sample rate converter
	if err = c.sc.Add(s); err != nil {
		err = errors.Wrap(err, "deepspeech: adding to sample rate converter failed")
		return
	}
	return
}
