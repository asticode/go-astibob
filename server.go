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
	c    ServerConfiguration
	name string
	s    *http.Server
	ws   *astiws.Manager
}

// ServerConfiguration is a server configuration
type ServerConfiguration struct {
	ListenAddr string                      `toml:"listen_addr"`
	Password   string                      `toml:"password"`
	PublicAddr string                      `toml:"public_addr"`
	Timeout    time.Duration               `toml:"timeout"`
	Username   string                      `toml:"username"`
	Ws         astiws.ManagerConfiguration `toml:"ws"`
}

// newServer creates a new server
func newServer(name string, ws *astiws.Manager, c ServerConfiguration) *server {
	// Create
	s := &server{
		c:    c,
		name: name,
		ws:   ws,
	}

	// Default configuration values
	if len(s.c.PublicAddr) == 0 {
		s.c.PublicAddr = s.c.ListenAddr
	}
	if s.c.Timeout == 0 {
		s.c.Timeout = 5 * time.Second
	}
	return s
}

// setHandler sets the handler
func (s *server) setHandler(h http.Handler) {
	s.s = &http.Server{Addr: s.c.ListenAddr, Handler: h}
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
