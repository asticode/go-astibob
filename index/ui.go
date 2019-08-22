package index

import (
	"net/http"

	"encoding/json"

	"time"

	"encoding/base64"
	"fmt"

	"github.com/asticode/go-astibob"
	"github.com/asticode/go-astilog"
	astiptr "github.com/asticode/go-astitools/ptr"
	"github.com/asticode/go-astiws"
	"github.com/gorilla/websocket"
	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
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
			// Create message
			var m *astibob.Message
			if m, err = astibob.NewEventUIDisconnectedMessage(from, nil, name); err != nil {
				err = errors.Wrap(err, "astibob: creating message failed")
				return
			}

			// Dispatch
			i.d.Dispatch(m)
			return
		})

		// Register client
		i.wu.RegisterClient(name, c)

		// Log
		astilog.Infof("astibob: ui %s has connected", name)

		// Create message
		var m *astibob.Message
		if m, err = astibob.NewEventUIWelcomeMessage(from, &astibob.Identifier{
			Name: astiptr.Str(name),
			Type: astibob.UIIdentifierType,
		}, name); err != nil {
			err = errors.Wrap(err, "index: creating message failed")
			return
		}

		// Dispatch
		i.d.Dispatch(m)
		return
	}); err != nil {
		if v, ok := errors.Cause(err).(*websocket.CloseError); !ok || (v.Code != websocket.CloseNoStatusReceived && v.Code != websocket.CloseNormalClosure) {
			astilog.Error(errors.Wrap(err, "index: handling ui websocket failed"))
		}
		return
	}
}

func (i *Index) handleUIMessage(p []byte) (err error) {
	// Log
	astilog.Debugf("index: handling ui message %s", p)

	// Unmarshal
	m := astibob.NewMessage()
	if err = json.Unmarshal(p, m); err != nil {
		err = errors.Wrap(err, "index: unmarshaling failed")
		return
	}

	// Dispatch
	i.d.Dispatch(m)
	return
}

func (i *Index) sendMessageToUI(m *astibob.Message) (err error) {
	// Log
	astilog.Debugf("index: sending %s message to ui", m.Name)

	// Names is specified
	if m.To != nil && m.To.Name != nil {
		// Retrieve client from manager
		c, ok := i.wu.Client(*m.To.Name)
		if !ok {
			err = fmt.Errorf("index: client %s doesn't exist", *m.To.Name)
			return
		}

		// Write
		if err = c.WriteJSON(m); err != nil {
			err = errors.Wrap(err, "worker: writing JSON message failed")
			return
		}
		return
	}
	return
}

func (i *Index) unregisterUI(m *astibob.Message) (err error) {
	// Parse payload
	var name string
	if name, err = astibob.ParseEventUIDisconnectedPayload(m); err != nil {
		err = errors.Wrap(err, "index: parsing message payload failed")
		return
	}

	// Unregister client
	i.wu.UnregisterClient(name)

	// Log
	astilog.Infof("index: ui %s has disconnected", name)
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
		err = errors.Wrap(err, "index: extending connection failed")
		return
	}
	return
}

func (i *Index) homepage(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
	http.Redirect(rw, r, webPrefix+"/index", http.StatusPermanentRedirect)
}

func (i *Index) web(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
	// Get template name
	var name = p.ByName("page") + ".html"
	if _, ok := i.t.Template(name); !ok {
		name = "/errors/404.html"
	}

	// Get template data
	data, code := i.templateData(name)

	// Write header
	rw.WriteHeader(code)

	// Get template
	t, _ := i.t.Template(name)

	// Execute template
	if err := t.Execute(rw, data); err != nil {
		astilog.Error(errors.Wrapf(err, "index: executing %s template with data %#v failed", name, data))
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
	astibob.APIWriteData(rw, APIReferences{Websocket: APIWebsocket{
		Addr:       "ws://" + i.o.Server.Addr + "/websockets/ui",
		PingPeriod: astiws.PingPeriod,
	}})
}
