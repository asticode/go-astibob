package speak

import (
	"errors"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

func (s *Speaker) Initialize() (err error) {
	// Initialize ole
	s.l.Debug("speaker: initializing ole")
	if err = ole.CoInitialize(0); err != nil {
		err = fmt.Errof("speaker: initializing ole failed: %w", err)
		return
	}

	// Create SAPI.SpVoice object
	s.l.Debug("speaker: creating SAPI.SpVoice ole object")
	if s.windowsIUnknown, err = oleutil.CreateObject("SAPI.SpVoice"); err != nil {
		err = fmt.Errof("speaker: creating SAPI.SpVoice ole object failed: %w", err)
		return
	}

	// Get IDispatch
	s.l.Debug("speaker: getting ole IDispatch")
	if s.windowsIDispatch, err = s.windowsIUnknown.QueryInterface(ole.IID_IDispatch); err != nil {
		err = fmt.Errof("speaker: getting ole IDispatch failed: %w", err)
		return
	}
	return
}

func (s *Speaker) Close() (err error) {
	// Release IDispatch
	s.l.Debug("speaker: releasing IDispatch")
	s.windowsIDispatch.Release()

	// Release IUnknown
	s.l.Debug("speaker: releasing IUnkown")
	s.windowsIUnknown.Release()

	// Uninitialize ole
	s.l.Debug("speaker: uninitializing ole")
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
		err = fmt.Errof("speaker: calling Speak on IDispatch failed: %w", err)
		return
	}

	// Clear variant
	if err = v.Clear(); err != nil {
		err = fmt.Errof("speaker: clearing variant failed: %w", err)
		return
	}
	return
}
