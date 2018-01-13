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

// WebsocketListener represents an object that can listen to a websocket
type WebsocketListener interface {
	WebsocketListeners() map[string]astiws.ListenerFunc
}

// WebTemplater represents an object that can handle web templates
type WebTemplater interface {
	WebTemplates() map[string]string
}
