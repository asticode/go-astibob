package speaker

import (
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/asticode/go-astilog"
	"github.com/go-ole/go-ole"
	"github.com/pkg/errors"
)

type Speaker struct {
	o                Options
	windowsIDispatch *ole.IDispatch
	windowsIUnknown  *ole.IUnknown
}

type Options struct {
	BinaryDirPath string `toml:"binary_dir_path"`
	Voice         string `toml:"voice"`
}

func New(o Options) *Speaker {
	return &Speaker{o: o}
}

func (s *Speaker) execute(name, i string) (err error) {
	// Create args
	args := []string{i}

	// Add voice
	if s.o.Voice != "" {
		args = append([]string{"-v", s.o.Voice}, args...)
	}

	// Add binary dir path
	if s.o.BinaryDirPath != "" {
		name = filepath.Join(s.o.BinaryDirPath, name)
	}

	// Create cmd
	cmd := exec.Command(name, args...)

	// Execute cmd
	astilog.Debugf("speaker: executing %s", strings.Join(cmd.Args, " "))
	var b []byte
	if b, err = cmd.CombinedOutput(); err != nil {
		err = errors.Wrapf(err, "speaker: running %s failed with combined output %s", strings.Join(cmd.Args, " "), b)
		return
	}
	return
}
