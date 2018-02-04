package astispeak

import "github.com/go-ole/go-ole"

// Speaker represents a speaker
type Speaker struct {
	c Configuration

	// Windows
	windowsIDispatch *ole.IDispatch
	windowsIUnknown  *ole.IUnknown
}

// Configuration represents a speaker configuration.
type Configuration struct {
	BinaryDirPath string `toml:"binary_dir_path"`
	Voice         string `toml:"voice"`
}

// New creates a new speaker
func New(c Configuration) *Speaker {
	return &Speaker{c: c}
}
