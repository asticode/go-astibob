package astibob

import "sync"

// brains is a pool of brains
type brains struct {
	k map[string]*brain // Indexed by key
	m sync.Mutex        // Locks b
	n map[string]*brain // Indexed by name
}

// newBrains creates a new collection of brains
func newBrains() *brains {
	return &brains{
		k: make(map[string]*brain),
		n: make(map[string]*brain),
	}
}

// brain returns a specific brain based on its name.
func (bs *brains) brain(name string) (b *brain, ok bool) {
	bs.m.Lock()
	defer bs.m.Unlock()
	b, ok = bs.n[name]
	return
}

// brainByKey returns a specific brain based on its key.
func (bs *brains) brainByKey(name string) (b *brain, ok bool) {
	bs.m.Lock()
	defer bs.m.Unlock()
	b, ok = bs.k[name]
	return
}

// brains loops through brains and execute a function on each of them.
// If an error is returned by the function, the loop is stopped.
func (bs *brains) brains(fn func(b *brain) error) (err error) {
	bs.m.Lock()
	defer bs.m.Unlock()
	for _, b := range bs.n {
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
	delete(bs.k, b.key)
	delete(bs.n, b.name)
}

// set sets the brain in the pool.
func (bs *brains) set(b *brain) {
	bs.m.Lock()
	defer bs.m.Unlock()
	bs.k[b.key] = b
	bs.n[b.name] = b
}
