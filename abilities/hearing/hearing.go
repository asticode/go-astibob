package astihearing

// Constants
const (
	Name = "Hearing"
)

// SampleReader represents a sample reader
type SampleReader interface {
	ReadSample() (int32, error)
}

// Starter represents an object capable of starting and stopping itself
type Starter interface {
	Start() error
	Stop() error
}

// Websocket event names
const (
	websocketEventNameSamples = "samples"
)
