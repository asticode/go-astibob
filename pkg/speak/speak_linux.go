package astispeak

import (
	"os/exec"
	"strings"

	"path/filepath"

	"github.com/asticode/go-astilog"
	"github.com/pkg/errors"
)

// Init initializes the speaker
func (s *Speaker) Init() error { return nil }

// Close implements the io.Closer interface
func (s *Speaker) Close() error { return nil }

// Say says words
func (s *Speaker) Say(i string) (err error) {
	// Init args
	var args []string
	if len(s.c.Voice) > 0 {
		args = append(args, "-v", s.c.Voice)
	}
	args = append(args, i)

	// Binary path
	var name = "espeak"
	if len(s.c.BinaryDirPath) > 0 {
		name = filepath.Join(s.c.BinaryDirPath, name)
	}

	// Init cmd
	var cmd = exec.Command(name, args...)

	// Exec
	astilog.Debugf("astispeak: executing %s", strings.Join(cmd.Args, " "))
	var b []byte
	if b, err = cmd.CombinedOutput(); err != nil {
		err = errors.Wrapf(err, "astispeak: running %s failed with combined output %s", strings.Join(cmd.Args, " "), b)
		return
	}
	return
}
