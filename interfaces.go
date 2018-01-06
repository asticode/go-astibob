package astibob

import "sync"

// interfaces is a pool of interfaces
type interfaces struct {
	i map[string]Interface
	m sync.Mutex // Locks i
}

// newInterfaces creates a new collection of interfaces
func newInterfaces() *interfaces {
	return &interfaces{i: make(map[string]Interface)}
}

// get returns a specific interface based on its name.
func (is *interfaces) get(name string) (i Interface, ok bool) {
	is.m.Lock()
	defer is.m.Unlock()
	i, ok = is.i[name]
	return
}

// set sets the interface in the pool.
func (is *interfaces) set(i Interface) {
	is.m.Lock()
	defer is.m.Unlock()
	is.i[i.Name()] = i
}
