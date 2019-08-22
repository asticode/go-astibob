package astibob

import (
	"encoding/json"

	"fmt"

	"github.com/pkg/errors"
)

// Identifier types
const (
	IndexIdentifierType  = "index"
	UIIdentifierType     = "ui"
	WorkerIdentifierType = "worker"
)

// Message names
const (
	CmdUIPingMessage               = "cmd.ui.ping"
	CmdWorkerRegisterMessage       = "cmd.worker.register"
	EventUIDisconnectedMessage     = "event.ui.disconnected"
	EventUIWelcomeMessage          = "event.ui.welcome"
	EventWorkerDisconnectedMessage = "event.worker.disconnected"
	EventWorkerWelcomeMessage      = "event.worker.welcome"
)

type Message struct {
	From    Identifier      `json:"from"`
	Name    string          `json:"name"`
	Payload json.RawMessage `json:"payload,omitempty"`
	To      *Identifier     `json:"to,omitempty"`
}

type Identifier struct {
	Name *string `json:"name,omitempty"`
	Type string  `json:"type"`
}

type WelcomeUI struct {
	Name    string   `json:"name"`
	Workers []Worker `json:"workers,omitempty"`
}

type Worker struct {
	Abilities []Ability `json:"abilities,omitempty"`
	Name      string    `json:"name"`
}

type Ability struct {
	Description string `json:"description"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	UIHomepage string   `json:"ui_homepage"`
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

func NewCmdWorkerRegisterMessage(from Identifier, to *Identifier) *Message {
	return newMessage(from, to, CmdWorkerRegisterMessage)
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

func NewEventUIWelcomeMessage(from Identifier, to *Identifier, name string, ws []Worker) (m *Message, err error) {
	// Create message
	m = newMessage(from, to, EventUIWelcomeMessage)

	// Marshal payload
	if m.Payload, err = json.Marshal(WelcomeUI{
		Name:    name,
		Workers: ws,
	}); err != nil {
		err = errors.Wrap(err, "astibob: marshaling payload failed")
		return
	}
	return
}

func ParseEventUIDisconnectedPayload(m *Message) (name string, err error) {
	// Check name
	if m.Name != EventUIDisconnectedMessage {
		err = fmt.Errorf("astibob: invalid name %s, requested %s", m.Name, EventUIDisconnectedMessage)
		return
	}

	// Unmarshal
	if err = json.Unmarshal(m.Payload, &name); err != nil {
		err = errors.Wrap(err, "astibob: unmarshaling failed")
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
	// Check name
	if m.Name != EventWorkerDisconnectedMessage {
		err = fmt.Errorf("astibob: invalid name %s, requested %s", m.Name, EventWorkerDisconnectedMessage)
		return
	}

	// Unmarshal
	if err = json.Unmarshal(m.Payload, &worker); err != nil {
		err = errors.Wrap(err, "astibob: unmarshaling failed")
	}
	return
}

func NewEventWorkerWelcomeMessage(from Identifier, to *Identifier) (m *Message, err error) {
	// Create message
	m = newMessage(from, to, EventWorkerWelcomeMessage)
	return
}

func ParseEventWorkerWelcomePayload(m *Message) (err error) {
	// Check name
	if m.Name != EventWorkerWelcomeMessage {
		err = fmt.Errorf("astibob: invalid name %s, requested %s", m.Name, EventWorkerWelcomeMessage)
		return
	}
	return
}
