package astispeech

// Handler represents a handler
type Handler struct{}

// NewHandler creates a new Handler
func NewHandler() *Handler {
	return &Handler{}
}

// Handle implements the astibob.HearingHandler interface
func (h Handler) Handle(e astibob.HearingEvent) (err error) {
	// TODO Try to parse using deepspeech
	// TODO If failure, store file and add in DB
	return
}
