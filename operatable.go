package astibob

import (
	"sync"

	"github.com/julienschmidt/httprouter"
)

type Operatable interface {
	Routes() map[string]map[string]httprouter.Handle // Indexed by path --> method
	Templates() map[string][]byte                    // Indexed by name
}

type BaseOperatable struct {
	mr *sync.Mutex // Locks rs
	mt *sync.Mutex // Locks ts
	rs map[string]map[string]httprouter.Handle
	ts map[string][]byte
}

func NewBaseOperatable() *BaseOperatable {
	return &BaseOperatable{
		mr: &sync.Mutex{},
		mt: &sync.Mutex{},
		rs: make(map[string]map[string]httprouter.Handle),
		ts: make(map[string][]byte),
	}
}

func (o *BaseOperatable) Routes() map[string]map[string]httprouter.Handle {
	o.mr.Lock()
	defer o.mr.Unlock()
	return o.rs
}

func (o *BaseOperatable) Templates() map[string][]byte {
	o.mt.Lock()
	defer o.mt.Unlock()
	return o.ts
}

func (o *BaseOperatable) AddRoute(path, method string, h httprouter.Handle) {
	// Lock
	o.mr.Lock()
	defer o.mr.Unlock()

	// Path doesn't exist
	if _, ok := o.rs[path]; !ok {
		o.rs[path] = make(map[string]httprouter.Handle)
	}

	// Add handler
	o.rs[path][method] = h
}

func (o *BaseOperatable) AddTemplate(n string, c []byte) {
	o.mt.Lock()
	defer o.mt.Unlock()
	o.ts[n] = c
}
