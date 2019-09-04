package deepspeech

import (
	"github.com/asticode/go-astibob/abilities/speech_to_text"
	"github.com/pkg/errors"
)

func (d *DeepSpeech) train(speeches []speech_to_text.SpeechFile, progressFunc func(speech_to_text.Progress), p *speech_to_text.Progress) (err error) {
	// Update progress
	p.CurrentStep = trainingStep
	p.Progress = 0
	progressFunc(*p)

	err = errors.New("deepspeech: training is not implemented")
	return

	// Update progress
	p.Progress = 100
	progressFunc(*p)
	return
}
