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
	From    Identifier        `json:"from"`
	Name    string            `json:"name"`
	Payload json.RawMessage   `json:"payload,omitempty"`
	State   map[string]string `json:"state,omitempty"`
}

type Identifier struct {
	Name *string `json:"name,omitempty"`
	Type string  `json:"type"`
}

func NewMessage() *Message {
	return &Message{State: make(map[string]string)}
}

func newMessage(from Identifier, name string) *Message {
	m := NewMessage()
	m.From = from
	m.Name = name
	return m
}

func NewCmdWorkerRegisterMessage(from Identifier, worker string) (m *Message, err error) {
	// Create message
	m = newMessage(from, CmdWorkerRegisterMessage)

	// Marshal payload
	if m.Payload, err = json.Marshal(worker); err != nil {
		err = errors.Wrap(err, "astibob: marshaling payload failed")
		return
	}
	return
}

func ParseWorkerRegisterCmdPayload(m *Message) (worker string, err error) {
	// Check name
	if m.Name != CmdWorkerRegisterMessage {
		err = fmt.Errorf("astibob: invalid name %s, requested %s", m.Name, CmdWorkerRegisterMessage)
		return
	}

	// Unmarshal
	if err = json.Unmarshal(m.Payload, &worker); err != nil {
		err = errors.Wrap(err, "astibob: unmarshaling failed")
		return
	}
	return
}

func NewEventWorkerDisconnectedMessage(from Identifier, worker string) (m *Message, err error) {
	// Create message
	m = newMessage(from, EventWorkerDisconnectedMessage)

	// Marshal payload
	if m.Payload, err = json.Marshal(worker); err != nil {
		err = errors.Wrap(err, "astibob: marshaling payload failed")
		return
	}
	return
}

func ParseWorkerDisconnectedEventPayload(m *Message) (worker string, err error) {
	// Check name
	if m.Name != EventWorkerDisconnectedMessage {
		err = fmt.Errorf("astibob: invalid name %s, requested %s", m.Name, EventWorkerDisconnectedMessage)
		return
	}

	// Unmarshal
	if err = json.Unmarshal(m.Payload, &worker); err != nil {
		err = errors.Wrap(err, "astibob: unmarshaling failed")
		return
	}
	return
}

func NewEventWorkerWelcomeMessage(from Identifier) (m *Message, err error) {
	// Create message
	m = newMessage(from, EventWorkerWelcomeMessage)
	return
}

func ParseWorkerWelcomeEventPayload(m *Message) (err error) {
	// Check name
	if m.Name != EventWorkerWelcomeMessage {
		err = fmt.Errorf("astibob: invalid name %s, requested %s", m.Name, EventWorkerWelcomeMessage)
		return
	}
	return
}
