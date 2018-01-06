package astibob

import "sync"

// Listener represents a listener executed when an event is dispatched
type Listener func(e Event) (deleteListener bool)

// dispatcher represents an object capable of dispatching events
type dispatcher struct {
	id int
	f  map[string]map[int]Listener
	m  sync.Mutex
}

// newDispatcher creates a new dispatcher
func newDispatcher() *dispatcher {
	return &dispatcher{f: make(map[string]map[int]Listener)}
}

// addListener adds a listener
func (d *dispatcher) addListener(eventName string, l Listener) {
	d.m.Lock()
	defer d.m.Unlock()
	if _, ok := d.f[eventName]; !ok {
		d.f[eventName] = make(map[int]Listener)
	}
	d.id++
	d.f[eventName][d.id] = l
}

// delListener deletes a listener
func (d *dispatcher) delListener(eventName string, id int) {
	d.m.Lock()
	defer d.m.Unlock()
	if _, ok := d.f[eventName]; !ok {
		return
	}
	delete(d.f[eventName], id)
}

// Dispatch dispatches an event
func (d *dispatcher) dispatch(e Event) {
	// needed so dispatches of events triggered in the listeners can be received without blocking
	go func() {
		for id, l := range d.listeners(e.Name) {
			if l(e) {
				d.delListener(e.Name, id)
			}
		}
	}()
}

// listeners returns the listeners for an event name
func (d *dispatcher) listeners(eventName string) (ls map[int]Listener) {
	d.m.Lock()
	defer d.m.Unlock()
	ls = map[int]Listener{}
	if _, ok := d.f[eventName]; !ok {
		return
	}
	for k, v := range d.f[eventName] {
		ls[k] = v
	}
	return
}
