package index

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/asticode/go-astibob"
	"github.com/asticode/go-astiws"
	"github.com/gorilla/websocket"
	"github.com/julienschmidt/httprouter"
)

func uiName(c *astiws.Client) string {
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%p", c)))
}

func (i *Index) handleUIWebsocket(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
	if err := i.wu.ServeHTTP(rw, r, func(c *astiws.Client) (err error) {
		// Set message handler
		c.SetMessageHandler(i.handleUIMessage)

		// Contrary to workers, UI can't provide proper unique names therefore we need to come up with one when it
		// connects and send it right away for future messages
		name := uiName(c)

		// Handle disconnect
		c.SetListener(astiws.EventNameDisconnect, func(_ *astiws.Client, _ string, _ json.RawMessage) (err error) {
			// Create disconnected message
			var m *astibob.Message
			if m, err = astibob.NewUIDisconnectedMessage(
				*astibob.NewIndexIdentifier(),
				nil,
				name,
			); err != nil {
				err = fmt.Errorf("astibob: creating disconnected message failed: %w", err)
				return
			}

			// Dispatch
			i.d.Dispatch(m)
			return
		})

		// Register client
		i.wu.RegisterClient(name, c)

		// Log
		i.l.Infof("astibob: ui %s has connected", name)

		// Create welcome message
		var m *astibob.Message
		if m, err = astibob.NewUIWelcomeMessage(
			*astibob.NewIndexIdentifier(),
			astibob.NewUIIdentifier(name),
			astibob.WelcomeUI{
				Name:    name,
				Workers: i.workers(),
			},
		); err != nil {
			err = fmt.Errorf("index: creating welcome message failed: %w", err)
			return
		}

		// Dispatch
		i.d.Dispatch(m)
		return
	}); err != nil {
		var e *websocket.CloseError
		if ok := errors.As(err, &e); !ok ||
			(e.Code != websocket.CloseNoStatusReceived && e.Code != websocket.CloseNormalClosure) {
			i.l.Error(fmt.Errorf("index: handling ui websocket failed: %w", err))
		}
		return
	}
}

func (i *Index) handleUIMessage(p []byte) (err error) {
	// Log
	i.l.Debugf("index: handling ui message %s", p)

	// Unmarshal
	m := astibob.NewMessage()
	if err = json.Unmarshal(p, m); err != nil {
		err = fmt.Errorf("index: unmarshaling failed: %w", err)
		return
	}

	// Dispatch
	i.d.Dispatch(m)
	return
}

func (i *Index) sendMessageToUI(m *astibob.Message) (err error) {
	// Invalid to
	if m.To == nil {
		err = errors.New("index: invalid to")
		return
	}

	// Get names
	var names []string
	if m.To.Name != nil {
		names = append(names, *m.To.Name)
	} else {
		// Lock
		i.mu.Lock()

		// No UI has requested this message
		us, ok := i.us[m.Name]
		if !ok {
			i.mu.Unlock()
			return
		}

		// Get names
		for n := range us {
			names = append(names, n)
		}

		// Unlock
		i.mu.Unlock()
	}

	// Send message
	if err = sendMessage(i.l, m, "ui", i.wu, names...); err != nil {
		err = fmt.Errorf("index: sending message failed: %w", err)
		return
	}
	return
}

func (i *Index) registerUI(m *astibob.Message) (err error) {
	// From name
	if m.From.Name == nil {
		err = errors.New("index: from name is empty")
		return
	}

	// Parse payload
	var u astibob.UI
	if u, err = astibob.ParseUIRegisterPayload(m); err != nil {
		err = fmt.Errorf("index: parsing message payload failed: %w", err)
		return
	}

	// Add message names
	i.mu.Lock()
	var ns []string
	for _, n := range u.MessageNames {
		// Message name key doesn't exist
		if _, ok := i.us[n]; !ok {
			// Create key
			i.us[n] = make(map[string]bool)

			// Append
			ns = append(ns, n)
		}

		// Add ui
		i.us[n][u.Name] = true
	}
	i.mu.Unlock()

	// Log
	i.l.Infof("index: ui %s has registered", *m.From.Name)

	// Create message
	if m, err = astibob.NewUIMessageNamesAddMessage(
		*astibob.NewIndexIdentifier(),
		&astibob.Identifier{Type: astibob.WorkerIdentifierType},
		ns,
	); err != nil {
		err = fmt.Errorf("index: creating ui message names add failed: %w", err)
		return
	}

	// Dispatch
	i.d.Dispatch(m)
	return
}

func (i *Index) unregisterUI(m *astibob.Message) (err error) {
	// Parse payload
	var name string
	if name, err = astibob.ParseUIDisconnectedPayload(m); err != nil {
		err = fmt.Errorf("index: parsing message payload failed: %w", err)
		return
	}

	// Delete message names
	i.mu.Lock()
	var ns []string
	for n := range i.us {
		// Remove ui
		delete(i.us[n], name)

		// Message name is not needed anymore
		if len(i.us[n]) == 0 {
			// Remove key
			delete(i.us, n)

			// Append
			ns = append(ns, n)
		}
	}
	i.mu.Unlock()

	// Unregister client
	i.wu.UnregisterClient(name)

	// Log
	i.l.Infof("index: ui %s has disconnected", name)

	// Create message
	if m, err = astibob.NewUIMessageNamesDeleteMessage(
		*astibob.NewIndexIdentifier(),
		&astibob.Identifier{Type: astibob.WorkerIdentifierType},
		ns,
	); err != nil {
		err = fmt.Errorf("index: creating ui message names delete failed: %w", err)
		return
	}

	// Dispatch
	i.d.Dispatch(m)
	return
}

func (i *Index) extendUIConnection(m *astibob.Message) (err error) {
	// From name
	if m.From.Name == nil {
		err = errors.New("index: from name is empty")
		return
	}

	// Retrieve client from manager
	c, ok := i.wu.Client(*m.From.Name)
	if !ok {
		err = fmt.Errorf("index: client %s doesn't exist", *m.From.Name)
		return
	}

	// Extend connection
	if err = c.ExtendConnection(); err != nil {
		err = fmt.Errorf("index: extending connection failed: %w", err)
		return
	}
	return
}

func (i *Index) homepage(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
	http.Redirect(rw, r, "/web/index", http.StatusPermanentRedirect)
}

func (i *Index) web(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
	// Get template name
	var name = p.ByName("page") + ".html"
	if _, ok := i.t.Template(name); !ok {
		name = "/errors/404.html"
	}

	// Get template data
	data, code := i.templateData(name)

	// Set content type
	rw.Header().Set("Content-Type", "text/html; charset=UTF-8")

	// Write header
	rw.WriteHeader(code)

	// Get template
	t, _ := i.t.Template(name)

	// Execute template
	if err := t.Execute(rw, data); err != nil {
		i.l.Error(fmt.Errorf("index: executing %s template with data %#v failed: %w", name, data, err))
		return
	}
}

func (i *Index) templateData(name string) (data interface{}, code int) {
	code = http.StatusOK
	switch name {
	case "/errors/404.html":
		code = http.StatusNotFound
	}
	return
}

type APIReferences struct {
	Websocket APIWebsocket `json:"websocket"`
}

type APIWebsocket struct {
	Addr       string        `json:"addr"`
	PingPeriod time.Duration `json:"ping_period"`
}

func (i *Index) references(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
	astibob.WriteHTTPData(i.l, rw, APIReferences{Websocket: APIWebsocket{
		Addr:       "ws://" + i.o.Server.Addr + "/websockets/ui",
		PingPeriod: astiws.PingPeriod,
	}})
}
