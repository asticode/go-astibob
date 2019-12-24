package deepspeech

import (
	"github.com/asticode/go-astikit"
	"github.com/pkg/errors"
)

type audioConverter struct {
	cc *astikit.PCMChannelsConverter
	fn astikit.PCMSampleFunc
	sc *astikit.PCMSampleRateConverter
}

func newAudioConverter(bitDepth, numChannels, sampleRate int, fn astikit.PCMSampleFunc) (c *audioConverter) {
	// Create converter
	c = &audioConverter{fn: fn}

	// Create channels converter
	c.cc = astikit.NewPCMChannelsConverter(numChannels, deepSpeechNumChannels, func(s int) (err error) {
		// Convert bit depth
		if s, err = astikit.ConvertPCMBitDepth(s, bitDepth, deepSpeechBitDepth); err != nil {
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
	c.sc = astikit.NewPCMSampleRateConverter(sampleRate, deepSpeechSampleRate, numChannels, func(s int) (err error) {
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
