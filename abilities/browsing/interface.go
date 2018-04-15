package astibrowsing

// Interface is the interface of the ability
type Interface struct{}

// NewInterface creates a new interface
func NewInterface() (i *Interface) {
	return &Interface{}
}

// Name implements the astibob.Interface interface
func (i *Interface) Name() string {
	return name
}
