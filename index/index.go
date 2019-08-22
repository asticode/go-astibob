package index

import (
	"sync"

	"path/filepath"

	"github.com/asticode/go-astibob"
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
		d:  astibob.NewDispatcher(),
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
	i.d.On(astibob.DispatchConditions{Name: astiptr.Str(astibob.EventUIDisconnectedMessage)}, i.unregisterUI)
	i.d.On(astibob.DispatchConditions{Name: astiptr.Str(astibob.EventWorkerDisconnectedMessage)}, i.delWorker)
	i.d.On(astibob.DispatchConditions{To: &astibob.Identifier{Type: astibob.WorkerIdentifierType}}, i.sendMessageToWorker)
	i.d.On(astibob.DispatchConditions{To: &astibob.Identifier{Type: astibob.UIIdentifierType}}, i.sendMessageToUI)
	return
}

// Close closes the index properly
func (i *Index) Close() {
	i.ww.Close()
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
