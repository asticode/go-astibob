package astibob

import (
	"encoding/json"

	astiptr "github.com/asticode/go-astitools/ptr"
	"github.com/pkg/errors"
)

// Identifier types
const (
	IndexIdentifierType    = "index"
	RunnableIdentifierType = "runnable"
	UIIdentifierType       = "ui"
	WorkerIdentifierType   = "worker"
)

// Message names
const (
	CmdListenablesRegisterMessage  = "cmd.listenables.register"
	CmdRunnableStartMessage        = "cmd.runnable.start"
	CmdRunnableStopMessage         = "cmd.runnable.stop"
	CmdUIPingMessage               = "cmd.ui.ping"
	CmdWorkerRegisterMessage       = "cmd.worker.register"
	EventRunnableCrashedMessage    = "event.runnable.crashed"
	EventRunnableStartedMessage    = "event.runnable.started"
	EventRunnableStoppedMessage    = "event.runnable.stopped"
	EventUIDisconnectedMessage     = "event.ui.disconnected"
	EventUIWelcomeMessage          = "event.ui.welcome"
	EventWorkerDisconnectedMessage = "event.worker.disconnected"
	EventWorkerRegisteredMessage   = "event.worker.registered"
	EventWorkerWelcomeMessage      = "event.worker.welcome"
)

type Message struct {
	From    Identifier      `json:"from"`
	Name    string          `json:"name"`
	Payload json.RawMessage `json:"payload,omitempty"`
	To      *Identifier     `json:"to,omitempty"`
}

func (m *Message) Clone() (o *Message) {
	// Create message
	o = &Message{
		From: *m.From.Clone(),
		Name: m.Name,
	}

	// Clone to
	if m.To != nil {
		o.To = m.To.Clone()
	}

	// Clone payload
	if len(m.Payload) > 0 {
		o.Payload = make(json.RawMessage, len(m.Payload))
		copy(o.Payload, m.Payload)
	}
	return
}

type Identifier struct {
	Name   *string         `json:"name,omitempty"`
	Type   string          `json:"type,omitempty"`
	Types  map[string]bool `json:"types,omitempty"`
	Worker *string         `json:"worker,omitempty"`
}

func NewIndexIdentifier() *Identifier {
	return &Identifier{Type: IndexIdentifierType}
}

func NewUIIdentifier(name string) *Identifier {
	return &Identifier{
		Name: astiptr.Str(name),
		Type: UIIdentifierType,
	}
}

func NewRunnableIdentifier(runnable, worker string) *Identifier {
	return &Identifier{
		Name:   astiptr.Str(runnable),
		Type:   RunnableIdentifierType,
		Worker: astiptr.Str(worker),
	}
}

func NewWorkerIdentifier(name string) *Identifier {
	return &Identifier{
		Name: astiptr.Str(name),
		Type: WorkerIdentifierType,
	}
}

func (i Identifier) Clone() (o *Identifier) {
	// Create identifier
	o = &Identifier{Type: i.Type}

	// Add name
	if i.Name != nil {
		o.Name = astiptr.Str(*i.Name)
	}

	// Add types
	if len(i.Types) > 0 {
		o.Types = make(map[string]bool)
		for k, v := range i.Types {
			o.Types[k] = v
		}
	}

	// Add worker
	if i.Worker != nil {
		o.Worker = astiptr.Str(*i.Worker)
	}
	return
}

func (i *Identifier) match(id Identifier) bool {
	// Check type
	if i.Types != nil {
		if id.Types != nil {
			match := false
			for t := range id.Types {
				if _, ok := i.Types[t]; ok {
					match = true
					break
				}
			}
			if !match {
				return false
			}
		} else {
			if _, ok := i.Types[id.Type]; !ok {
				return false
			}
		}
	} else {
		if id.Types != nil {
			if _, ok := id.Types[i.Type]; !ok {
				return false
			}
		} else if i.Type != id.Type {
			return false
		}
	}

	// Check name
	if i.Name != nil && (id.Name == nil || *i.Name != *id.Name) {
		return false
	}

	// Check worker
	if i.Worker != nil && (id.Worker == nil || *i.Worker != *id.Worker) {
		return false
	}
	return true
}

func (i Identifier) WorkerName() string {
	switch i.Type {
	case RunnableIdentifierType:
		if i.Worker != nil {
			return *i.Worker
		}
	case WorkerIdentifierType:
		if i.Name != nil {
			return *i.Name
		}
	}
	return ""
}

type WelcomeUI struct {
	Name    string   `json:"name"`
	Workers []Worker `json:"workers,omitempty"`
}

type Worker struct {
	Addr      string            `json:"addr,omitempty"`
	Name      string            `json:"name"`
	Runnables []RunnableMessage `json:"runnables,omitempty"`
}

type RunnableMessage struct {
	Metadata
	Status     string `json:"status"`
	UIHomepage string `json:"ui_homepage,omitempty"`
}

type Metadata struct {
	Description string `json:"description"`
	Name        string `json:"name"`
}

type Error struct {
	Message string `json:"message"`
}

type Listenables struct {
	Names    []string `json:"names"`
	Runnable string   `json:"runnable"`
}

func NewMessage() *Message {
	return &Message{}
}

func newMessage(from Identifier, to *Identifier, name string) *Message {
	m := NewMessage()
	m.From = from
	m.Name = name
	m.To = to
	return m
}

func NewCmdListenablesRegisterMessage(from Identifier, to *Identifier, l Listenables) (m *Message, err error) {
	// Create message
	m = newMessage(from, to, CmdListenablesRegisterMessage)

	// Marshal payload
	if m.Payload, err = json.Marshal(l); err != nil {
		err = errors.Wrap(err, "astibob: marshaling payload failed")
		return
	}
	return
}

func ParseCmdListenablesRegisterPayload(m *Message) (l Listenables, err error) {
	if err = json.Unmarshal(m.Payload, &l); err != nil {
		err = errors.Wrap(err, "astibob: unmarshaling failed")
		return
	}
	return
}

func ParseCmdRunnableStartPayload(m *Message) (name string, err error) {
	if err = json.Unmarshal(m.Payload, &name); err != nil {
		err = errors.Wrap(err, "astibob: unmarshaling failed")
		return
	}
	return
}

func ParseCmdRunnableStopPayload(m *Message) (name string, err error) {
	if err = json.Unmarshal(m.Payload, &name); err != nil {
		err = errors.Wrap(err, "astibob: unmarshaling failed")
		return
	}
	return
}

func NewCmdWorkerRegisterMessage(from Identifier, to *Identifier, w Worker) (m *Message, err error) {
	// Create message
	m = newMessage(from, to, CmdWorkerRegisterMessage)

	// Marshal payload
	if m.Payload, err = json.Marshal(w); err != nil {
		err = errors.Wrap(err, "astibob: marshaling payload failed")
		return
	}
	return
}

func ParseCmdWorkerRegisterPayload(m *Message) (w Worker, err error) {
	if err = json.Unmarshal(m.Payload, &w); err != nil {
		err = errors.Wrap(err, "astibob: unmarshaling failed")
		return
	}
	return
}

func NewEventRunnableCrashedMessage(from Identifier, to *Identifier) *Message {
	return newMessage(from, to, EventRunnableCrashedMessage)
}

func NewEventRunnableStartedMessage(from Identifier, to *Identifier) *Message {
	return newMessage(from, to, EventRunnableStartedMessage)
}

func NewEventRunnableStoppedMessage(from Identifier, to *Identifier) *Message {
	return newMessage(from, to, EventRunnableStoppedMessage)
}

func NewEventUIDisconnectedMessage(from Identifier, to *Identifier, name string) (m *Message, err error) {
	// Create message
	m = newMessage(from, to, EventUIDisconnectedMessage)

	// Marshal payload
	if m.Payload, err = json.Marshal(name); err != nil {
		err = errors.Wrap(err, "astibob: marshaling payload failed")
		return
	}
	return
}

func NewEventUIWelcomeMessage(from Identifier, to *Identifier, w WelcomeUI) (m *Message, err error) {
	// Create message
	m = newMessage(from, to, EventUIWelcomeMessage)

	// Marshal payload
	if m.Payload, err = json.Marshal(w); err != nil {
		err = errors.Wrap(err, "astibob: marshaling payload failed")
		return
	}
	return
}

func ParseEventUIDisconnectedPayload(m *Message) (name string, err error) {
	if err = json.Unmarshal(m.Payload, &name); err != nil {
		err = errors.Wrap(err, "astibob: unmarshaling failed")
		return
	}
	return
}

func NewEventWorkerDisconnectedMessage(from Identifier, to *Identifier, worker string) (m *Message, err error) {
	// Create message
	m = newMessage(from, to, EventWorkerDisconnectedMessage)

	// Marshal payload
	if m.Payload, err = json.Marshal(worker); err != nil {
		err = errors.Wrap(err, "astibob: marshaling payload failed")
		return
	}
	return
}

func ParseEventWorkerDisconnectedPayload(m *Message) (worker string, err error) {
	if err = json.Unmarshal(m.Payload, &worker); err != nil {
		err = errors.Wrap(err, "astibob: unmarshaling failed")
		return
	}
	return
}

func NewEventWorkerRegisteredMessage(from Identifier, to *Identifier, w Worker) (m *Message, err error) {
	// Create message
	m = newMessage(from, to, EventWorkerRegisteredMessage)

	// Marshal payload
	if m.Payload, err = json.Marshal(w); err != nil {
		err = errors.Wrap(err, "astibob: marshaling payload failed")
		return
	}
	return
}

func ParseEventWorkerRegisteredPayload(m *Message) (w Worker, err error) {
	if err = json.Unmarshal(m.Payload, &w); err != nil {
		err = errors.Wrap(err, "astibob: unmarshaling failed")
		return
	}
	return
}

func NewEventWorkerWelcomeMessage(from Identifier, to *Identifier, ws []Worker) (m *Message, err error) {
	// Create message
	m = newMessage(from, to, EventWorkerWelcomeMessage)

	// Marshal payload
	if m.Payload, err = json.Marshal(ws); err != nil {
		err = errors.Wrap(err, "astibob: marshaling payload failed")
		return
	}
	return
}

func ParseEventWorkerWelcomePayload(m *Message) (ws []Worker, err error) {
	if err = json.Unmarshal(m.Payload, &ws); err != nil {
		err = errors.Wrap(err, "astibob: unmarshaling failed")
		return
	}
	return
}
