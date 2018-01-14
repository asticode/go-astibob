package astibob

import (
	"net/http"

	"github.com/asticode/go-astiws"
)

// Interface represents required methods of an interface
type Interface interface {
	Name() string
}

// APIHandler represents an object that can handle API requests
type APIHandler interface {
	APIHandlers() map[string]http.Handler
}

// BrainWebsocketListener represents an object that can listen to a brain websocket
type BrainWebsocketListener interface {
	BrainWebsocketListeners() map[string]astiws.ListenerFunc
}

// ClientWebsocketListener represents an object that can listen to a client websocket
type ClientWebsocketListener interface {
	ClientWebsocketListeners() map[string]astiws.ListenerFunc
}

// ClientEvent represents a client event
type ClientEvent struct {
	Name    string
	Payload interface{}
}

// DispatchFunc represents a dispatch func
type DispatchFunc func(e ClientEvent)

// Dispatcher represents an object that can dispatch an event to clients
type Dispatcher interface {
	SetDispatchFunc(DispatchFunc)
}

// StaticHandler represents an object that can handle static files
type StaticHandler interface {
	StaticHandlers() map[string]http.Handler
}

// WebTemplater represents an object that can handle web templates
type WebTemplater interface {
	WebTemplates() map[string]string
}
