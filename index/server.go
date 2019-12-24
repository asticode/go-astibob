package index

import (
	"net/http"

	"github.com/asticode/go-astikit"
	"github.com/julienschmidt/httprouter"
)

// Serve spawns the server
func (i *Index) Serve() {
	// Create router
	r := httprouter.New()

	// Statics
	for p, h := range i.r.statics() {
		r.GET("/static"+p, h)
	}

	// Web
	r.GET("/", i.homepage)
	r.GET("/web/*page", i.web)

	// API
	r.GET("/api/ok", i.ok)
	r.GET("/api/references", i.references)

	// Websockets
	r.GET("/websockets/ui", i.handleUIWebsocket)
	r.GET("/websockets/worker", i.handleWorkerWebsocket)

	// Runnable
	for _, m := range []string{http.MethodDelete, http.MethodGet, http.MethodPatch, http.MethodPost} {
		r.Handle(m, "/workers/:worker/runnables/:runnable/routes/*path", i.runnableRoutes)
	}
	r.GET("/workers/:worker/runnables/:runnable/web/*path", i.runnableWeb)

	// Chain middlewares
	h := astikit.ChainHTTPMiddlewares(r, astikit.HTTPMiddlewareBasicAuth(i.o.Server.Username, i.o.Server.Password))
	h = astikit.ChainHTTPMiddlewaresWithPrefix(h, []string{"/api/"}, astikit.HTTPMiddlewareContentType("application/json"))

	// Serve
	astikit.ServeHTTP(i.w, astikit.ServeHTTPOptions{
		Addr:    i.o.Server.Addr,
		Handler: h,
	})
}

func (i *Index) ok(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {}
