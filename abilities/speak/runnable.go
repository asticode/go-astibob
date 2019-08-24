package speak

import (
	"sync"

	"github.com/asticode/go-astibob"
)

func NewRunnable(name string) astibob.Runnable {
	r := newRunnable()
	return astibob.NewRunnable(astibob.RunnableOptions{
		Metadata: astibob.Metadata{
			Description: "Says words to your audio output using speech synthesis",
			Name:        name,
		},
		OnMessage: r.onMessage,
	})
}

type runnable struct {
	m *sync.Mutex
}

func newRunnable() *runnable {
	return &runnable{m: &sync.Mutex{}}
}

func (r *runnable) onMessage(m *astibob.Message) (err error) {
	return
}
