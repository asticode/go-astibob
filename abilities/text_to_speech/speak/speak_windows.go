package speak

import (
	"github.com/asticode/go-astilog"
	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	"github.com/pkg/errors"
)

func (s *Speaker) Init() (err error) {
	// Initialize ole
	astilog.Debug("speaker: initializing ole")
	if err = ole.CoInitialize(0); err != nil {
		err = errors.Wrap(err, "speaker: initializing ole failed")
		return
	}

	// Create SAPI.SpVoice object
	astilog.Debug("speaker: creating SAPI.SpVoice ole object")
	if s.windowsIUnknown, err = oleutil.CreateObject("SAPI.SpVoice"); err != nil {
		err = errors.Wrap(err, "speaker: creating SAPI.SpVoice ole object failed")
		return
	}

	// Get IDispatch
	astilog.Debug("speaker: getting ole IDispatch")
	if s.windowsIDispatch, err = s.windowsIUnknown.QueryInterface(ole.IID_IDispatch); err != nil {
		err = errors.Wrap(err, "speaker: getting ole IDispatch failed")
		return
	}
	return
}

func (s *Speaker) Close() (err error) {
	// Release IDispatch
	astilog.Debug("speaker: releasing IDispatch")
	s.windowsIDispatch.Release()

	// Release IUnknown
	astilog.Debug("speaker: releasing IUnkown")
	s.windowsIUnknown.Release()

	// Uninitialize ole
	astilog.Debug("speaker: uninitializing ole")
	ole.CoUninitialize()
	return
}

func (s *Speaker) Say(i string) (err error) {
	// Init has not been executed
	if s.windowsIDispatch == nil {
		err = errors.New("speaker: the Init() method should be called before running anything else")
		return
	}

	// Say
	var v *ole.VARIANT
	if v, err = oleutil.CallMethod(s.windowsIDispatch, "Speak", i); err != nil {
		err = errors.Wrap(err, "speaker: calling Speak on IDispatch failed")
		return
	}

	// Clear variant
	if err = v.Clear(); err != nil {
		err = errors.Wrap(err, "speaker: clearing variant failed")
		return
	}
	return
}
