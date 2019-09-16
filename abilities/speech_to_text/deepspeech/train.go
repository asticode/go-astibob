package deepspeech

import (
	"context"
	"io/ioutil"
	"path/filepath"

	"github.com/asticode/go-astibob/abilities/speech_to_text"
	"github.com/pkg/errors"
)

func (d *DeepSpeech) trainHashPath() string {
	return filepath.Join(filepath.Dir(d.o.ModelPath), "hash")
}

func (d *DeepSpeech) train(ctx context.Context, h []byte, speeches []speech_to_text.SpeechFile, progressFunc func(speech_to_text.Progress), p *speech_to_text.Progress) (err error) {
	// Update progress
	p.CurrentStep = trainingStep
	p.Progress = 0
	progressFunc(*p)

	// Check whether hashes are the same
	var same bool
	if same, err = d.sameHashes(h, d.trainHashPath()); err != nil {
		err = errors.Wrap(err, "deepspeech: checking whether hashes are the same failed")
		return
	} else if same {
		// Update progress
		p.Progress = 100
		progressFunc(*p)
		return
	}

	err = errors.New("deepspeech: training is not implemented")
	return

	// Store hash
	if err = ioutil.WriteFile(d.trainHashPath(), h, 0666); err != nil {
		err = errors.Wrapf(err, "deepspeech: storing hash in %s failed", d.prepareHashPath())
		return
	}

	// Update progress
	p.Progress = 100
	progressFunc(*p)
	return
}
