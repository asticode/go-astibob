package astibob

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/asticode/go-astibob/brain"
	"github.com/asticode/go-astilog"
	"github.com/asticode/go-astitools/http"
	"github.com/asticode/go-astitools/template"
	"github.com/asticode/go-astiws"
	"github.com/gorilla/websocket"
	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
)

// Server patterns
const (
	serverPatternAPI = "/api"
	serverPatternWeb = "/web"
)

// Clients websocket events
const (
	clientsWebsocketEventNameAbilityStart      = "ability.start"
	clientsWebsocketEventNameAbilityStarted    = "ability.started"
	clientsWebsocketEventNameAbilityStop       = "ability.stop"
	clientsWebsocketEventNameAbilityStopped    = "ability.stopped"
	clientsWebsocketEventNameBrainRegistered   = "brain.registered"
	clientsWebsocketEventNameBrainDisconnected = "brain.disconnected"
	clientsWebsocketEventNamePing              = "ping"
)

// clientsServer is a server for the clients
type clientsServer struct {
	*server
	brains     *brains
	interfaces *interfaces
	stopFunc   func()
	templater  *astitemplate.Templater
}

// newClientsServer creates a new clients server.
func newClientsServer(t *astitemplate.Templater, b *brains, cWs *astiws.Manager, interfaces *interfaces, stopFunc func(), o Options) (s *clientsServer) {
	// Create server
	s = &clientsServer{
		brains:     b,
		interfaces: interfaces,
		server:     newServer("clients", cWs, o.ClientsServer),
		stopFunc:   stopFunc,
		templater:  t,
	}

	// Init router
	var r = httprouter.New()

	// Static files
	r.ServeFiles("/static/*filepath", http.Dir(filepath.Join(o.ResourcesDirectory, "static")))

	// Web
	r.GET("/", s.handleHomepageGET)
	r.GET(serverPatternWeb+"/*page", s.handleWebGET)

	// Websockets
	r.GET("/websocket", s.handleWebsocketGET)

	// API
	r.GET(serverPatternAPI+"/bob", s.handleAPIBobGET)
	r.GET(serverPatternAPI+"/bob/stop", s.handleAPIBobStopGET)
	r.GET(serverPatternAPI+"/ok", s.handleAPIOKGET)
	r.GET(serverPatternAPI+"/references", s.handleAPIReferencesGET)
	r.GET(serverPatternAPI+"/brains/:brain/abilities/:ability/*path", s.handleAPICustomGET)

	// Chain middlewares
	var h = astihttp.ChainMiddlewares(r, astihttp.MiddlewareBasicAuth(o.ClientsServer.Username, o.ClientsServer.Password))
	h = astihttp.ChainMiddlewaresWithPrefix(h, []string{serverPatternWeb + "/", serverPatternAPI + "/"}, astihttp.MiddlewareTimeout(o.ClientsServer.Timeout))
	h = astihttp.ChainMiddlewaresWithPrefix(h, []string{serverPatternWeb + "/"}, astihttp.MiddlewareContentType("text/html; charset=UTF-8"))
	h = astihttp.ChainMiddlewaresWithPrefix(h, []string{serverPatternAPI + "/"}, astihttp.MiddlewareContentType("application/json"))

	// Set handler
	s.setHandler(h)
	return
}

// handleHomepageGET handles the homepage.
func (s *clientsServer) handleHomepageGET(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
	http.Redirect(rw, r, serverPatternWeb+"/index", http.StatusPermanentRedirect)
}

// handleWebGET handles the Web pages.
func (s *clientsServer) handleWebGET(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
	// Check if template exists
	var name = p.ByName("page") + ".html"
	if _, ok := s.templater.Template(name); !ok {
		name = "/errors/404.html"
	}

	// Get data
	var code = http.StatusOK
	var data interface{}
	data = s.templateData(r, p, &name, &code)

	// Write header
	rw.WriteHeader(code)

	// Execute template
	tpl, _ := s.templater.Template(name)
	if err := tpl.Execute(rw, data); err != nil {
		astilog.Error(errors.Wrapf(err, "astibob: executing %s template with data %#v failed", name, data))
		return
	}
}

// TemplateDataAbilityWebTemplate represents ability web template data
type TemplateDataAbilityWebTemplate struct {
	AbilityKey                    string
	AbilityAPIBasePattern         string
	AbilityWebsocketBaseEventName string
	BrainKey                      string
}

// templateData returns a template data.
func (s *clientsServer) templateData(r *http.Request, p httprouter.Params, name *string, code *int) (data interface{}) {
	// Switch on name
	switch *name {
	case "/errors/404.html":
		*code = http.StatusNotFound
	default:
		// Ability web template
		if matches := regexpAbilityWebTemplatePattern.FindAllStringSubmatch(*name, -1); len(matches) > 0 && len(matches[0]) >= 3 {
			t := TemplateDataAbilityWebTemplate{
				AbilityKey: matches[0][2],
				BrainKey:   matches[0][1],
			}
			t.AbilityAPIBasePattern = fmt.Sprintf(serverPatternAPI+"/brains/%s/abilities/%s", t.BrainKey, t.AbilityKey)
			t.AbilityWebsocketBaseEventName = clientAbilityWebsocketBaseEventName(t.BrainKey, t.AbilityKey)
			data = t
		}
	}
	return
}

// handleWebsocketGET handles the websockets.
func (s *clientsServer) handleWebsocketGET(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
	if err := s.ws.ServeHTTP(rw, r, s.adaptWebsocketClient); err != nil {
		if v, ok := errors.Cause(err).(*websocket.CloseError); !ok || (v.Code != websocket.CloseNoStatusReceived && v.Code != websocket.CloseNormalClosure) {
			astilog.Error(errors.Wrapf(err, "astibob: handling websocket on %s failed", s.s.Addr))
		}
		return
	}
}

// clientAbilityWebsocketBaseEventName returns the client ability websocket base event name
func clientAbilityWebsocketBaseEventName(brainKey, abilityKey string) string {
	return fmt.Sprintf("brain.%s.ability.%s", brainKey, abilityKey)
}

// clientAbilityWebsocketEventName returns the client ability websocket event name
func clientAbilityWebsocketEventName(brainKey, abilityKey, eventName string) string {
	return fmt.Sprintf("%s.%s", clientAbilityWebsocketBaseEventName(brainKey, abilityKey), eventName)
}

// ClientAdapter returns the client adapter.
func (s *clientsServer) adaptWebsocketClient(c *astiws.Client) {
	// Register client
	s.ws.AutoRegisterClient(c)

	// Add default listeners
	c.AddListener(astiws.EventNameDisconnect, s.handleWebsocketDisconnected)
	c.AddListener(clientsWebsocketEventNameAbilityStart, s.handleWebsocketAbilityToggle)
	c.AddListener(clientsWebsocketEventNameAbilityStop, s.handleWebsocketAbilityToggle)
	c.AddListener(clientsWebsocketEventNamePing, s.handleWebsocketPing)

	// Loop through brains
	s.brains.brains(func(b *brain) error {
		// Loop through abilities
		b.abilities(func(a *ability) error {
			// Fetch interface
			i, ok := s.interfaces.get(a.name)
			if !ok {
				return nil
			}

			// Add client websocket listener
			if v, ok := i.(ClientWebsocketListener); ok {
				for n, l := range v.ClientWebsocketListeners() {
					c.AddListener(clientAbilityWebsocketEventName(b.key, a.key, n), l)
				}
			}
			return nil
		})
		return nil
	})
}

// handleWebsocketDisconnected handles the disconnected websocket event
func (s *clientsServer) handleWebsocketDisconnected(c *astiws.Client, eventName string, payload json.RawMessage) error {
	s.ws.UnregisterClient(c)
	return nil
}

// handleWebsocketPing handles the ping websocket event
func (s *clientsServer) handleWebsocketPing(c *astiws.Client, eventName string, payload json.RawMessage) error {
	if err := c.HandlePing(); err != nil {
		astilog.Error(errors.Wrap(err, "handling ping failed"))
	}
	return nil
}

// handleWebsocketAbilityToggle handles the ability toggle websocket events
func (s *clientsServer) handleWebsocketAbilityToggle(c *astiws.Client, eventName string, payload json.RawMessage) error {
	// Decode payload
	var e EventAbility
	if err := json.Unmarshal(payload, &e); err != nil {
		astilog.Error(errors.Wrapf(err, "astibob: json unmarshaling %s payload %#v failed", eventName, payload))
		return nil
	}

	// Retrieve brain
	b, ok := s.brains.brain(e.BrainName)
	if !ok {
		astilog.Error(fmt.Errorf("astibob: unknown brain %s", e.BrainName))
		return nil
	}

	// Retrieve ability
	_, ok = b.ability(e.Name)
	if !ok {
		astilog.Error(fmt.Errorf("astibob: unknown ability %s for brain %s", e.Name, b.name))
		return nil
	}

	// Get event name
	var eventNameBrain = astibrain.WebsocketEventNameAbilityStop
	if eventName == clientsWebsocketEventNameAbilityStart {
		eventNameBrain = astibrain.WebsocketEventNameAbilityStart
	}

	// Dispatch to brain
	dispatchWsEventToClient(b.ws, eventNameBrain, e.Name)
	return nil
}

// APIError represents an API error.
type APIError struct {
	Message string `json:"message"`
}

// APIWriteError writes an API error
func APIWriteError(rw http.ResponseWriter, code int, err error) {
	rw.WriteHeader(code)
	astilog.Error(err)
	if err := json.NewEncoder(rw).Encode(APIError{Message: err.Error()}); err != nil {
		astilog.Error(errors.Wrap(err, "astibob: json encoding failed"))
	}
}

// APIWrite writes API data
func APIWrite(rw http.ResponseWriter, data interface{}) {
	if err := json.NewEncoder(rw).Encode(data); err != nil {
		APIWriteError(rw, http.StatusInternalServerError, errors.Wrap(err, "astibob: json encoding failed"))
		return
	}
}

// handleAPIBobGET returns Bob's information.
func (s *clientsServer) handleAPIBobGET(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
	APIWrite(rw, newEventBob(s.brains))
}

// handleAPIBobStopGET stops Bob.
func (s *clientsServer) handleAPIBobStopGET(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
	s.stopFunc()
	rw.WriteHeader(http.StatusNoContent)
}

// handleAPIOKGET returns the ok status.
func (s *clientsServer) handleAPIOKGET(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
	rw.WriteHeader(http.StatusNoContent)
}

// APIReferences represents the references.
type APIReferences struct {
	WsURL        string `json:"ws_url"`
	WsPingPeriod int    `json:"ws_ping_period"` // In seconds
}

// handleAPIReferencesGET returns the references.
func (s *clientsServer) handleAPIReferencesGET(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
	APIWrite(rw, APIReferences{
		WsURL:        "ws://" + s.o.PublicAddr + "/websocket",
		WsPingPeriod: int(astiws.PingPeriod.Seconds()),
	})
}

// handleAPICustomGET returns the custom API handler
func (s *clientsServer) handleAPICustomGET(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
	// Fetch brain
	b, ok := s.brains.brainByKey(p.ByName("brain"))
	if !ok {
		astilog.Errorf("astibob: unknown brain key %s", p.ByName("brain"))
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	// Fetch ability
	a, ok := b.abilityByKey(p.ByName("ability"))
	if !ok {
		astilog.Errorf("astibob: unknown ability key %s for brain key %s", p.ByName("ability"), p.ByName("brain"))
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	// Fetch handler
	h, ok := a.apiHandler(p.ByName("path"))
	if !ok {
		astilog.Errorf("astibob: unknown API handler %s for ability key %s and brain key %s", p.ByName("path"), p.ByName("ability"), p.ByName("brain"))
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	// Execute handler
	h.ServeHTTP(rw, r)
}
