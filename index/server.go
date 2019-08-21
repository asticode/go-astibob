package index

import (
	"net/http"

	astihttp "github.com/asticode/go-astitools/http"
	"github.com/julienschmidt/httprouter"
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
