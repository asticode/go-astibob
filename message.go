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
	ListenablesRegisterMessage = "listenables.register"
	RunnableCrashedMessage     = "runnable.crashed"
	RunnableStartMessage       = "runnable.start"
	RunnableStartedMessage     = "runnable.started"
	RunnableStopMessage        = "runnable.stop"
	RunnableStoppedMessage     = "runnable.stopped"
	UIDisconnectedMessage      = "ui.disconnected"
	UIMessageNamesAddMessage    = "ui.message.names.add"
	UIMessageNamesDeleteMessage = "ui.message.names.delete"
	UIPingMessage              = "ui.ping"
	UIRegisterMessage          = "ui.register"
	UIWelcomeMessage           = "ui.welcome"
	WorkerDisconnectedMessage  = "worker.disconnected"
	WorkerRegisterMessage      = "worker.register"
	WorkerRegisteredMessage    = "worker.registered"
	WorkerWelcomeMessage       = "worker.welcome"
)

type MessageContent struct {
	Name    string      `json:"name"`
	Payload interface{} `json:"payload"`
}

func NewMessageFromContent(from Identifier, to *Identifier, c MessageContent) (m *Message, err error) {
	// Create message
	m = newMessage(from, to, c.Name)

	// Marshal payload
	if c.Payload != nil {
		if m.Payload, err = json.Marshal(c.Payload); err != nil {
			err = errors.Wrap(err, "astibob: marshaling payload failed")
			return
		}
	}
	return
}

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
	if !strOrMapMatch(astiptr.Str(i.Type), astiptr.Str(id.Type), i.Types, id.Types) {
		return false
	}

	// Check worker
	if i.Worker != nil && (id.Worker == nil || *i.Worker != *id.Worker) {
		return false
	}
	return true
}

func strOrMapMatch(srcStr, dstStr *string, srcMap, dstMap map[string]bool) bool {
	if srcMap != nil {
		if dstMap != nil {
			match := false
			for t := range dstMap {
				if _, ok := srcMap[t]; ok {
					match = true
					break
				}
			}
			if !match {
				return false
			}
		} else if dstStr != nil {
			if _, ok := srcMap[*dstStr]; !ok {
				return false
			}
		}
	} else if srcStr != nil {
		if dstMap != nil {
			if _, ok := dstMap[*srcStr]; !ok {
				return false
			}
		} else if dstStr != nil && *srcStr != *dstStr {
			return false
		}
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

type UI struct {
	MessageNames []string `json:"message_names"`
	Name         string   `json:"name"`
}

type WelcomeWorker struct {
	UIMessageNames []string `json:"ui_message_names,omitempty"`
	Workers        []Worker `json:"workers,omitempty"`
}

type Worker struct {
	Addr      string            `json:"addr,omitempty"`
	Name      string            `json:"name"`
	Runnables []RunnableMessage `json:"runnables,omitempty"`
}

type RunnableMessage struct {
	Metadata
	Status      string `json:"status"`
	WebHomepage string `json:"web_homepage,omitempty"`
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

func NewListenablesRegisterMessage(from Identifier, to *Identifier, l Listenables) (m *Message, err error) {
	// Create message
	m = newMessage(from, to, ListenablesRegisterMessage)

	// Marshal payload
	if m.Payload, err = json.Marshal(l); err != nil {
		err = errors.Wrap(err, "astibob: marshaling payload failed")
		return
	}
	return
}

func ParseListenablesRegisterPayload(m *Message) (l Listenables, err error) {
	if err = json.Unmarshal(m.Payload, &l); err != nil {
		err = errors.Wrap(err, "astibob: unmarshaling failed")
		return
	}
	return
}

func ParseRunnableStartPayload(m *Message) (name string, err error) {
	if err = json.Unmarshal(m.Payload, &name); err != nil {
		err = errors.Wrap(err, "astibob: unmarshaling failed")
		return
	}
	return
}

func ParseRunnableStopPayload(m *Message) (name string, err error) {
	if err = json.Unmarshal(m.Payload, &name); err != nil {
		err = errors.Wrap(err, "astibob: unmarshaling failed")
		return
	}
	return
}

func NewRunnableCrashedMessage(from Identifier, to *Identifier) *Message {
	return newMessage(from, to, RunnableCrashedMessage)
}

func NewRunnableStartedMessage(from Identifier, to *Identifier) *Message {
	return newMessage(from, to, RunnableStartedMessage)
}

func NewRunnableStoppedMessage(from Identifier, to *Identifier) *Message {
	return newMessage(from, to, RunnableStoppedMessage)
}

func NewUIDisconnectedMessage(from Identifier, to *Identifier, name string) (m *Message, err error) {
	// Create message
	m = newMessage(from, to, UIDisconnectedMessage)

	// Marshal payload
	if m.Payload, err = json.Marshal(name); err != nil {
		err = errors.Wrap(err, "astibob: marshaling payload failed")
		return
	}
	return
}

func ParseUIDisconnectedPayload(m *Message) (name string, err error) {
	if err = json.Unmarshal(m.Payload, &name); err != nil {
		err = errors.Wrap(err, "astibob: unmarshaling failed")
		return
	}
	return
}

func NewUIMessageNamesAddMessage(from Identifier, to *Identifier, names []string) (m *Message, err error) {
	// Create message
	m = newMessage(from, to, UIMessageNamesAddMessage)

	// Marshal payload
	if m.Payload, err = json.Marshal(names); err != nil {
		err = errors.Wrap(err, "astibob: marshaling payload failed")
		return
	}
	return
}

func ParseUIMessageNamesAddPayload(m *Message) (names []string, err error) {
	if err = json.Unmarshal(m.Payload, &names); err != nil {
		err = errors.Wrap(err, "astibob: unmarshaling failed")
		return
	}
	return
}

func NewUIMessageNamesDeleteMessage(from Identifier, to *Identifier, names []string) (m *Message, err error) {
	// Create message
	m = newMessage(from, to, UIMessageNamesDeleteMessage)

	// Marshal payload
	if m.Payload, err = json.Marshal(names); err != nil {
		err = errors.Wrap(err, "astibob: marshaling payload failed")
		return
	}
	return
}

func ParseUIMessageNamesDeletePayload(m *Message) (names []string, err error) {
	if err = json.Unmarshal(m.Payload, &names); err != nil {
		err = errors.Wrap(err, "astibob: unmarshaling failed")
		return
	}
	return
}

func ParseUIRegisterPayload(m *Message) (u UI, err error) {
	if err = json.Unmarshal(m.Payload, &u); err != nil {
		err = errors.Wrap(err, "astibob: unmarshaling failed")
		return
	}
	return
}

func NewUIWelcomeMessage(from Identifier, to *Identifier, w WelcomeUI) (m *Message, err error) {
	// Create message
	m = newMessage(from, to, UIWelcomeMessage)

	// Marshal payload
	if m.Payload, err = json.Marshal(w); err != nil {
		err = errors.Wrap(err, "astibob: marshaling payload failed")
		return
	}
	return
}

func NewWorkerDisconnectedMessage(from Identifier, to *Identifier, worker string) (m *Message, err error) {
	// Create message
	m = newMessage(from, to, WorkerDisconnectedMessage)

	// Marshal payload
	if m.Payload, err = json.Marshal(worker); err != nil {
		err = errors.Wrap(err, "astibob: marshaling payload failed")
		return
	}
	return
}

func ParseWorkerDisconnectedPayload(m *Message) (worker string, err error) {
	if err = json.Unmarshal(m.Payload, &worker); err != nil {
		err = errors.Wrap(err, "astibob: unmarshaling failed")
		return
	}
	return
}

func NewWorkerRegisterMessage(from Identifier, to *Identifier, w Worker) (m *Message, err error) {
	// Create message
	m = newMessage(from, to, WorkerRegisterMessage)

	// Marshal payload
	if m.Payload, err = json.Marshal(w); err != nil {
		err = errors.Wrap(err, "astibob: marshaling payload failed")
		return
	}
	return
}

func ParseWorkerRegisterPayload(m *Message) (w Worker, err error) {
	if err = json.Unmarshal(m.Payload, &w); err != nil {
		err = errors.Wrap(err, "astibob: unmarshaling failed")
		return
	}
	return
}

func NewWorkerRegisteredMessage(from Identifier, to *Identifier, w Worker) (m *Message, err error) {
	// Create message
	m = newMessage(from, to, WorkerRegisteredMessage)

	// Marshal payload
	if m.Payload, err = json.Marshal(w); err != nil {
		err = errors.Wrap(err, "astibob: marshaling payload failed")
		return
	}
	return
}

func ParseWorkerRegisteredPayload(m *Message) (w Worker, err error) {
	if err = json.Unmarshal(m.Payload, &w); err != nil {
		err = errors.Wrap(err, "astibob: unmarshaling failed")
		return
	}
	return
}

func NewWorkerWelcomeMessage(from Identifier, to *Identifier, w WelcomeWorker) (m *Message, err error) {
	// Create message
	m = newMessage(from, to, WorkerWelcomeMessage)

	// Marshal payload
	if m.Payload, err = json.Marshal(w); err != nil {
		err = errors.Wrap(err, "astibob: marshaling payload failed")
		return
	}
	return
}

func ParseWorkerWelcomePayload(m *Message) (w WelcomeWorker, err error) {
	if err = json.Unmarshal(m.Payload, &w); err != nil {
		err = errors.Wrap(err, "astibob: unmarshaling failed")
		return
	}
	return
}
