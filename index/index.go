package index

import (
	"fmt"
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

// Vars
var (
	from = astibob.Identifier{Type: astibob.IndexIdentifierType}
)

type Options struct {
	Server        astibob.ServerOptions `toml:"server"`
	ResourcesPath string                `toml:"resources_path"`
}

type Index struct {
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
	i.d = astibob.NewDispatcher(i.w.NewTask)

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
	i.d.On(astibob.DispatchConditions{Name: astiptr.Str(astibob.CmdUIPingMessage)}, i.extendUIConnection)
	i.d.On(astibob.DispatchConditions{Name: astiptr.Str(astibob.CmdWorkerRegisterMessage)}, i.addWorker)
	i.d.On(astibob.DispatchConditions{Names: map[string]bool{
		astibob.EventAbilityCrashedMessage: true,
		astibob.EventAbilityStartedMessage: true,
		astibob.EventAbilityStoppedMessage: true,
	}}, i.updateAbilityStatus)
	i.d.On(astibob.DispatchConditions{Name: astiptr.Str(astibob.EventUIDisconnectedMessage)}, i.unregisterUI)
	i.d.On(astibob.DispatchConditions{Name: astiptr.Str(astibob.EventWorkerDisconnectedMessage)}, i.delWorker)
	i.d.On(astibob.DispatchConditions{
		From: &astibob.Identifier{Types: map[string]bool{
			astibob.IndexIdentifierType: true,
			astibob.UIIdentifierType:    true,
		}},
		To: &astibob.Identifier{Type: astibob.AbilityIdentifierType},
	}, i.sendMessageToAbility)
	i.d.On(astibob.DispatchConditions{
		From: &astibob.Identifier{Types: map[string]bool{
			astibob.AbilityIdentifierType: true,
			astibob.IndexIdentifierType:   true,
		}},
		To: &astibob.Identifier{Type: astibob.UIIdentifierType},
	}, i.sendMessageToUI)
	i.d.On(astibob.DispatchConditions{
		From: &astibob.Identifier{Type: astibob.IndexIdentifierType},
		To:   &astibob.Identifier{Type: astibob.WorkerIdentifierType},
	}, i.sendMessageToWorkers)
	return
}

// Close closes the index properly
func (i *Index) Close() error {
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

func sendMessage(m *astibob.Message, wm *astiws.Manager) (err error) {
	// Get clients
	var cs []*astiws.Client
	if m.To != nil && m.To.Name != nil {
		// Retrieve client from manager
		c, ok := wm.Client(*m.To.Name)
		if !ok {
			err = fmt.Errorf("index: client %s doesn't exist", *m.To.Name)
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
		// Write
		if err = c.WriteJSON(m); err != nil {
			err = errors.Wrap(err, "worker: writing JSON message failed")
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
