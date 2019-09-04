package index

import (
	"fmt"
	"net/http"
	"path/filepath"
	"sort"
	"sync"

	"github.com/asticode/go-astibob"
	"github.com/asticode/go-astilog"
	astiptr "github.com/asticode/go-astitools/ptr"
	astitemplate "github.com/asticode/go-astitools/template"
	astiworker "github.com/asticode/go-astitools/worker"
	"github.com/asticode/go-astiws"
	"github.com/pkg/errors"
)

type Options struct {
	Server        astibob.ServerOptions `toml:"server"`
	ResourcesPath string                `toml:"resources_path"`
}

type Index struct {
	c  *http.Client
	d  *astibob.Dispatcher
	mw *sync.Mutex // Locks ws
	o  Options
	t  *astitemplate.Templater
	w  *astiworker.Worker
	ws map[string]*worker // Workers indexed by name
	wu *astiws.Manager
	ww *astiws.Manager
}

// New creates a new index
func New(o Options) (i *Index, err error) {
	// Create index
	i = &Index{
		c:  &http.Client{},
		mw: &sync.Mutex{},
		o:  o,
		w:  astiworker.NewWorker(),
		ws: make(map[string]*worker),
		wu: astiws.NewManager(astiws.ManagerConfiguration{}),
		ww: astiws.NewManager(astiws.ManagerConfiguration{}),
	}

	// Default resources path
	if i.o.ResourcesPath == "" {
		i.o.ResourcesPath = "index/resources"
	}

	// Create dispatcher
	i.d = astibob.NewDispatcher(i.w.Context(), i.w.NewTask)

	// Create templater
	if i.t, err = astitemplate.NewTemplater(
		filepath.Join(i.o.ResourcesPath, "templates", "pages"),
		filepath.Join(i.o.ResourcesPath, "templates", "layouts"),
		".html",
	); err != nil {
		err = errors.Wrapf(err, "index: creating templater with resources path %s failed", i.o.ResourcesPath)
		return
	}

	// Add dispatcher handlers
	i.d.On(astibob.DispatchConditions{Name: astiptr.Str(astibob.UIPingMessage)}, i.extendUIConnection)
	i.d.On(astibob.DispatchConditions{Name: astiptr.Str(astibob.WorkerRegisterMessage)}, i.addWorker)
	i.d.On(astibob.DispatchConditions{Names: map[string]bool{
		astibob.RunnableCrashedMessage: true,
		astibob.RunnableStartedMessage: true,
		astibob.RunnableStoppedMessage: true,
	}}, i.updateRunnableStatus)
	i.d.On(astibob.DispatchConditions{Name: astiptr.Str(astibob.UIDisconnectedMessage)}, i.unregisterUI)
	i.d.On(astibob.DispatchConditions{Name: astiptr.Str(astibob.WorkerDisconnectedMessage)}, i.delWorker)
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
			astilog.Error(errors.Wrap(err, "index: closing ui clients failed"))
		}
	}

	// Close worker clients
	if i.ww != nil {
		if err := i.ww.Close(); err != nil {
			astilog.Error(errors.Wrap(err, "index: closing worker clients failed"))
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

func sendMessage(m *astibob.Message, name, label string, wm *astiws.Manager) (err error) {
	// Get clients
	var cs []*astiws.Client
	if name != "" {
		// Retrieve client from manager
		c, ok := wm.Client(name)
		if !ok {
			err = fmt.Errorf("index: client %s doesn't exist", name)
			return
		}

		// Append client
		cs = append(cs, c)
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
		astilog.Debugf("index: sending %s message to %s %s", m.Name, label, name)

		// Write
		if err = c.WriteJSON(m); err != nil {
			err = errors.Wrap(err, "index: writing JSON message failed")
			return
		}
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
