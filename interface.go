package astibob

import "github.com/asticode/go-astiws"

// Interface represents required methods of an interface
type Interface interface {
	Name() string
}

// UIDisplayer represents an object that can tell what should be displayed in the UI
type UIDisplayer interface {
	UI() *UI
}

// WebsocketListener represents an object that can listen to a websocket
type WebsocketListener interface {
	WebsocketListeners() map[string]astiws.ListenerFunc
}
