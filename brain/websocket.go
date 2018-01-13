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
	gorilla "github.com/gorilla/websocket"
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

// websocket represents a websocket wrapper
type websocket struct {
	abilities   *abilities
	c           *astiws.Client
	isConnected bool
	h           http.Header
	m           sync.Mutex
	o           WebsocketOptions
	q           []astiws.BodyMessage
}

// WebsocketOptions are websocket options
type WebsocketOptions struct {
	MaxMessageSize int    `toml:"max_message_size"`
	Password       string `toml:"password"`
	URL            string `toml:"url"`
	Username       string `toml:"username"`
}

// newWebsocket creates a new websocket wrapper
func newWebsocket(abilities *abilities, o WebsocketOptions) (ws *websocket) {
	// Create websocket
	ws = &websocket{
		abilities: abilities,
		c:         astiws.NewClient(o.MaxMessageSize),
		h:         make(http.Header),
		o:         o,
	}

	// Set headers
	if len(o.Username) > 0 && len(o.Password) > 0 {
		ws.h.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(o.Username+":"+o.Password)))
	}

	// Add default listeners
	ws.c.AddListener(WebsocketEventNameAbilityStart, ws.handleAbilityToggle)
	ws.c.AddListener(WebsocketEventNameAbilityStop, ws.handleAbilityToggle)
	ws.c.AddListener(WebsocketEventNameRegistered, ws.handleRegistered)
	return
}

// WebsocketAbilityEventName returns the websocket ability event name
func WebsocketAbilityEventName(abilityName, eventName string) string {
	return fmt.Sprintf("ability.%s.%s", abilityName, eventName)
}

// Close implements the io.Closer interface
func (ws *websocket) Close() (err error) {
	// Close client
	astilog.Debug("astibrain: closing websocket client")
	if err = ws.c.Close(); err != nil {
		err = errors.Wrap(err, "astibrain: closing websocket client failed")
		return
	}
	return
}

// dial dials the websocket
func (ws *websocket) dial(ctx context.Context, name string) {
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
			if v, ok := errors.Cause(err).(*gorilla.CloseError); ok && v.Code == gorilla.CloseNormalClosure {
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
	Abilities   map[string]APIAbility `json:"abilities"`
	Name        string                `json:"name"`
}

// APIAbility is an ability API payload
type APIAbility struct {
	IsOn bool   `json:"is_on"`
	Description string                `json:"description"`
	Name string `json:"name"`
}

// sendRegister sends a register event
func (ws *websocket) sendRegister(name string) (err error) {
	// Create payload
	p := APIRegister{
		Abilities:   make(map[string]APIAbility),
		Name:        name,
	}

	// Loop through abilities
	ws.abilities.abilities(func(a *ability) error {
		p.Abilities[a.name] = APIAbility{
			Description: a.description,
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
func (ws *websocket) processQueue() {
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
func (ws *websocket) send(eventName string, payload interface{}) {
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
func (ws *websocket) write(eventName string, payload interface{}) {
	if err := ws.c.Write(eventName, payload); err != nil {
		astilog.Error(errors.Wrapf(err, "astibrain: sending %s websocket event with payload %#v failed", eventName, payload))
	}
}

// handleRegistered handles the registered websocket event
func (ws *websocket) handleRegistered(c *astiws.Client, eventName string, payload json.RawMessage) error {
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
func (ws *websocket) handleAbilityToggle(c *astiws.Client, eventName string, payload json.RawMessage) error {
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
