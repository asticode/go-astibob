package astihearing

// Name
const Name = "hearing"

// SampleReader represents a sample reader
type SampleReader interface {
	ReadSample() (int32, error)
}

// Starter represents an object capable of starting and stopping itself
type Starter interface {
	Start() error
	Stop() error
}
