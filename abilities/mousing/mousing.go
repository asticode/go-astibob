package astimousing

// Constants
const (
	name = "Mousing"
)

// Actions
const (
	actionClickLeft   = "click.left"
	actionClickMiddle = "click.middle"
	actionClickRight  = "click.right"
	actionMove        = "move"
	actionScrollDown  = "scroll.down"
	actionScrollUp    = "scroll.up"
)

// Websocket event names
const (
	websocketEventNameAction = "action"
)

// PayloadAction represents an action payload
type PayloadAction struct {
	Action string `json:"action"`
	Double bool   `json:"double,omitempty"`
	X      int    `json:"x,omitempty"`
	Y      int    `json:"y,omitempty"`
}

// Mouser represents an object capable of interacting with a mouse
type Mouser interface {
	ClickLeft(double bool)
	ClickMiddle(double bool)
	ClickRight(double bool)
	Move(x, y int)
	ScrollDown(x int)
	ScrollUp(x int)
}
