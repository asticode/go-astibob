package astibob

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"text/template"

	"github.com/asticode/go-astilog"
	"github.com/asticode/go-astitools/http"
	"github.com/asticode/go-astitools/template"
	"github.com/asticode/go-astiws"
	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
)

// Web socket event names.
const (
	websocketEventNameAbilityCrashed = "ability.crashed"
	websocketEventNameAbilityOff     = "ability.off"
	websocketEventNameAbilityOn      = "ability.on"
	websocketEventNamePing           = "ping"
)

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

// runServer runs the server.
func (b *Bob) runServer(ctx context.Context, cancel context.CancelFunc, o Options) (err error) {
	// Init router
	var r = httprouter.New()

	// Static files
	r.ServeFiles("/static/*filepath", http.Dir(filepath.Join(o.ResourcesDirectory, "static")))

	// Parse templates
	var t map[string]*template.Template
	if t, err = astitemplate.ParseDirectoryWithLayouts(filepath.Join(o.ResourcesDirectory, "templates", "pages"), filepath.Join(o.ResourcesDirectory, "templates", "layouts"), ".html"); err != nil {
		err = errors.Wrapf(err, "astibob: parsing templates in resources directory %s failed", o.ResourcesDirectory)
		return
	}

	// Web
	r.GET("/", b.handleHomepageGET)
	r.GET("/web/*page", astihttp.ChainRouterMiddlewares(
		b.handleWebGET(t),
		astihttp.RouterMiddlewareTimeout(o.ServerTimeout),
		astihttp.RouterMiddlewareContentType("text/html; charset=UTF-8"),
	))

	// Websocket
	r.GET("/websocket", b.handleWebsocketGET)

	// API
	r.GET("/api/bob", astihttp.ChainRouterMiddlewares(b.handleAPIBobGET, astihttp.RouterMiddlewareContentType("application/json")))
	r.GET("/api/bob/stop", b.handleAPIBobStopGET(cancel))
	r.GET("/api/references", astihttp.ChainRouterMiddlewares(b.handleAPIReferencesGET, astihttp.RouterMiddlewareContentType("application/json")))

	// Abilities
	b.abilities(func(a *ability) error {
		a.adaptRouter(r)
		return nil
	})

	// Chain middlewares
	var h = astihttp.ChainMiddlewares(r, astihttp.MiddlewareBasicAuth(o.ServerUsername, o.ServerPassword))

	// Init server
	var s = &http.Server{Addr: o.ServerAddr, Handler: h}

	// Handle shutdown
	go func() {
		select {
		case <-ctx.Done():
			astilog.Infof("astibob: shutting down server serving on %s", s.Addr)
			if err := s.Shutdown(context.Background()); err != nil {
				astilog.Error(errors.Wrapf(err, "shutting down server serving on %s failed", s.Addr))
			}
		}
	}()

	// Serve
	// Do not wrap error here since we want to be able to detect the http.ErrServerClosed error
	astilog.Infof("astibob: serving on %s", s.Addr)
	return s.ListenAndServe()
}

// handleHomepageGET handles the homepage.
func (b *Bob) handleHomepageGET(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
	http.Redirect(rw, r, "/web/index", http.StatusPermanentRedirect)
}

// handleWebGET handles the Web pages.
func (b *Bob) handleWebGET(t map[string]*template.Template) httprouter.Handle {
	return func(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
		// Check if template exists
		var name = p.ByName("page") + ".html"
		if _, ok := t[name]; !ok {
			name = "/errors/404.html"
		}

		// Get data
		var code = http.StatusOK
		var data interface{}
		data = b.templateData(r, p, &name, &code)

		// Write header
		rw.WriteHeader(code)

		// Execute template
		if err := t[name].Execute(rw, data); err != nil {
			astilog.Error(errors.Wrapf(err, "astibob: executing %s template with data %#v failed", name, data))
			return
		}
	}
}

// templateData returns a template data.
func (b *Bob) templateData(r *http.Request, p httprouter.Params, name *string, code *int) (data interface{}) {
	// Switch on name
	switch *name {
	case "/errors/404.html":
		*code = http.StatusNotFound
	case "/index.html":
	}
	return
}

// handleWebsocketGET handles the websockets.
func (b *Bob) handleWebsocketGET(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
	if err := b.ws.ServeHTTP(rw, r, b.adaptWebsocketClient); err != nil {
		astilog.Error(errors.Wrap(err, "astibob: handling websocket failed"))
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
}

// ClientAdapter returns the client adapter.
func (b *Bob) adaptWebsocketClient(c *astiws.Client) {
	k := b.ws.AutoRegisterClient(c)
	c.AddListener(websocketEventNamePing, func(c *astiws.Client, eventName string, payload json.RawMessage) error {
		return c.HandlePing()
	})
	c.AddListener(astiws.EventNameDisconnect, func(c *astiws.Client, eventName string, payload json.RawMessage) (err error) {
		b.ws.UnregisterClient(k)
		return
	})
}

// APIBob represents Bob.
type APIBob struct {
	Abilities map[string]APIAbility `json:"abilities,omitempty"`
}

// APIAbility represents an ability.
type APIAbility struct {
	AutoStart bool   `json:"auto_start"`
	IsOn      bool   `json:"is_on"`
	Name      string `json:"name"`
}

// handleAPIBobGET returns Bob's information.
func (b *Bob) handleAPIBobGET(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
	// Init data
	d := APIBob{Abilities: make(map[string]APIAbility)}

	// Loop through abilities
	b.abilities(func(a *ability) error {
		d.Abilities[a.key] = APIAbility{
			AutoStart: a.o.AutoStart,
			IsOn:      a.t.isOn(),
			Name:      a.name,
		}
		return nil
	})

	// Write
	APIWrite(rw, d)
}

// handleAPIBobStopGET stops Bob.
func (b *Bob) handleAPIBobStopGET(cancel context.CancelFunc) httprouter.Handle {
	return func(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
		cancel()
	}
}

// APIReferences represents the references.
type APIReferences struct {
	WsPingPeriod int `json:"ws_ping_period"` // In seconds
}

// handleAPIReferencesGET returns the references.
func (b *Bob) handleAPIReferencesGET(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
	APIWrite(rw, APIReferences{
		WsPingPeriod: int(astiws.PingPeriod.Seconds()),
	})
}
