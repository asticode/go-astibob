package astibob

import "sync"

// ability represents an ability
type ability struct {
	o    bool
	m    sync.Mutex
	name string
	ui   *UI
}

// UI represents the UI info
type UI struct {
	Description  string
	Homepage     string
	Title        string
	WebTemplates map[string]string
}

// newAbility creates a new ability
func newAbility(name string, isOn bool) *ability {
	return &ability{
		o:    isOn,
		name: name,
	}
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
