package astibob

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/asticode/go-astibob/brain"
	"github.com/asticode/go-astilog"
	"github.com/asticode/go-astitools/http"
	"github.com/asticode/go-astiws"
	"github.com/gorilla/websocket"
	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
)

// brainsServer is a server for the brains
type brainsServer struct {
	*server
	brains    *brains
	clientsWs *astiws.Manager
}

// newBrainsServer creates a new brains server.
func newBrainsServer(brains *brains, brainsWs *astiws.Manager, clientsWs *astiws.Manager, o ServerOptions) (s *brainsServer) {
	// Create server
	s = &brainsServer{
		brains:    brains,
		clientsWs: clientsWs,
		server:    newServer("brains", brainsWs, o),
	}

	// Init router
	var r = httprouter.New()

	// Websocket
	r.GET("/websocket", s.handleWebsocketGET)

	// Chain middlewares
	var h = astihttp.ChainMiddlewares(r, astihttp.MiddlewareBasicAuth(o.Username, o.Password))

	// Set handler
	s.setHandler(h)
	return
}

// handleWebsocketGET handles the websockets.
func (s *brainsServer) handleWebsocketGET(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
	if err := s.ws.ServeHTTP(rw, r, s.adaptWebsocketClient); err != nil {
		if v, ok := errors.Cause(err).(*websocket.CloseError); !ok || v.Code != websocket.CloseNormalClosure {
			astilog.Error(errors.Wrapf(err, "astibob: handling websocket on %s failed", s.s.Addr))
		}
		return
	}
}

// ClientAdapter returns the client adapter.
func (s *brainsServer) adaptWebsocketClient(c *astiws.Client) {
	s.ws.AutoRegisterClient(c)
	c.AddListener(astibrain.WebsocketEventNameRegister, s.handleWebsocketRegistered)
}

// handleWebsocketRegistered handles the registered websocket event
func (s *brainsServer) handleWebsocketRegistered(c *astiws.Client, eventName string, payload json.RawMessage) error {
	// Unmarshal payload
	var ip astibrain.APIRegister
	if err := json.Unmarshal(payload, &ip); err != nil {
		astilog.Error(errors.Wrapf(err, "astibob: unmarshaling %s into %#v failed", payload, ip))
		return nil
	}

	// Create brain
	var b = newBrain(ip.Name, c)

	// Loop through abilities
	for _, pa := range ip.Abilities {
		// Create ability
		var a = newAbility(pa.Name, pa.IsOn)

		// Add ability
		b.set(a)
	}

	// Add brain
	s.brains.set(b)

	// Adapt ws client
	c.AddListener(astiws.EventNameDisconnect, s.handleWebsocketDisconnected(b))
	c.AddListener(astibrain.WebsocketEventNameAbilityStarted, s.handleWebsocketAbilityToggle(b))
	c.AddListener(astibrain.WebsocketEventNameAbilityStopped, s.handleWebsocketAbilityToggle(b))
	c.AddListener(astibrain.WebsocketEventNameAbilityCrashed, s.handleWebsocketAbilityToggle(b))

	// Log
	astilog.Infof("astibob: brain %s has registered", b.name)

	// Dispatch event to brain
	dispatchWsEventToClient(c, astibrain.WebsocketEventNameRegistered, nil)

	// Dispatch event to clients
	dispatchWsEventToManager(s.clientsWs, clientsWebsocketEventNameBrainRegistered, newAPIBrain(b))
	return nil
}

// handleWebsocketDisconnected handles the disconnected websocket event
func (s *brainsServer) handleWebsocketDisconnected(b *brain) astiws.ListenerFunc {
	return func(c *astiws.Client, eventName string, payload json.RawMessage) error {
		// Delete brain
		s.brains.del(b)

		// Log
		astilog.Infof("astibob: brain %s has disconnected", b.name)

		// Dispatch event to clients
		dispatchWsEventToManager(s.clientsWs, clientsWebsocketEventNameBrainDisconnected, newAPIBrain(b))

		// Unregister client
		s.ws.UnregisterClient(c)
		return nil
	}
}

// handleWebsocketAbilityToggle handles the ability toggle websocket event
func (s *brainsServer) handleWebsocketAbilityToggle(b *brain) astiws.ListenerFunc {
	return func(c *astiws.Client, eventName string, payload json.RawMessage) error {
		// Decode payload
		var name string
		if err := json.Unmarshal(payload, &name); err != nil {
			astilog.Error(errors.Wrapf(err, "astibob: json unmarshaling %s payload %#v failed", eventName, payload))
			return nil
		}

		// Retrieve ability
		a, ok := b.ability(name)
		if !ok {
			astilog.Error(fmt.Errorf("astibob: unknown ability %s for brain %s", name, b.name))
			return nil
		}

		// Get event name
		var e string
		if eventName == astibrain.WebsocketEventNameAbilityStarted {
			e = clientsWebsocketEventNameAbilityStarted
			a.isOn = true
		} else {
			e = clientsWebsocketEventNameAbilityStopped
			a.isOn = false
		}

		// Create payload
		var p = newAPIAbility(a)
		p.BrainName = b.name

		// Dispatch event to clients
		dispatchWsEventToManager(s.clientsWs, e, p)
		return nil
	}
}
