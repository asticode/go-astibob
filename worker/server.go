package worker

import (
	"net/http"

	"encoding/json"

	"github.com/asticode/go-astibob"
	"github.com/asticode/go-astilog"
	astihttp "github.com/asticode/go-astitools/http"
	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
)

func (w *Worker) Serve() {
	// Create router
	r := httprouter.New()

	// Add routes
	r.GET("/ok", w.ok)
	r.POST("/messages", w.handleWorkerMessage)

	// Chain middlewares
	h := astihttp.ChainMiddlewares(r, astihttp.MiddlewareContentType("application/json"))

	// Serve
	w.w.Serve(w.o.Server.Addr, h)
}

func (w *Worker) ok(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {}

func (w *Worker) handleWorkerMessage(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
	// Log
	astilog.Debug("worker: handling worker message")

	// Unmarshal
	var m astibob.Message
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		astibob.WriteHTTPError(rw, http.StatusInternalServerError, errors.Wrap(err, "worker: unmarshaling failed"))
		return
	}

	// Dispatch
	w.d.Dispatch(&m)
}
