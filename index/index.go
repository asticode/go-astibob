package index

import (
	"sync"

	"github.com/asticode/go-astibob"
	astiptr "github.com/asticode/go-astitools/ptr"
	astiworker "github.com/asticode/go-astitools/worker"
	"github.com/asticode/go-astiws"
)

// Vars
var (
	from = astibob.Identifier{Type: astibob.IndexIdentifierType}
)

type Options struct {
	Server astibob.ServerOptions `toml:"server"`
}

type Index struct {
	d  *astibob.Dispatcher
	mw *sync.Mutex // Locks ws
	o  Options
	w  *astiworker.Worker
	ws map[string]*worker // Workers indexed by name
	ww *astiws.Manager
}

// New creates a new index
func New(o Options) (i *Index) {
	i = &Index{
		d:  astibob.NewDispatcher(),
		mw: &sync.Mutex{},
		o:  o,
		w:  astiworker.NewWorker(),
		ws: make(map[string]*worker),
		ww: astiws.NewManager(astiws.ManagerConfiguration{}),
	}
	i.d.On(astibob.DispatchConditions{Name: astiptr.Str(astibob.CmdWorkerRegisterMessage)}, i.addWorker)
	i.d.On(astibob.DispatchConditions{Name: astiptr.Str(astibob.EventWorkerDisconnectedMessage)}, i.delWorker)
	i.d.On(astibob.DispatchConditions{Name: astiptr.Str(astibob.EventWorkerWelcomeMessage)}, i.sendWebsocketMessage)
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
