package deepspeech

import (
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/asticode/go-astibob/abilities/speech_to_text"
	"github.com/pkg/errors"
)

func (d *DeepSpeech) hashPath() string {
	return filepath.Join(d.o.SpeechesDirPath, "hash")
}

func (d *DeepSpeech) check(speeches []speech_to_text.SpeechFile, progressFunc func(speech_to_text.Progress), p *speech_to_text.Progress) (ch []byte, err error) {
	// Update progress
	p.CurrentStep = checkingStep
	p.Progress = 0
	progressFunc(*p)

	// Get current speeches hash
	if ch, err = d.speechesHash(speeches); err != nil {
		err = errors.Wrap(err, "deepspeech: getting current speeches hash failed")
		return
	}

	// Get previous speeches hash
	var ph []byte
	if ph, err = ioutil.ReadFile(d.hashPath()); err != nil && !os.IsNotExist(err) {
		err = errors.Wrapf(err, "deepspeech: reading %s failed", d.hashPath())
		return
	}
	err = nil

	// Hashes are the same, there's nothing to do here
	if err == nil && bytes.Equal(ph, ch) {
		err = errors.New("deepspeech: dataset hasn't changed since last training")
		return
	}

	// Reset error
	err = nil

	// Update progress
	p.Progress = 100
	progressFunc(*p)
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
