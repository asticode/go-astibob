package main

import (
	"context"
	"io"
	"math/rand"
	"os"
	"time"

	"github.com/asticode/go-astibob"
	"github.com/asticode/go-astibob/abilities/understanding"
	"github.com/asticode/go-astibob/examples"
	"github.com/asticode/go-astilog"
	"github.com/cryptix/wav"
	"github.com/pkg/errors"
)

var ctxDispatch context.Context
var cancelDispatch context.CancelFunc

func main() {
	// Base init
	ctx := astiexamples.Init()

	// Create bob
	bob, err := astibob.New(astiexamples.BobOptions)
	if err != nil {
		astilog.Fatal(errors.Wrap(err, "main: creating bob failed"))
	}
	defer bob.Close()

	// Get files
	files, err := files()
	if err != nil {
		astilog.Fatal(errors.Wrap(err, "main: getting files failed"))
	}

	// Create understanding
	understanding := astiunderstanding.NewInterface()

	// Add listeners
	bob.On(astibob.EventNameAbilityStarted, func(e astibob.Event) bool {
		if e.Ability != nil && e.Ability.Name == understanding.Name() {
			startSamples(bob, understanding, files[rand.Intn(len(files))])
		}
		return false
	})
	bob.On(astibob.EventNameAbilityStopped, func(e astibob.Event) bool {
		if e.Ability != nil && e.Ability.Name == understanding.Name() {
			stopSamples()
		}
		return false
	})

	// Add callback
	understanding.OnAnalysis(func(text string) error {
		astilog.Debugf("main: received analysis: %s", text)
		return nil
	})

	// Declare
	bob.Declare(understanding)

	// Run Bob
	if err = bob.Run(ctx); err != nil {
		astilog.Fatal(errors.Wrap(err, "main: running bob failed"))
	}
}

type file struct {
	filename        string
	samples         []int32
	sampleRate      int
	significantBits int
}

func files() (files []file, err error) {
	for _, filename := range []string{
		"tmp/deepspeech/audio/2830-3980-0043.wav",
		"tmp/deepspeech/audio/4507-16021-0012.wav",
		"tmp/deepspeech/audio/8455-210777-0068.wav",
	} {
		var f file
		if f, err = parseFile(filename); err != nil {
			return
		}
		files = append(files, f)
	}
	return
}

func parseFile(filename string) (o file, err error) {
	// Stat file
	var i os.FileInfo
	if i, err = os.Stat(filename); err != nil {
		err = errors.Wrapf(err, "main: stating %s failed", filename)
		return
	}

	// Open file
	var f *os.File
	if f, err = os.Open(filename); err != nil {
		err = errors.Wrapf(err, "main: opening %s failed", filename)
		return
	}
	defer f.Close()

	// Create reader
	var r *wav.Reader
	if r, err = wav.NewReader(f, i.Size()); err != nil {
		err = errors.Wrap(err, "main: creating wav reader failed")
		return
	}

	// Update file
	o = file{
		filename:        filename,
		sampleRate:      int(r.GetFile().SampleRate),
		significantBits: int(r.GetFile().SignificantBits),
	}

	// Read
	var s int32
	for {
		// Read sample
		if s, err = r.ReadSample(); err != nil {
			if err == io.EOF {
				err = nil
				break
			}
			err = errors.Wrap(err, "main: reading sample failed")
			return
		}

		// Append sample
		o.samples = append(o.samples, s)
	}
	return
}

func startSamples(bob *astibob.Bob, i *astiunderstanding.Interface, file file) {
	// Reset ctx
	ctxDispatch, cancelDispatch = context.WithCancel(context.Background())
	defer cancelDispatch()

	// Create ticker
	t := time.NewTicker(time.Second)
	defer t.Stop()

	// Loop
	var idx int
	var silencesCount int
	for {
		select {
		case <-t.C:
			// No more samples to send
			var buf []int32
			if idx >= len(file.samples)-1 {
				// Enough silences sent
				if silencesCount >= 5 {
					return
				}

				// Add silences
				silencesCount++
				buf = make([]int32, file.sampleRate)
			} else {
				// Create buffer
				if len(file.samples[idx:]) > file.sampleRate {
					buf = make([]int32, file.sampleRate)
					copy(buf, file.samples[idx:idx+file.sampleRate])
					idx += file.sampleRate
				} else {
					buf = make([]int32, len(file.samples)-idx)
					copy(buf, file.samples[idx:])
					idx = len(file.samples) - 1
				}
			}

			// Send samples
			bob.Exec(i.Samples(buf, file.sampleRate, file.significantBits, 1))
		case <-ctxDispatch.Done():
			return
		}
	}
}

func stopSamples() {
	if cancelDispatch != nil {
		cancelDispatch()
	}
}
