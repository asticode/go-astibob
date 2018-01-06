package astibrain

import "sync"

// abilities is a pool of abilities
type abilities struct {
	a map[string]*ability
	m sync.Mutex // Locks a
}

// newAbilities creates a new pool of abilities
func newAbilities() *abilities {
	return &abilities{a: make(map[string]*ability)}
}

// ability returns a specific ability based on its name.
func (as *abilities) ability(name string) (a *ability, ok bool) {
	as.m.Lock()
	defer as.m.Unlock()
	a, ok = as.a[name]
	return
}

// abilities loops through abilities and execute a function on each of them.
// If an error is returned by the function, the loop is stopped.
func (as *abilities) abilities(fn func(a *ability) error) (err error) {
	as.m.Lock()
	defer as.m.Unlock()
	for _, a := range as.a {
		if err = fn(a); err != nil {
			return
		}
	}
	return
}

// set sets a new ability in the pool.
func (as *abilities) set(a *ability) {
	as.m.Lock()
	defer as.m.Unlock()
	as.a[a.name] = a
	return
}
