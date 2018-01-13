package astibob

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/asticode/go-astibob/brain"
	"github.com/asticode/go-astilog"
	"github.com/asticode/go-astitools/http"
	"github.com/asticode/go-astitools/template"
	"github.com/asticode/go-astiws"
	"github.com/gorilla/websocket"
	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
)

// brainsServer is a server for the brains
type brainsServer struct {
	*server
	brains     *brains
	clientsWs  *astiws.Manager
	dispatcher *dispatcher
	interfaces *interfaces
	templater  *astitemplate.Templater
}

// newBrainsServer creates a new brains server.
func newBrainsServer(t *astitemplate.Templater, b *brains, bWs *astiws.Manager, cWs *astiws.Manager, d *dispatcher, i *interfaces, o ServerOptions) (s *brainsServer) {
	// Create server
	s = &brainsServer{
		brains:     b,
		clientsWs:  cWs,
		dispatcher: d,
		interfaces: i,
		server:     newServer("brains", bWs, o),
		templater:  t,
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

		// Check if interface has been declared for this ability
		i, ok := s.interfaces.get(a.name)
		if ok {
			// Add custom websocket listeners
			if v, ok := i.(WebsocketListener); ok {
				for n, l := range v.WebsocketListeners() {
					c.AddListener(astibrain.WebsocketAbilityEventName(a.name, n), l)
				}
			}

			// Add UI
			if v, ok := i.(UIDisplayer); ok {
				// Set UI
				a.ui = v.UI()

				// Update homepage
				if len(a.ui.Homepage) > 0 {
					a.ui.Homepage = serverPatternWeb + s.webTemplatePattern(b.name, a.name, a.ui.Homepage)
				}

				// Add web templates
				if a.ui.WebTemplates != nil {
					for path, content := range a.ui.WebTemplates {
						if err := s.templater.Add(s.webTemplatePath(b.name, a.name, path), content); err != nil {
							astilog.Error(errors.Wrapf(err, "astibob: adding web template for brain %s, ability %s and path %s", b.name, a.name, path))
						}
					}
				}
			}
		}

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

	// Create event payload
	e := newEventBrain(b)

	// Dispatch event to clients
	dispatchWsEventToManager(s.clientsWs, clientsWebsocketEventNameBrainRegistered, e)

	// Dispatch event to GO
	s.dispatcher.dispatch(Event{Brain: e, Name: EventNameBrainRegistered})
	return nil
}

// webTemplatePattern returns the web template pattern
func (s *brainsServer) webTemplatePattern(brainName, abilityName, path string) string {
	return fmt.Sprintf("/brains/%s/abilities/%s%s", brainName, abilityName, path)
}

// webTemplatePath returns the web template path
func (s *brainsServer) webTemplatePath(brainName, abilityName, path string) string {
	return s.webTemplatePattern(brainName, abilityName, path) + ".html"
}

// handleWebsocketDisconnected handles the disconnected websocket event
func (s *brainsServer) handleWebsocketDisconnected(b *brain) astiws.ListenerFunc {
	return func(c *astiws.Client, eventName string, payload json.RawMessage) error {
		// Loop through abilities
		b.abilities(func(a *ability) error {
			// Remove web templates
			if a.ui != nil && a.ui.WebTemplates != nil {
				for path := range a.ui.WebTemplates {
					s.templater.Del(s.webTemplatePath(b.name, a.name, path))
				}
			}
			return nil
		})

		// Delete brain
		s.brains.del(b)

		// Log
		astilog.Infof("astibob: brain %s has disconnected", b.name)

		// Create event payload
		e := newEventBrain(b)

		// Dispatch event to clients
		dispatchWsEventToManager(s.clientsWs, clientsWebsocketEventNameBrainDisconnected, e)

		// Dispatch event to GO
		s.dispatcher.dispatch(Event{Brain: e, Name: EventNameBrainDisconnected})

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
		var eventNameClients, eventNameGO string
		if eventName == astibrain.WebsocketEventNameAbilityStarted {
			eventNameClients = clientsWebsocketEventNameAbilityStarted
			eventNameGO = EventNameAbilityStarted
			a.setOn(true)
		} else {
			eventNameClients = clientsWebsocketEventNameAbilityStopped
			eventNameGO = EventNameAbilityStopped
			a.setOn(false)
		}

		// Create event payload
		e := newEventAbility(a)
		e.BrainName = b.name

		// Dispatch event to clients
		dispatchWsEventToManager(s.clientsWs, eventNameClients, e)

		// Dispatch event to GO
		s.dispatcher.dispatch(Event{Ability: e, Name: eventNameGO})
		return nil
	}
}
