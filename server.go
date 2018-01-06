package astibob

import (
	"context"
	"net/http"
	"time"

	"github.com/asticode/go-astilog"
	"github.com/asticode/go-astiws"
	"github.com/pkg/errors"
)

// server represents a server
type server struct {
	name string
	o    ServerOptions
	s    *http.Server
	ws   *astiws.Manager
}

// ServerOptions are server options
type ServerOptions struct {
	ListenAddr string
	Password   string
	PublicAddr string
	Timeout    time.Duration
	Username   string
}

// newServer creates a new server
func newServer(name string, ws *astiws.Manager, o ServerOptions) *server {
	return &server{
		name: name,
		o:    o,
		ws:   ws,
	}
}

// setHandler sets the handler
func (s *server) setHandler(h http.Handler) {
	s.s = &http.Server{Addr: s.o.ListenAddr, Handler: h}
}

// Close implements the io.Closer interface
func (s *server) Close() (err error) {
	// Close ws
	astilog.Debugf("astibob: closing %s ws", s.name)
	if err = s.ws.Close(); err != nil {
		astilog.Error(errors.Wrapf(err, "astibob: closing %s ws failed", s.name))
	}

	// Shut down
	astilog.Debugf("astibob: shutting down %s server", s.name)
	if err = s.s.Shutdown(context.Background()); err != nil {
		astilog.Error(errors.Wrapf(err, "shutting down %s server serving failed", s.name))
	}
	return
}

// run runs the server
func (s *server) run() (err error) {
	// Run
	astilog.Infof("astibob: running %s server on %s", s.name, s.s.Addr)
	if err = s.s.ListenAndServe(); err != nil {
		err = errors.Wrapf(err, "astibob: running %s server failed")
		return
	}
	return
}
