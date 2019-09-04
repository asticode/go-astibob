package index

import (
	"net/http"

	"path/filepath"

	astihttp "github.com/asticode/go-astitools/http"
	"github.com/julienschmidt/httprouter"
)

// Serve spawns the server
func (i *Index) Serve() {
	// Create router
	r := httprouter.New()

	// Static
	r.ServeFiles("/static/*filepath", http.Dir(filepath.Join(i.o.ResourcesPath, "static")))

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
	h := astihttp.ChainMiddlewares(r, astihttp.MiddlewareBasicAuth(i.o.Server.Username, i.o.Server.Password))
	h = astihttp.ChainMiddlewaresWithPrefix(h, []string{"/api/"}, astihttp.MiddlewareContentType("application/json"))

	// Serve
	i.w.Serve(i.o.Server.Addr, h)
}

func (i *Index) ok(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {}
