package astibob

import (
	"sync"

	"net/http"

	"mime"
	"path/filepath"

	"github.com/asticode/go-astilog"
	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
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

func ContentHandle(path string, c []byte) httprouter.Handle {
	// Get mime type
	t := mime.TypeByExtension(filepath.Ext(path))
	if t == "" {
		t = "binary"
	}
	return func(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
		// Set content type
		rw.Header().Set("Content-Type", t)

		// Write
		if _, err := rw.Write(c); err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			astilog.Error(errors.Wrapf(err, "astibob: writing %s failed", r.URL.Path))
			return
		}
	}
}

func DirHandle(path string) httprouter.Handle {
	return func(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
		req.URL.Path = ps.ByName("path")
		http.FileServer(http.Dir(path)).ServeHTTP(w, req)
	}
}
