package deepspeech

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"github.com/asticode/go-astibob/abilities/speech_to_text"
	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
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
		err = fmt.Errorf("deepspeech: getting current speeches hash failed: %w", err)
		return
	}

	// Check whether hashes are the same
	var same bool
	if same, err = d.sameHashes(h, d.prepareHashPath()); err != nil {
		err = fmt.Errorf("deepspeech: checking whether hashes are the same failed: %w", err)
		return
	} else if same {
		// Update progress
		p.Progress = 100
		progressFunc(*p)
		return
	}

	// Remove directory
	if err = os.RemoveAll(d.o.PrepareDirPath); err != nil {
		err = fmt.Errorf("deepspeech: removing %s failed: %w", d.o.PrepareDirPath, err)
		return
	}

	// Create directory
	if err = os.MkdirAll(d.o.PrepareDirPath, 0755); err != nil {
		err = fmt.Errorf("deepspeech: mkdirall %s failed: %w", d.o.PrepareDirPath, err)
		return
	}

	// Create indexes
	var train, dev, test *index
	if train, dev, test, err = d.createIndexes(); err != nil {
		err = fmt.Errorf("deepspeech: creating indexes failed: %w", err)
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
			err = fmt.Errorf("deepspeech: converting audio file %s failed: %w", s.Path, err)
			return
		}

		// Stat audio file
		var fi os.FileInfo
		if fi, err = os.Stat(path); err != nil {
			err = fmt.Errorf("deepspeech: stating %s failed: %w", path, err)
			return
		}

		// Loop through indexes
		for _, i := range d.indexes(idx, speeches, train, dev, test) {
			// Write csv line
			if err = i.w.Write([]string{path, strconv.Itoa(int(fi.Size())), s.Text}); err != nil {
				err = fmt.Errorf("deepspeech: writing csv line failed: %w", err)
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
		err = fmt.Errorf("deepspeech: storing hash in %s failed: %w", d.prepareHashPath(), err)
		return
	}
	return
}

func (d *DeepSpeech) indexes(idx int, speeches []speech_to_text.SpeechFile, train, dev, test *index) (is []*index) {
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
	return
}

func (d *DeepSpeech) speechesHash(speeches []speech_to_text.SpeechFile) (h []byte, err error) {
	// Marshal
	var b []byte
	if b, err = json.Marshal(speeches); err != nil {
		err = fmt.Errorf("deepspeech: marshaling failed: %w", err)
		return
	}

	// Create hasher
	hh := sha1.New()

	// Write
	if _, err = hh.Write(b); err != nil {
		err = fmt.Errorf("deepspeech: writing in hasher failed: %w", err)
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
		err = fmt.Errorf("deepspeech: reading %s failed: %w", path, err)
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
		err = fmt.Errorf("deepspeech: creating train index failed: %w", err)
		return
	}

	// Create dev index
	if dev, err = d.createIndex(filepath.Join(d.o.PrepareDirPath, "dev.csv")); err != nil {
		err = fmt.Errorf("deepspeech: creating dev index failed: %w", err)
		return
	}

	// Create test index
	if test, err = d.createIndex(filepath.Join(d.o.PrepareDirPath, "test.csv")); err != nil {
		err = fmt.Errorf("deepspeech: creating test index failed: %w", err)
		return
	}
	return
}

func (d *DeepSpeech) createIndex(path string) (i *index, err error) {
	// Create index
	i = &index{}

	// Create csv
	if i.f, err = os.Create(path); err != nil {
		err = fmt.Errorf("deepspeech: creating %s failed: %w", path, err)
		return
	}

	// Create csv writer
	i.w = csv.NewWriter(i.f)

	// Write csv header
	if err = i.w.Write([]string{"wav_filename", "wav_filesize", "transcript"}); err != nil {
		err = fmt.Errorf("deepspeech: writing csv header failed: %w", err)
		return
	}
	i.w.Flush()
	return
}

func (d *DeepSpeech) convertAudioFile(s speech_to_text.SpeechFile) (path string, err error) {
	// Open src
	var src *os.File
	if src, err = os.Open(s.Path); err != nil {
		err = fmt.Errorf("deepspeech: opening %s failed: %w", s.Path, err)
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
		err = fmt.Errorf("deepspeech: creating %s failed: %w", path, err)
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
			err = fmt.Errorf("deepspeech: writing wav sample failed: %w", err)
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
			err = fmt.Errorf("deepspeech: getting next buffer failed: %w", err)
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
				err = fmt.Errorf("deepspeech: adding to audio converter failed: %w", err)
				return
			}
		}
	}
	return
}
