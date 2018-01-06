package astispeaking

import (
	"github.com/asticode/go-astilog"
	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	"github.com/pkg/errors"
)

// Init implements the astibrain.Initializer interface
func (a *Ability) Init() (err error) {
	// Initialize ole
	astilog.Debug("astispeaking: initializing ole")
	if err = ole.CoInitialize(0); err != nil {
		err = errors.Wrap(err, "astispeaking: initializing ole failed")
		return
	}

	// Create SAPI.SpVoice object
	astilog.Debug("astispeaking: creating SAPI.SpVoice ole object")
	if a.windowsIUnknown, err = oleutil.CreateObject("SAPI.SpVoice"); err != nil {
		err = errors.Wrap(err, "astispeaking: creating SAPI.SpVoice ole object failed")
		return
	}

	// Get IDispatch
	astilog.Debug("astispeaking: getting ole IDispatch")
	if a.windowsIDispatch, err = a.windowsIUnknown.QueryInterface(ole.IID_IDispatch); err != nil {
		err = errors.Wrap(err, "astispeaking: getting ole IDispatch failed")
		return
	}
	return
}

// Close implements the io.Closer interface
func (a *Ability) Close() (err error) {
	// Release IDispatch
	astilog.Debug("astispeaking: releasing IDispatch")
	a.windowsIDispatch.Release()

	// Release IUnknown
	astilog.Debug("astispeaking: releasing IUnkown")
	a.windowsIUnknown.Release()

	// Uninitialize ole
	astilog.Debug("astispeaking: uninitializing ole")
	ole.CoUninitialize()
	return
}

// say says words
func (a *Ability) say(i string) (err error) {
	// Get muted attribute
	a.m.Lock()
	m := a.isMuted
	a.m.Unlock()

	// Do nothing if muted
	if m {
		return
	}

	// Init has not been executed
	if a.windowsIDispatch == nil {
		err = errors.New("astispeaking: the Init() method should be called before running anything else")
		return
	}

	// Say
	var v *ole.VARIANT
	if v, err = oleutil.CallMethod(a.windowsIDispatch, "Speak", i); err != nil {
		err = errors.Wrap(err, "astispeaking: calling Speak on IDispatch failed")
		return
	}

	// Clear variant
	if err = v.Clear(); err != nil {
		err = errors.Wrap(err, "astispeaking: clearing variant failed")
		return
	}
	return
}
