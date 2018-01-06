package astibob

import "sync"

// ability represents an ability as Bob knows it
type ability struct {
	isOn bool
	m    sync.Mutex // Locks attributes
	name string
}

// newAbility creates a new ability
func newAbility(name string, isOn bool) *ability {
	return &ability{
		isOn: isOn,
		name: name,
	}
}
