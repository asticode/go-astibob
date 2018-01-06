package astibrain

import "sync"

// Event represents an event
type Event struct {
	AbilityName string
	Name        string
	Payload     interface{}
}

// dispatcher is an object capable of dispatching events from an ability to Bob
type dispatcher struct {
	chEvent chan Event
	chQuit  chan bool
	o       sync.Once
	ws      *websocket
}

// newDispatcher creates a new dispatcher
func newDispatcher(ws *websocket) *dispatcher {
	return &dispatcher{
		chEvent: make(chan Event),
		chQuit:  make(chan bool),
		ws:      ws,
	}
}

// Close implements the io.Closer interface
func (d *dispatcher) Close() error {
	d.o.Do(func() {
		close(d.chQuit)
	})
	return nil
}

// read reads events coming from abilities and dispatch them to Bob
func (d *dispatcher) read() {
	for {
		select {
		case e := <-d.chEvent:
			// TODO improve since if the send blocks, it blocks all events
			d.ws.send(WebsocketAbilityEventName(e.AbilityName, e.Name), e.Payload)
		case <-d.chQuit:
			return
		}
	}
}
