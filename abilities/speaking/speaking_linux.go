package astispeaking

import (
	"os/exec"
	"strings"

	"github.com/asticode/go-astilog"
	"github.com/pkg/errors"
)

// say says words
func (s *Speaking) say(i string) (err error) {
	// Init args
	var args []string
	if len(s.o.Voice) > 0 {
		args = append(args, "-v", s.o.Voice)
	}
	args = append(args, i)

	// Init cmd
	var cmd = exec.Command(s.o.BinaryPath, args...)

	// Exec
	astilog.Debugf("astispeaking: executing %s", strings.Join(cmd.Args, " "))
	var b []byte
	if b, err = cmd.CombinedOutput(); err != nil {
		err = errors.Wrapf(err, "astispeaking: running %s failed with combined output %s", strings.Join(cmd.Args, " "), b)
		return
	}
	return
}
