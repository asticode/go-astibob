package astibob

import (
	"sync"

	"github.com/asticode/go-astiws"
)

// brain is a brain as Bob knows it
type brain struct {
	k    map[string]*ability // Indexed by key
	key  string
	m    sync.Mutex          // Locks a
	n    map[string]*ability // Indexed by name
	name string
	ws   *astiws.Client
}

// newBrain creates a new brain
func newBrain(name string, ws *astiws.Client) *brain {
	return &brain{
		k:    make(map[string]*ability),
		key:  key(name),
		n:    make(map[string]*ability),
		name: name,
		ws:   ws,
	}
}

// ability returns a specific ability based on its name.
func (b *brain) ability(name string) (a *ability, ok bool) {
	b.m.Lock()
	defer b.m.Unlock()
	a, ok = b.n[name]
	return
}

// abilityByKey returns a specific ability based on its key.
func (b *brain) abilityByKey(key string) (a *ability, ok bool) {
	b.m.Lock()
	defer b.m.Unlock()
	a, ok = b.k[key]
	return
}

// abilities loops through abilities and execute a function on each of them.
// If an error is returned by the function, the loop is stopped.
func (b *brain) abilities(fn func(a *ability) error) (err error) {
	b.m.Lock()
	defer b.m.Unlock()
	for _, a := range b.n {
		if err = fn(a); err != nil {
			return
		}
	}
	return
}

// set sets the ability in the pool.
func (b *brain) set(a *ability) {
	b.m.Lock()
	defer b.m.Unlock()
	b.k[a.key] = a
	b.n[a.name] = a
}
