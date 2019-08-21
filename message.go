package astibob

import (
	"encoding/json"

	"fmt"

	"github.com/pkg/errors"
)

// Identifier types
const (
	IndexIdentifierType  = "index"
	WorkerIdentifierType = "worker"
)

// Message names
const (
	CmdWorkerRegisterMessage       = "cmd.worker.register"
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
