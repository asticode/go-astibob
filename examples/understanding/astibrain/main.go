package main

import (
	"time"

	"github.com/asticode/go-astibob/abilities/understanding"
	"github.com/asticode/go-astibob/brain"
	"github.com/asticode/go-astibob/examples"
	"github.com/asticode/go-astibob/pkg/speechtotext"
	"github.com/asticode/go-astilog"
	"github.com/asticode/go-astitools/audio"
	"github.com/pkg/errors"
)

func main() {
	// Base init
	ctx := astiexamples.Init()

	// Create brain
	brain := astibrain.New(astiexamples.BrainOptions)
	defer brain.Close()

	// Create parser
	p := astispeechtotext.New(astispeechtotext.Options{
		AlphabetConfigPath:   "tmp/deepspeech/models/alphabet.txt",
		BeamWidth:            500,
		LMPath:               "tmp/deepspeech/models/lm.binary",
		LMWeight:             1.75,
		ModelPath:            "tmp/deepspeech/models/output_graph.pb",
		NCep:                 26,
		NContext:             9,
		TriePath:             "tmp/deepspeech/models/trie",
		ValidWordCountWeight: 1.00,
		WordCountWeight:      1.00,
	})

	// Create understanding
	understanding := astiunderstanding.NewAbility(p, astiunderstanding.AbilityOptions{
		SilenceDetector: astiaudio.SilenceDetectorOptions{
			AnalysisDuration:   300 * time.Millisecond,
			SilenceMinDuration: time.Second,
		},
	})

	// Learn ability
	brain.Learn(understanding, astiexamples.AbilityOptions)

	// Run the brain
	if err := brain.Run(ctx); err != nil {
		astilog.Fatal(errors.Wrap(err, "main: running brain failed"))
	}
}
