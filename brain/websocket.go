package astibrain

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/asticode/go-astilog"
	"github.com/asticode/go-astiws"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

// Websocket event names
const (
	WebsocketEventNameAbilityCrashed = "ability.crashed"
	WebsocketEventNameAbilityStart   = "ability.start"
	WebsocketEventNameAbilityStarted = "ability.started"
	WebsocketEventNameAbilityStop    = "ability.stop"
	WebsocketEventNameAbilityStopped = "ability.stopped"
	WebsocketEventNameRegister       = "register"
	WebsocketEventNameRegistered     = "registered"
)

// webSocket represents a websocket wrapper
type webSocket struct {
	abilities   *abilities
	c           *astiws.Client
	isConnected bool
	h           http.Header
	m           sync.Mutex
	o           WebSocketOptions
	q           []astiws.BodyMessage
}

// WebSocketOptions are websocket options
type WebSocketOptions struct {
	Password string `toml:"password"`
	URL      string `toml:"url"`
	Username string `toml:"username"`
}

// newWebSocket creates a new websocket wrapper
func newWebSocket(abilities *abilities, o WebSocketOptions) (ws *webSocket) {
	// Create websocket
	ws = &webSocket{
		abilities: abilities,
		c:         astiws.NewClient(4096),
		h:         make(http.Header),
		o:         o,
	}

	// Set headers
	if len(o.Username) > 0 && len(o.Password) > 0 {
		ws.h.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(o.Username+":"+o.Password)))
	}

	// Add listeners
	ws.c.AddListener(WebsocketEventNameAbilityStart, ws.handleAbilityToggle)
	ws.c.AddListener(WebsocketEventNameAbilityStop, ws.handleAbilityToggle)
	ws.c.AddListener(WebsocketEventNameRegistered, ws.handleRegistered)
	return
}

// Close implements the io.Closer interface
func (ws *webSocket) Close() (err error) {
	// Close client
	astilog.Debug("astibrain: closing websocket client")
	if err = ws.c.Close(); err != nil {
		err = errors.Wrap(err, "astibrain: closing websocket client failed")
		return
	}
	return
}

// dial dials the websocket
func (ws *webSocket) dial(ctx context.Context, name string) {
	// Infinite loop to handle reconnect
	const sleepError = 5 * time.Second
	for {
		// Check context error
		if ctx.Err() != nil {
			return
		}

		// Dial
		if err := ws.c.DialWithHeaders(ws.o.URL, ws.h); err != nil {
			astilog.Error(errors.Wrap(err, "astibrain: dialing websocket failed"))
			time.Sleep(sleepError)
			continue
		}

		// Register
		if err := ws.sendRegister(name); err != nil {
			astilog.Error(errors.Wrap(err, "astibrain: sending register websocket event failed"))
			time.Sleep(sleepError)
			continue
		}

		// Read
		if err := ws.c.Read(); err != nil {
			ws.m.Lock()
			ws.isConnected = false
			ws.m.Unlock()
			if v, ok := errors.Cause(err).(*websocket.CloseError); ok && v.Code == websocket.CloseNormalClosure {
				astilog.Info("astibrain: brain has disconnected from bob")
			} else {
				astilog.Error(errors.Wrap(err, "astibrain: reading websocket failed"))
			}
			time.Sleep(sleepError)
			continue
		}
	}
}

// APIRegister is a register API payload
type APIRegister struct {
	Abilities map[string]APIAbility `json:"abilities"`
	Name      string                `json:"name"`
}

// APIAbility is an ability API payload
type APIAbility struct {
	IsOn bool   `json:"is_on"`
	Name string `json:"name"`
}

// sendRegister sends a register event
func (ws *webSocket) sendRegister(name string) (err error) {
	// Create payload
	p := APIRegister{
		Abilities: make(map[string]APIAbility),
		Name:      name,
	}

	// Loop through abilities
	ws.abilities.abilities(func(a *ability) error {
		p.Abilities[a.name] = APIAbility{
			IsOn: a.isOn(),
			Name: a.name,
		}
		return nil
	})

	// Write
	if err = ws.c.Write(WebsocketEventNameRegister, p); err != nil {
		err = errors.Wrapf(err, "astibrain: sending register event with payload %#v failed", p)
		return
	}
	return
}

// processQueue processes the queue
func (ws *webSocket) processQueue() {
	// Lock
	ws.m.Lock()
	defer ws.m.Unlock()

	// Nothing to do
	if len(ws.q) == 0 {
		return
	}

	// Log
	astilog.Debugf("astibrain: processing %d queued websocket messages", len(ws.q))

	// Loop through queue
	for _, m := range ws.q {
		ws.write(m.EventName, m.Payload)
	}

	// Reset queue
	ws.q = []astiws.BodyMessage{}
}

// send sends an event and mutes the error (which is still logged)
func (ws *webSocket) send(eventName string, payload interface{}) {
	// Retrieve connected status
	ws.m.Lock()
	isConnected := ws.isConnected
	ws.m.Unlock()

	// Websocket is not connected, add message to queue
	if !isConnected {
		ws.m.Lock()
		ws.q = append(ws.q, astiws.BodyMessage{EventName: eventName, Payload: payload})
		ws.m.Unlock()
		return
	}

	// Write
	ws.write(eventName, payload)
}

// write writes an event and mutes the error (which is still logged)
func (ws *webSocket) write(eventName string, payload interface{}) {
	if err := ws.c.Write(eventName, payload); err != nil {
		astilog.Error(errors.Wrapf(err, "astibrain: sending %s websocket event with payload %#v failed", eventName, payload))
	}
}

// handleRegistered handles the registered websocket event
func (ws *webSocket) handleRegistered(c *astiws.Client, eventName string, payload json.RawMessage) error {
	// Process queued message
	ws.processQueue()

	// Update connected attribute
	ws.m.Lock()
	ws.isConnected = true
	ws.m.Unlock()

	// Log
	astilog.Info("astibrain: brain has connected to bob")
	return nil
}

// handleAbilityToggle handles the ability toggle websocket events
func (ws *webSocket) handleAbilityToggle(c *astiws.Client, eventName string, payload json.RawMessage) error {
	// Decode payload
	var name string
	if err := json.Unmarshal(payload, &name); err != nil {
		astilog.Error(errors.Wrapf(err, "astibrain: json unmarshaling %s payload %#v failed", eventName, payload))
		return nil
	}

	// Retrieve ability
	a, ok := ws.abilities.ability(name)
	if !ok {
		astilog.Error(fmt.Errorf("astibrain: unknown ability %s", name))
		return nil
	}

	// Either start or stop the ability
	if eventName == WebsocketEventNameAbilityStart {
		a.on()
	} else {
		a.off()
	}
	return nil
}
