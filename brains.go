package astibob

import "sync"

// brains is a pool of brains
type brains struct {
	b map[string]*brain
	m sync.Mutex // Locks b
}

// newBrains creates a new collection of brains
func newBrains() *brains {
	return &brains{b: make(map[string]*brain)}
}

// brain returns a specific brain based on its name.
func (bs *brains) brain(name string) (b *brain, ok bool) {
	bs.m.Lock()
	defer bs.m.Unlock()
	b, ok = bs.b[name]
	return
}

// brains loops through brains and execute a function on each of them.
// If an error is returned by the function, the loop is stopped.
func (bs *brains) brains(fn func(b *brain) error) (err error) {
	bs.m.Lock()
	defer bs.m.Unlock()
	for _, b := range bs.b {
		if err = fn(b); err != nil {
			return
		}
	}
	return
}

// del deletes the brain from the pool.
func (bs *brains) del(b *brain) {
	bs.m.Lock()
	defer bs.m.Unlock()
	delete(bs.b, b.name)
}

// set sets the brain in the pool.
func (bs *brains) set(b *brain) {
	bs.m.Lock()
	defer bs.m.Unlock()
	bs.b[b.name] = b
}
