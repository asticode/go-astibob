package astiunderstanding

// Constants
const (
	name = "Understanding"
)

// SpeechParser represents an object capable of parsing speech and returning the corresponding text
type SpeechParser interface {
	SpeechToText(buffer []int32, bufferSize, sampleRate, significantBits int) string
}

// Websocket event names
const (
	websocketEventNameAnalysis = "analysis"
	websocketEventNameSamples  = "samples"
)
