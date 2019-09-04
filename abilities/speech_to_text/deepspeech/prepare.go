package deepspeech

import (
	"encoding/csv"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/asticode/go-astibob/abilities/speech_to_text"
	astiaudio "github.com/asticode/go-astitools/audio"
	"github.com/cryptix/wav"
	"github.com/pkg/errors"
)

func (d *DeepSpeech) prepare(speeches []speech_to_text.SpeechFile, progressFunc func(speech_to_text.Progress), p *speech_to_text.Progress) (err error) {
	// Update progress
	p.CurrentStep = preparingStep
	p.Progress = 0
	progressFunc(*p)

	// Remove directory
	if err = os.RemoveAll(d.o.SpeechesDirPath); err != nil {
		err = errors.Wrapf(err, "deepspeech: removing %s failed", d.o.SpeechesDirPath)
		return
	}

	// Create directory
	if err = os.MkdirAll(d.o.SpeechesDirPath, 0755); err != nil {
		err = errors.Wrapf(err, "deepspeech: mkdirall %s failed", d.o.SpeechesDirPath)
		return
	}

	// Create csv
	var f *os.File
	if f, err = os.Create(filepath.Join(d.o.SpeechesDirPath, "index.csv")); err != nil {
		err = errors.Wrapf(err, "deepspeech: creating %s failed", filepath.Join(d.o.SpeechesDirPath, "index.csv"))
		return
	}
	defer f.Close()

	// Create csv writer
	w := csv.NewWriter(f)

	// Write csv header
	if err = w.Write([]string{"wav_filename", "wav_filesize", "transcript"}); err != nil {
		err = errors.Wrap(err, "deepspeech: writing csv header failed")
		return
	}
	w.Flush()

	// Loop through speeches
	for idx, s := range speeches {
		// Convert audio file
		var path string
		if path, err = d.convertAudioFile(s); err != nil {
			err = errors.Wrapf(err, "deepspeech: converting audio file %s failed", s.Path)
			return
		}

		// Stat audio file
		var i os.FileInfo
		if i, err = os.Stat(path); err != nil {
			err = errors.Wrapf(err, "deepspeech: stating %s failed", path)
			return
		}

		// Write csv line
		if err = w.Write([]string{path, strconv.Itoa(int(i.Size())), s.Text}); err != nil {
			err = errors.Wrap(err, "deepspeech: writing csv line failed")
			return
		}
		w.Flush()

		// Update progress
		p.Progress = float64(idx+1) / float64(len(speeches)) * 100.0
		progressFunc(*p)
	}
	return
}

func (d *DeepSpeech) convertAudioFile(s speech_to_text.SpeechFile) (path string, err error) {
	// Stat src
	var i os.FileInfo
	if i, err = os.Stat(s.Path); err != nil {
		err = errors.Wrapf(err, "deepspeech: stating %s failed", s.Path)
		return
	}

	// Open src
	var src *os.File
	if src, err = os.Open(s.Path); err != nil {
		err = errors.Wrapf(err, "deepspeech: opening %s failed", s.Path)
		return
	}
	defer src.Close()

	// Create wav reader
	var r *wav.Reader
	if r, err = wav.NewReader(src, i.Size()); err != nil {
		err = errors.Wrap(err, "deepspeech: creating wav reader failed")
		return
	}

	// Create path
	path = filepath.Join(d.o.SpeechesDirPath, filepath.Base(s.Path))

	// Create dst
	var dst *os.File
	if dst, err = os.Create(path); err != nil {
		err = errors.Wrapf(err, "deepspeech: creating %s failed", path)
		return
	}
	defer dst.Close()

	// Create wav options
	o := wav.File{
		Channels:        1,
		SampleRate:      deepSpeechSampleRate,
		SignificantBits: deepSpeechBitDepth,
	}

	// Create wav writer
	var w *wav.Writer
	if w, err = o.NewWriter(dst); err != nil {
		err = errors.Wrap(err, "deepspeech: creating wav writer failed")
		return
	}
	defer w.Close()

	// Create sample rate converter
	c := astiaudio.NewSampleRateConverter(float64(r.GetSampleRate()), float64(o.SampleRate), func(s int32) (err error) {
		// Convert bit depth
		if s, err = astiaudio.ConvertBitDepth(s, int(r.GetFile().SignificantBits), int(o.SignificantBits)); err != nil {
			err = errors.Wrap(err, "deepspeech: converting bit depth failed")
			return
		}

		// Write
		if err = w.WriteSample([]byte{byte(s & 0xff), byte(s >> 8 & 0xff)}); err != nil {
			err = errors.Wrap(err, "deepspeech: writing wav sample failed")
			return
		}
		return
	})

	// Loop through samples
	for {
		// Read sample
		var s int32
		if s, err = r.ReadSample(); err != nil {
			if err != io.EOF {
				err = errors.Wrap(err, "deepspeech: reading wav sample failed")
				return
			}
			err = nil
			break
		}

		// Add to sample rate converter
		if err = c.Add(s); err != nil {
			err = errors.Wrap(err, "deepspeech: adding sample to sample rate converter failed")
			return
		}
	}
	return
}
