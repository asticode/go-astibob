package speak

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/asticode/go-astikit"
	"github.com/go-ole/go-ole"
)

type Speaker struct {
	l                astikit.SeverityLogger
	o                Options
	windowsIDispatch *ole.IDispatch
	windowsIUnknown  *ole.IUnknown
}

type Options struct {
	BinaryDirPath string `toml:"binary_dir_path"`
	Voice         string `toml:"voice"`
}

func New(o Options, l astikit.StdLogger) *Speaker {
	return &Speaker{
		l: astikit.AdaptStdLogger(l),
		o: o,
	}
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
	s.l.Debugf("speaker: executing %s", strings.Join(cmd.Args, " "))
	var b []byte
	if b, err = cmd.CombinedOutput(); err != nil {
		err = fmt.Errorf("speaker: running %s failed with combined output %s: %w", strings.Join(cmd.Args, " "), b, err)
		return
	}
	return
}
