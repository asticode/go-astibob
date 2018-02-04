package astispeak

import (
	"github.com/asticode/go-astilog"
	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	"github.com/pkg/errors"
)

// Init initializes the speaker
func (s *Speaker) Init() (err error) {
	// Initialize ole
	astilog.Debug("astispeak: initializing ole")
	if err = ole.CoInitialize(0); err != nil {
		err = errors.Wrap(err, "astispeak: initializing ole failed")
		return
	}

	// Create SAPI.SpVoice object
	astilog.Debug("astispeak: creating SAPI.SpVoice ole object")
	if s.windowsIUnknown, err = oleutil.CreateObject("SAPI.SpVoice"); err != nil {
		err = errors.Wrap(err, "astispeak: creating SAPI.SpVoice ole object failed")
		return
	}

	// Get IDispatch
	astilog.Debug("astispeak: getting ole IDispatch")
	if s.windowsIDispatch, err = s.windowsIUnknown.QueryInterface(ole.IID_IDispatch); err != nil {
		err = errors.Wrap(err, "astispeak: getting ole IDispatch failed")
		return
	}
	return
}

// Close implements the io.Closer interface
func (s *Speaker) Close() (err error) {
	// Release IDispatch
	astilog.Debug("astispeak: releasing IDispatch")
	s.windowsIDispatch.Release()

	// Release IUnknown
	astilog.Debug("astispeak: releasing IUnkown")
	s.windowsIUnknown.Release()

	// Uninitialize ole
	astilog.Debug("astispeak: uninitializing ole")
	ole.CoUninitialize()
	return
}

// Say says words
func (s *Speaker) Say(i string) (err error) {
	// Init has not been executed
	if s.windowsIDispatch == nil {
		err = errors.New("astispeak: the Init() method should be called before running anything else")
		return
	}

	// Say
	var v *ole.VARIANT
	if v, err = oleutil.CallMethod(s.windowsIDispatch, "Speak", i); err != nil {
		err = errors.Wrap(err, "astispeak: calling Speak on IDispatch failed")
		return
	}

	// Clear variant
	if err = v.Clear(); err != nil {
		err = errors.Wrap(err, "astispeak: clearing variant failed")
		return
	}
	return
}
