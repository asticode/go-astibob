package deepspeech

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/csv"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"github.com/asticode/go-astibob/abilities/speech_to_text"
	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
	"github.com/pkg/errors"
)

// Audio formats
const (
	audioFormatPCM = 1
)

func (d *DeepSpeech) prepare(ctx context.Context, speeches []speech_to_text.SpeechFile, progressFunc func(speech_to_text.Progress), p *speech_to_text.Progress) (err error) {
	// Update progress
	p.CurrentStep = preparingStep
	p.Progress = 0
	progressFunc(*p)

	// Get current speeches hash
	var h []byte
	if h, err = d.speechesHash(speeches); err != nil {
		err = errors.Wrap(err, "deepspeech: getting current speeches hash failed")
		return
	}

	// Check whether hashes are the same
	var same bool
	if same, err = d.sameHashes(h, d.prepareHashPath()); err != nil {
		err = errors.Wrap(err, "deepspeech: checking whether hashes are the same failed")
		return
	} else if same {
		// Update progress
		p.Progress = 100
		progressFunc(*p)
		return
	}

	// Remove directory
	if err = os.RemoveAll(d.o.PrepareDirPath); err != nil {
		err = errors.Wrapf(err, "deepspeech: removing %s failed", d.o.PrepareDirPath)
		return
	}

	// Create directory
	if err = os.MkdirAll(d.o.PrepareDirPath, 0755); err != nil {
		err = errors.Wrapf(err, "deepspeech: mkdirall %s failed", d.o.PrepareDirPath)
		return
	}

	// Create indexes
	var train, dev, test *index
	if train, dev, test, err = d.createIndexes(); err != nil {
		err = errors.Wrap(err, "deepspeech: creating indexes failed")
		return
	}

	// Make sure indexes are closed properly
	defer func() {
		train.f.Close()
		dev.f.Close()
		test.f.Close()
	}()

	// Loop through speeches
	for idx, s := range speeches {
		// Check context
		if ctx.Err() != nil {
			err = ctx.Err()
			return
		}

		// Convert audio file
		var path string
		if path, err = d.convertAudioFile(s); err != nil {
			err = errors.Wrapf(err, "deepspeech: converting audio file %s failed", s.Path)
			return
		}

		// Stat audio file
		var fi os.FileInfo
		if fi, err = os.Stat(path); err != nil {
			err = errors.Wrapf(err, "deepspeech: stating %s failed", path)
			return
		}

		// Choose indexes
		var is []*index
		if len(speeches) < 10 {
			if idx == len(speeches)-1 {
				is = []*index{train, dev, test}
			} else {
				is = []*index{train}
			}
		} else {
			if float64(idx) < float64(len(speeches))*0.8 {
				is = []*index{train}
			} else if float64(idx) < float64(len(speeches))*0.9 {
				is = []*index{dev}
			} else {
				is = []*index{test}
			}
		}

		// Loop through indexes
		for _, i := range is {
			// Write csv line
			if err = i.w.Write([]string{path, strconv.Itoa(int(fi.Size())), s.Text}); err != nil {
				err = errors.Wrap(err, "deepspeech: writing csv line failed")
				return
			}
			i.w.Flush()
		}

		// Update progress
		p.Progress = float64(idx+1) / float64(len(speeches)) * 100.0
		progressFunc(*p)
	}

	// Store hash
	if err = ioutil.WriteFile(d.prepareHashPath(), h, 0666); err != nil {
		err = errors.Wrapf(err, "deepspeech: storing hash in %s failed", d.prepareHashPath())
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

func (d *DeepSpeech) prepareHashPath() string {
	return filepath.Join(d.o.PrepareDirPath, "hash")
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

type index struct {
	f *os.File
	w *csv.Writer
}

func (d *DeepSpeech) createIndexes() (train, dev, test *index, err error) {
	// Create train index
	if train, err = d.createIndex(filepath.Join(d.o.PrepareDirPath, "train.csv")); err != nil {
		err = errors.Wrap(err, "deepspeech: creating train index failed")
		return
	}

	// Create dev index
	if dev, err = d.createIndex(filepath.Join(d.o.PrepareDirPath, "dev.csv")); err != nil {
		err = errors.Wrap(err, "deepspeech: creating dev index failed")
		return
	}

	// Create test index
	if test, err = d.createIndex(filepath.Join(d.o.PrepareDirPath, "test.csv")); err != nil {
		err = errors.Wrap(err, "deepspeech: creating test index failed")
		return
	}
	return
}

func (d *DeepSpeech) createIndex(path string) (i *index, err error) {
	// Create index
	i = &index{}

	// Create csv
	if i.f, err = os.Create(path); err != nil {
		err = errors.Wrapf(err, "deepspeech: creating %s failed", path)
		return
	}

	// Create csv writer
	i.w = csv.NewWriter(i.f)

	// Write csv header
	if err = i.w.Write([]string{"wav_filename", "wav_filesize", "transcript"}); err != nil {
		err = errors.Wrap(err, "deepspeech: writing csv header failed")
		return
	}
	i.w.Flush()
	return
}

func (d *DeepSpeech) convertAudioFile(s speech_to_text.SpeechFile) (path string, err error) {
	// Open src
	var src *os.File
	if src, err = os.Open(s.Path); err != nil {
		err = errors.Wrapf(err, "deepspeech: opening %s failed", s.Path)
		return
	}
	defer src.Close()

	// Create decoder
	dc := wav.NewDecoder(src)

	// Read info
	dc.ReadInfo()

	// Create path
	path = filepath.Join(d.o.PrepareDirPath, filepath.Base(s.Path))

	// Create dst
	var dst *os.File
	if dst, err = os.Create(path); err != nil {
		err = errors.Wrapf(err, "deepspeech: creating %s failed", path)
		return
	}
	defer dst.Close()

	// Create encoder
	e := wav.NewEncoder(dst, deepSpeechSampleRate, deepSpeechBitDepth, deepSpeechNumChannels, audioFormatPCM)
	defer e.Close()

	// Create audio converter
	c := newAudioConverter(int(dc.BitDepth), int(dc.NumChans), int(dc.SampleRate), func(s int) (err error) {
		// Write
		if err = e.Write(&audio.IntBuffer{
			Data: []int{s},
			Format: &audio.Format{
				NumChannels: e.NumChans,
				SampleRate:  e.SampleRate,
			},
			SourceBitDepth: e.BitDepth,
		}); err != nil {
			err = errors.Wrap(err, "deepspeech: writing wav sample failed")
			return
		}
		return
	})

	// Loop through buffers
	b := &audio.IntBuffer{Data: make([]int, 5000)}
	for {
		// Get next buffer
		var n int
		if n, err = dc.PCMBuffer(b); err != nil {
			err = errors.Wrap(err, "deepspeech: getting next buffer failed")
			return
		}

		// Nothing written
		if n == 0 {
			break
		}

		// Loop through samples
		for idx := 0; idx < n; idx++ {
			// Add to audio converter
			if err = c.add(b.Data[idx]); err != nil {
				err = errors.Wrap(err, "deepspeech: adding to audio converter failed")
				return
			}
		}
	}
	return
}
