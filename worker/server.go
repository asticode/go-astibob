package worker

import (
	"encoding/json"
	"fmt"
	"net/http"

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
	r.GET("/api/ok", w.ok)
	r.POST("/api/messages", w.handleWorkerMessage)

	// Loop through runnables
	w.mr.Lock()
	for _, rn := range w.rs {
		// Not operatable
		o, ok := rn.(astibob.Operatable)
		if !ok {
			continue
		}

		// Add routes
		for p, rs := range o.Routes() {
			for m, h := range rs {
				r.Handle(m, fmt.Sprintf("/runnables/%s/routes%s", rn.Metadata().Name, p), h)
			}
		}

		// Add templates
		for n, c := range o.Templates() {
			r.GET(fmt.Sprintf("/runnables/%s/templates%s", rn.Metadata().Name, n), w.template(c))
		}
	}
	w.mr.Unlock()

	// Chain middlewares
	h := astihttp.ChainMiddlewaresWithPrefix(r, []string{"/api/"}, astihttp.MiddlewareContentType("application/json"))

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

func (w *Worker) template(c []byte) httprouter.Handle {
	return func(rw http.ResponseWriter, req *http.Request, p httprouter.Params) {
		// Write
		if _, err := rw.Write(c); err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			astilog.Error(errors.Wrapf(err, "worker: writing template %s failed", req.URL.Path))
			return
		}
	}
}
