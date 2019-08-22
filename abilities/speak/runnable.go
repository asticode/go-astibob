package speak

import "github.com/asticode/go-astibob"

func NewRunnable(name string) astibob.Runnable {
	return astibob.NewRunnable(astibob.RunnableOptions{
		Metadata: astibob.Metadata{
			Description: "Says words to your audio output using speech synthesis",
			Name:        name,
		},
	})
}
