package astikeyboarding

// Constants
const (
	name = "Keyboarding"
)

// Actions
const (
	actionPress       = "press"
	actionType        = "type"
)

// Websocket event names
const (
	websocketEventNameAction = "action"
)

// PayloadAction represents an action payload
type PayloadAction struct {
	Action string   `json:"action"`
	Keys   []string `json:"keys,omitempty"`
	String string   `json:"string,omitempty"`
}

// Keyboarder represents an object capable of interacting with a keyboard
type Keyboarder interface {
	Press(keys ...string)
	Type(s string)
}
