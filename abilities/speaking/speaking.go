package astispeaking

// Constants
const (
	name = "Speaking"
)

// Websocket event names
const (
	websocketEventNameSay = "say"
)

// Speaker represents an object capable of saying things to an audio output
type Speaker interface {
	Say(s string) error
}
