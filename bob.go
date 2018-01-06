package astibob

import (
	"context"
	"path/filepath"
	"text/template"

	"github.com/asticode/go-astilog"
	"github.com/asticode/go-astitools/template"
	"github.com/asticode/go-astiws"
	"github.com/pkg/errors"
)

// Bob is an object handling a collection of brains.
type Bob struct {
	brains        *brains
	brainsServer  *brainsServer
	cancel        context.CancelFunc
	clientsServer *clientsServer
	ctx           context.Context
	dispatcher    *dispatcher
	o             Options
}

// Options are Bob options.
type Options struct {
	BrainsServer       ServerOptions
	ClientsServer      ServerOptions
	ResourcesDirectory string
}

// New creates a new Bob.
func New(o Options) (b *Bob, err error) {
	// Create bob
	b = &Bob{
		brains:     newBrains(),
		dispatcher: newDispatcher(),
		o:          o,
	}

	// Parse templates
	astilog.Debugf("astibob: parsing templates in %s", b.o.ResourcesDirectory)
	var t map[string]*template.Template
	if t, err = astitemplate.ParseDirectoryWithLayouts(filepath.Join(b.o.ResourcesDirectory, "templates", "pages"), filepath.Join(b.o.ResourcesDirectory, "templates", "layouts"), ".html"); err != nil {
		err = errors.Wrapf(err, "astibob: parsing templates in resources directory %s failed", b.o.ResourcesDirectory)
		return
	}

	// Create servers
	brainsWs := astiws.NewManager(o.BrainsServer.MaxMessageSize)
	clientsWs := astiws.NewManager(o.ClientsServer.MaxMessageSize)
	b.clientsServer = newClientsServer(t, b.brains, clientsWs, b.stop, o)
	b.brainsServer = newBrainsServer(b.brains, brainsWs, clientsWs, b.dispatcher, o.BrainsServer)
	return
}

// Close implements the io.Closer interface.
func (b *Bob) Close() (err error) {
	// Close brains server
	astilog.Debug("astibob: closing brains server")
	if err = b.brainsServer.Close(); err != nil {
		astilog.Error(errors.Wrap(err, "astibob: closing brains server failed"))
	}

	// Close clients server
	astilog.Debug("astibob: closing clients server")
	if err = b.clientsServer.Close(); err != nil {
		astilog.Error(errors.Wrap(err, "astibob: closing clients server failed"))
	}
	return
}

// Run runs Bob.
// This is cancellable through the ctx.
func (b *Bob) Run(ctx context.Context) (err error) {
	// Reset ctx
	b.ctx, b.cancel = context.WithCancel(ctx)
	defer b.cancel()

	// Run brains server
	var chanDone = make(chan error)
	go func() {
		if err := b.brainsServer.run(); err != nil {
			chanDone <- err
		}
	}()
	go func() {
		if err := b.clientsServer.run(); err != nil {
			chanDone <- err
		}
	}()

	// Dispatch event
	// TODO Only fire this event once servers are up and running
	b.dispatcher.dispatch(Event{Name: EventNameReady})

	// Wait for context or chanDone to be done
	select {
	case <-b.ctx.Done():
		if b.ctx.Err() != context.Canceled {
			err = errors.Wrap(err, "astibob: context error")
		}
		return
	case err = <-chanDone:
		if err != nil {
			err = errors.Wrap(err, "astibob: running servers failed")
		}
		return
	}
	return
}

// stop stops Bob
func (b *Bob) stop() {
	b.cancel()
}

// dispatchWsEventToManager dispatches a websocket event to a manager.
func dispatchWsEventToManager(ws *astiws.Manager, name string, payload interface{}) {
	ws.Loop(func(k interface{}, c *astiws.Client) {
		dispatchWsEventToClient(c, name, payload)
	})
}

// dispatchWsEventToClient dispatches a websocket event to a client.
func dispatchWsEventToClient(c *astiws.Client, name string, payload interface{}) {
	// Write
	if err := c.Write(name, payload); err != nil {
		astilog.Error(errors.Wrapf(err, "astibob: writing %s event with payload %#v to ws client %p failed", name, payload, c))
		return
	}
}

// On adds a listener to an event
func (b *Bob) On(eventName string, l Listener) {
	b.dispatcher.addListener(eventName, l)
}
