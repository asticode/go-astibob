package main

import (
	"io/ioutil"
	"os"

	"github.com/asticode/go-astibob"
	"github.com/asticode/go-astibob/abilities/hearing"
	"github.com/asticode/go-astibob/examples"
	"github.com/asticode/go-astilog"
	"github.com/cryptix/wav"
	"github.com/pkg/errors"
)

func main() {
	// Base init
	ctx := astiexamples.Init()

	// Create bob
	bob, err := astibob.New(astiexamples.BobOptions)
	if err != nil {
		astilog.Fatal(errors.Wrap(err, "main: creating bob failed"))
	}
	defer bob.Close()

	// Create wav files
	f, w, err := createWavFiles()
	if err != nil {
		astilog.Fatal(errors.Wrap(err, "main: initializing failed"))
	}
	defer f.Close()
	defer w.Close()

	// Create hearing
	hearing := astihearing.NewInterface()

	// Handle samples
	hearing.OnSamples(func(samples []int32) error {
		astilog.Debug("Writing samples")
		for idx, sample := range samples {
			if err := w.WriteInt32(sample); err != nil {
				astilog.Error(errors.Wrapf(err, "main: writing sample %d failed", idx))
			}
		}
		return nil
	})

	// Declare interface
	bob.Declare(hearing)

	// Run Bob
	if err = bob.Run(ctx); err != nil {
		astilog.Fatal(errors.Wrap(err, "main: running bob failed"))
	}
}

func createWavFiles() (f *os.File, w *wav.Writer, err error) {
	// Create wav file
	if f, err = ioutil.TempFile(os.TempDir(), "hearing"); err != nil {
		err = errors.Wrap(err, "main: creating wav file failed")
		return
	}
	wf := wav.File{
		Channels:        1,
		SampleRate:      16000,
		SignificantBits: 32,
	}

	// Create wav writer
	if w, err = wf.NewWriter(f); err != nil {
		err = errors.Wrap(err, "main: creating wav writer failed")
		return
	}

	// Log
	astilog.Infof("Results of the hearing ability will be written in %s", f.Name())
	return
}
