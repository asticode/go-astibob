package deepspeech

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/asticode/go-astibob/abilities/speech_to_text"
	"github.com/asticode/go-astilog"
	astiexec "github.com/asticode/go-astitools/exec"
	"github.com/pkg/errors"
)

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

	// Create args
	args := d.o.TrainingArgs
	if args == nil {
		args = make(map[string]string)
	}

	// Add mandatory args
	args["train_files"] = filepath.Join(d.o.PrepareDirPath, "train.csv")
	args["dev_files"] = filepath.Join(d.o.PrepareDirPath, "dev.csv")
	args["test_files"] = filepath.Join(d.o.PrepareDirPath, "test.csv")
	args["alphabet_config_path"] = d.o.AlphabetPath
	args["lm_binary_path"] = d.o.LMPath
	args["lm_trie_path"] = d.o.TriePath
	args["audio_sample_rate"] = strconv.Itoa(deepSpeechSampleRate)

	// Create command
	cmd := exec.CommandContext(ctx, d.o.ClientPath, argsToSlice(args)...)

	// Intercept stderr
	var stderr [][]byte
	cmd.Stderr = astiexec.NewStdWriter(func(i []byte) { stderr = append(stderr, i) })

	// Intercept stdout
	cmd.Stdout = astiexec.NewStdWriter(func(i []byte) { astilog.Warnf("stdout: %s", i) })

	// Run
	astilog.Debugf("deepspeech: running %s", strings.Join(cmd.Args, " "))
	if err = cmd.Run(); err != nil {
		var m string
		if len(stderr) > 0 {
			m = fmt.Sprintf(" with stderr:\n\n%s\n\n", bytes.Join(stderr, []byte("\n")))
		}
		err = errors.Wrapf(err, "deepspeech: running %s failed%s", strings.Join(cmd.Args, " "), m)
		return
	}

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

func (d *DeepSpeech) trainHashPath() string {
	return filepath.Join(filepath.Dir(d.o.ModelPath), "hash")
}

func argsToSlice(args map[string]string) (o []string) {
	for k, v := range args {
		o = append(o, "--"+k)
		if v != "" {
			o = append(o, v)
		}
	}
	return
}
