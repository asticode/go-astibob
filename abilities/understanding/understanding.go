package astiunderstanding

// Constants
const (
	name = "Understanding"
)

// SilenceDetector represents an object capable of detecting valid samples between silences
type SilenceDetector interface {
	Add(samples []int32, sampleRate int, silenceMaxAudioLevel float64) (validSamples [][]int32)
	Reset()
}

// SpeechParser represents an object capable of parsing speech and returning the corresponding text
type SpeechParser interface {
	SpeechToText(buffer []int32, bufferSize, sampleRate, significantBits int) string
}

// Websocket event names
const (
	websocketEventNameAnalysis      = "analysis"
	websocketEventNameSamples       = "samples"
	websocketEventNameSamplesStored = "samples.stored"
)
