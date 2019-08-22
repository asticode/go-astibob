package index

import (
	"net/http"

	"path/filepath"

	"github.com/asticode/go-astitools/http"
	"github.com/julienschmidt/httprouter"
)

// Server prefixes
const (
	apiPrefix    = "/api"
	staticPrefix = "/static"
	webPrefix    = "/web"
)

// Serve spawns the server
func (i *Index) Serve() {
	// Create router
	r := httprouter.New()

	// Static
	r.ServeFiles(staticPrefix+"/bob/*filepath", http.Dir(filepath.Join(i.o.ResourcesPath, "static")))

	// Web
	r.GET("/", i.homepage)
	r.GET(webPrefix+"/*page", i.web)

	// API
	r.GET(apiPrefix+"/ok", i.ok)
	r.GET(apiPrefix+"/references", i.references)

	// Websockets
	r.GET("/websockets/ui", i.handleUIWebsocket)
	r.GET("/websockets/worker", i.handleWorkerWebsocket)

	// Chain middlewares
	h := astihttp.ChainMiddlewares(r, astihttp.MiddlewareBasicAuth(i.o.Server.Username, i.o.Server.Password))
	h = astihttp.ChainMiddlewaresWithPrefix(h, []string{webPrefix + "/"}, astihttp.MiddlewareContentType("text/html; charset=UTF-8"))
	h = astihttp.ChainMiddlewaresWithPrefix(h, []string{apiPrefix + "/"}, astihttp.MiddlewareContentType("application/json"))

	// Serve
	i.w.Serve(i.o.Server.Addr, h)
}

func (i *Index) ok(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {}
