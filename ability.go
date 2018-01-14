package astibob

import (
	"sync"

	"net/http"
)

// ability represents an ability
type ability struct {
	apiHandlers map[string]http.Handler
	description string
	key         string
	o           bool
	m           sync.Mutex
	name        string
	webHomepage string
}

// newAbility creates a new ability
func newAbility(name, description string, isOn bool) *ability {
	return &ability{
		apiHandlers: make(map[string]http.Handler),
		description: description,
		key:         key(name),
		o:           isOn,
		name:        name,
	}
}

// apiHandler returns the API handler based on its path
func (a *ability) apiHandler(path string) (h http.Handler, ok bool) {
	a.m.Lock()
	defer a.m.Unlock()
	h, ok = a.apiHandlers[path]
	return
}

// isOn returns whether the ability is on
func (a *ability) isOn() bool {
	a.m.Lock()
	defer a.m.Unlock()
	return a.o
}

// setOn sets whether the ability is on
func (a *ability) setOn(on bool) {
	a.m.Lock()
	defer a.m.Unlock()
	a.o = on
}
