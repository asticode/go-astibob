package index

import (
	"encoding/json"
	"net/http"

	"github.com/asticode/go-astibob"
	"github.com/asticode/go-astilog"
	astihttp "github.com/asticode/go-astitools/http"
	"github.com/asticode/go-astiws"
	"github.com/gorilla/websocket"
	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
)

// Serve spawns the server
func (i *Index) Serve() {
	// Create router
	r := httprouter.New()

	// Add routes
	r.GET("/ok", i.ok)
	r.GET("/api/workers", i.listWorkers)
	r.GET("/websockets/worker", i.handleWorkerWebsocket)

	// Chain middlewares
	h := astihttp.ChainMiddlewares(r, astihttp.MiddlewareBasicAuth(i.o.Server.Username, i.o.Server.Password))

	// Serve
	i.w.Serve(i.o.Server.Addr, h)
}

func (i *Index) ok(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {}

func (i *Index) listWorkers(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {}

func (i *Index) handleWorkerWebsocket(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
	if err := i.ww.ServeHTTP(rw, r, func(c *astiws.Client) error {
		// Register client
		ck := clientStateKey(c)
		i.ww.RegisterClient(ck, c)

		// Set message handler
		c.SetMessageHandler(i.handleWorkerMessage(ck))
		return nil
	}); err != nil {
		if v, ok := errors.Cause(err).(*websocket.CloseError); !ok || v.Code != websocket.CloseNormalClosure {
			astilog.Error(errors.Wrap(err, "index: handling worker websocket failed"))
		}
		return
	}
}

func (i *Index) handleWorkerMessage(clientKey string) astiws.MessageHandler {
	return func(p []byte) (err error) {
		// Unmarshal
		m := astibob.NewMessage()
		if err = json.Unmarshal(p, m); err != nil {
			err = errors.Wrap(err, "index: unmarshaling failed")
			return
		}

		// Add client to state
		m.State[clientMessageStateKey] = clientKey

		// Dispatch
		i.d.Dispatch(m)
		return
	}
}
