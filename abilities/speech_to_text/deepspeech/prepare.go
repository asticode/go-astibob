package deepspeech

import (
	"context"
	"encoding/csv"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"github.com/asticode/go-astibob/abilities/speech_to_text"
	"github.com/asticode/go-astitools/audio"
	"github.com/cryptix/wav"
	"github.com/pkg/errors"
)

func (d *DeepSpeech) prepare(ctx context.Context, h []byte, speeches []speech_to_text.SpeechFile, progressFunc func(speech_to_text.Progress), p *speech_to_text.Progress) (err error) {
	// Update progress
	p.CurrentStep = preparingStep
	p.Progress = 0
	progressFunc(*p)

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

		// Choose index
		var i *index
		if float64(idx) < float64(len(speeches))*0.8 {
			i = train
		} else if float64(idx) < float64(len(speeches))*0.9 {
			i = dev
		} else {
			i = test
		}

		// Write csv line
		if err = i.w.Write([]string{path, strconv.Itoa(int(fi.Size())), s.Text}); err != nil {
			err = errors.Wrap(err, "deepspeech: writing csv line failed")
			return
		}
		i.w.Flush()

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

func (d *DeepSpeech) prepareHashPath() string {
	return filepath.Join(d.o.PrepareDirPath, "hash")
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
	path = filepath.Join(d.o.PrepareDirPath, filepath.Base(s.Path))

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
	c := astiaudio.NewSampleRateConverter(float64(r.GetSampleRate()), float64(o.SampleRate), func(i int32) (err error) {
		// Convert bit depth
		if i, err = astiaudio.ConvertBitDepth(i, int(r.GetFile().SignificantBits), int(o.SignificantBits)); err != nil {
			err = errors.Wrap(err, "deepspeech: converting bit depth failed")
			return
		}

		// Create sample
		var s []byte
		for idx := 0; idx < int(o.SignificantBits/8); idx++ {
			s = append(s, byte(i>>uint(idx*8)&0xff))
		}

		// Write
		if err = w.WriteSample(s); err != nil {
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
