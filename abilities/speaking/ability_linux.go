package astispeaking

import (
	"os/exec"
	"strings"

	"path/filepath"

	"github.com/asticode/go-astilog"
	"github.com/pkg/errors"
)

// say says words
func (a *Ability) say(i string) (err error) {
	// Init args
	var args []string
	if len(a.o.Voice) > 0 {
		args = append(args, "-v", a.o.Voice)
	}
	args = append(args, i)

	// Binary path
	var name = "espeak"
	if len(a.o.BinaryDirPath) > 0 {
		name = filepath.Join(a.o.BinaryDirPath, name)
	}

	// Init cmd
	var cmd = exec.Command(name, args...)

	// Exec
	astilog.Debugf("astispeaking: executing %s", strings.Join(cmd.Args, " "))
	var b []byte
	if b, err = cmd.CombinedOutput(); err != nil {
		err = errors.Wrapf(err, "astispeaking: running %s failed with combined output %s", strings.Join(cmd.Args, " "), b)
		return
	}
	return
}
