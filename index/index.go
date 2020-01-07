package index

import (
	"fmt"
	"net/http"
	"sort"
	"sync"

	"github.com/asticode/go-astibob"
	"github.com/asticode/go-astikit"
	"github.com/asticode/go-astiws"
)

type Options struct {
	Server astibob.ServerOptions `toml:"server"`
}

type Index struct {
	c  *http.Client
	d  *astibob.Dispatcher
	l  astikit.SeverityLogger
	mu *sync.Mutex // Locks us
	mw *sync.Mutex // Locks ws
	o  Options
	r  *resources
	t  *astikit.Templater
	us map[string]map[string]bool // UI message names indexed by message --> ui
	w  *astikit.Worker
	ws map[string]*worker // Workers indexed by name
	wu *astiws.Manager
	ww *astiws.Manager
}

// New creates a new index
func New(o Options, l astikit.StdLogger) (i *Index, err error) {
	// Create index
	i = &Index{
		c:  &http.Client{},
		l:  astikit.AdaptStdLogger(l),
		mu: &sync.Mutex{},
		mw: &sync.Mutex{},
		o:  o,
		t:  astikit.NewTemplater(),
		us: make(map[string]map[string]bool),
		w:  astikit.NewWorker(astikit.WorkerOptions{Logger: l}),
		ws: make(map[string]*worker),
		wu: astiws.NewManager(astiws.ManagerConfiguration{}, l),
		ww: astiws.NewManager(astiws.ManagerConfiguration{}, l),
	}

	// Add resources
	i.r = newResources(i.l)

	// Create dispatcher
	i.d = astibob.NewDispatcher(i.w.Context(), i.w.NewTask, i.l)

	// Loop through layouts
	for _, c := range i.r.layouts() {
		i.t.AddLayout(c)
	}

	// Loop through templates
	for p, c := range i.r.templates() {
		i.t.AddTemplate(p, c)
	}

	// Add dispatcher handlers
	i.d.On(astibob.DispatchConditions{Names: map[string]bool{
		astibob.RunnableCrashedMessage: true,
		astibob.RunnableStartedMessage: true,
		astibob.RunnableStoppedMessage: true,
	}}, i.updateRunnableStatus)
	i.d.On(astibob.DispatchConditions{Name: astikit.StrPtr(astibob.UIDisconnectedMessage)}, i.unregisterUI)
	i.d.On(astibob.DispatchConditions{Name: astikit.StrPtr(astibob.UIPingMessage)}, i.extendUIConnection)
	i.d.On(astibob.DispatchConditions{Name: astikit.StrPtr(astibob.UIRegisterMessage)}, i.registerUI)
	i.d.On(astibob.DispatchConditions{Name: astikit.StrPtr(astibob.WorkerDisconnectedMessage)}, i.delWorker)
	i.d.On(astibob.DispatchConditions{Name: astikit.StrPtr(astibob.WorkerRegisterMessage)}, i.addWorker)
	i.d.On(astibob.DispatchConditions{To: &astibob.Identifier{Types: map[string]bool{
		astibob.RunnableIdentifierType: true,
		astibob.WorkerIdentifierType:   true,
	}}}, i.sendMessageToWorker)
	i.d.On(astibob.DispatchConditions{To: &astibob.Identifier{Type: astibob.UIIdentifierType}}, i.sendMessageToUI)
	return
}

// Close closes the index properly
func (i *Index) Close() error {
	// Close dispatcher
	i.d.Close()

	// Close ui clients
	if i.wu != nil {
		if err := i.wu.Close(); err != nil {
			i.l.Error(fmt.Errorf("index: closing ui clients failed: %w", err))
		}
	}

	// Close worker clients
	if i.ww != nil {
		if err := i.ww.Close(); err != nil {
			i.l.Error(fmt.Errorf("index: closing worker clients failed: %w", err))
		}
	}
	return nil
}

// HandleSignals handles signals
func (i *Index) HandleSignals() {
	i.w.HandleSignals()
}

// Wait waits for the index to be stopped
func (i *Index) Wait() {
	i.w.Wait()
}

// On makes sure to handle messages with specific conditions
func (i *Index) On(c astibob.DispatchConditions, h astibob.MessageHandler) {
	i.d.On(c, h)
}

func sendMessage(l astikit.SeverityLogger, m *astibob.Message, label string, wm *astiws.Manager, names ...string) (err error) {
	// Get clients
	var cs []*astiws.Client
	if len(names) > 0 {
		// Loop through names
		for _, name := range names {
			// Retrieve client from manager
			c, ok := wm.Client(name)
			if !ok {
				err = fmt.Errorf("index: client %s doesn't exist", name)
				return
			}

			// Append client
			cs = append(cs, c)
		}
	} else {
		// Loop through clients
		wm.Clients(func(_ interface{}, c *astiws.Client) (err error) {
			cs = append(cs, c)
			return
		})
	}

	// Loop through clients
	for _, c := range cs {
		// Log
		l.Debugf("index: sending %s message to %s with client %p", m.Name, label, c)

		// Write
		if err = c.WriteJSON(m); err != nil {
			err = fmt.Errorf("index: writing JSON message failed: %w", err)
			return
		}
	}
	return
}

func (i *Index) uiMessageNames() (ms []string) {
	// Lock
	i.mu.Lock()
	defer i.mu.Unlock()

	// Loop through message names
	for n := range i.us {
		ms = append(ms, n)
	}
	return
}

func (i *Index) workers() (ws []astibob.Worker) {
	// Lock
	i.mw.Lock()
	defer i.mw.Unlock()

	// Get keys
	var ks []string
	for n := range i.ws {
		ks = append(ks, n)
	}

	// Sort keys
	sort.Strings(ks)

	// Loop through keys
	for _, k := range ks {
		// Append
		ws = append(ws, i.ws[k].toMessage())
	}
	return
}
