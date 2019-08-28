package astibob

type Listenable interface {
	MessageNames() []string
	OnMessage(m *Message) error
}
