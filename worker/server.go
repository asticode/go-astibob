package worker

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/asticode/go-astibob"
	"github.com/asticode/go-astikit"
	"github.com/julienschmidt/httprouter"
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
	h := astikit.ChainHTTPMiddlewaresWithPrefix(r, []string{"/api/"}, astikit.HTTPMiddlewareContentType("application/json"))

	// Serve
	astikit.ServeHTTP(w.w, astikit.ServeHTTPOptions{
		Addr:    w.o.Server.Addr,
		Handler: h,
	})
}

func (w *Worker) ok(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {}

func (w *Worker) handleWorkerMessage(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
	// Unmarshal
	var m astibob.Message
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		astibob.WriteHTTPError(w.l, rw, http.StatusInternalServerError, fmt.Errorf("worker: unmarshaling failed: %w", err))
		return
	}

	// Log
	w.l.Debugf("worker: handling worker message %s", m.Name)

	// Dispatch
	w.d.Dispatch(&m)
}

func (w *Worker) template(c []byte) httprouter.Handle {
	return func(rw http.ResponseWriter, req *http.Request, p httprouter.Params) {
		// Write
		if _, err := rw.Write(c); err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			w.l.Error(fmt.Errorf("worker: writing template %s failed: %w", req.URL.Path, err))
			return
		}
	}
}
