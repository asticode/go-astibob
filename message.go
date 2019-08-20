package astibob

import (
	"encoding/json"

	"fmt"

	"sync"

	"github.com/pkg/errors"
)

// Identifier types
const (
	IndexIdentifierType  = "index"
	WorkerIdentifierType = "worker"
)

// Message names
const (
	WorkerRegisterCmdMessage       = "cmd.worker.register"
	WorkerDisconnectedEventMessage = "event.worker.disconnected"
	WorkerWelcomeEventMessage      = "event.worker.welcome"
)

type Message struct {
	ctx     map[string]interface{} `json:"-"`
	From    Identifier             `json:"from"`
	m       *sync.Mutex            `json:"-"` // Lock ctx
	Name    string                 `json:"name"`
	Payload json.RawMessage        `json:"payload"`
}

type Identifier struct {
	Name *string `json:"name"`
	Type string  `json:"type"`
}

func NewMessage() *Message {
	return &Message{
		ctx: make(map[string]interface{}),
		m:   &sync.Mutex{},
	}
}

func newMessage(from Identifier, name string) *Message {
	m := NewMessage()
	m.From = from
	m.Name = name
	return m
}

func (m *Message) ToContext(v string, k interface{}) {
	m.m.Lock()
	defer m.m.Unlock()
	m.ctx[v] = k
}

func (m *Message) FromContext(v string) (k interface{}, ok bool) {
	m.m.Lock()
	defer m.m.Unlock()
	k, ok = m.ctx[v]
	return
}

func NewWorkerRegisterCmdMessage(from Identifier, worker string) (m *Message, err error) {
	// Create message
	m = newMessage(from, WorkerRegisterCmdMessage)

	// Marshal payload
	if m.Payload, err = json.Marshal(worker); err != nil {
		err = errors.Wrap(err, "astibob: marshaling payload failed")
		return
	}
	return
}

func ParseWorkerRegisterCmdPayload(m *Message) (worker string, err error) {
	// Check name
	if m.Name != WorkerRegisterCmdMessage {
		err = fmt.Errorf("astibob: invalid name %s, requested %s", m.Name, WorkerRegisterCmdMessage)
		return
	}

	// Unmarshal
	if err = json.Unmarshal(m.Payload, &worker); err != nil {
		err = errors.Wrap(err, "astibob: unmarshaling failed")
		return
	}
	return
}

func NewWorkerDisconnectedEventMessage(from Identifier, worker string) (m *Message, err error) {
	// Create message
	m = newMessage(from, WorkerDisconnectedEventMessage)

	// Marshal payload
	if m.Payload, err = json.Marshal(worker); err != nil {
		err = errors.Wrap(err, "astibob: marshaling payload failed")
		return
	}
	return
}

func ParseWorkerDisconnectedEventPayload(m *Message) (worker string, err error) {
	// Check name
	if m.Name != WorkerDisconnectedEventMessage {
		err = fmt.Errorf("astibob: invalid name %s, requested %s", m.Name, WorkerDisconnectedEventMessage)
		return
	}

	// Unmarshal
	if err = json.Unmarshal(m.Payload, &worker); err != nil {
		err = errors.Wrap(err, "astibob: unmarshaling failed")
		return
	}
	return
}

func NewWorkerWelcomeEventMessage(from Identifier) (m *Message, err error) {
	// Create message
	m = newMessage(from, WorkerWelcomeEventMessage)
	return
}

func ParseWorkerWelcomeEventPayload(m *Message) (err error) {
	// Check name
	if m.Name != WorkerWelcomeEventMessage {
		err = fmt.Errorf("astibob: invalid name %s, requested %s", m.Name, WorkerWelcomeEventMessage)
		return
	}
	return
}
