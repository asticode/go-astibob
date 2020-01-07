package speech_to_text

import (
	"fmt"

	"github.com/asticode/go-astibob"
)

type ListenableOptions struct {
	OnText func(from astibob.Identifier, text string) error
}

type Listenable struct {
	o ListenableOptions
}

func NewListenable(o ListenableOptions) *Listenable {
	return newListenable(o)
}

func newListenable(o ListenableOptions) *Listenable {
	return &Listenable{o: o}
}

func (l *Listenable) MessageNames() (ns []string) {
	if l.o.OnText != nil {
		ns = append(ns, textMessage)
	}
	return
}

func (l *Listenable) OnMessage(m *astibob.Message) (err error) {
	switch m.Name {
	case textMessage:
		if err = l.onText(m); err != nil {
			err = fmt.Errorf("speech_to_text: on text failed: %w", err)
			return
		}
	}
	return
}

func (l *Listenable) onText(m *astibob.Message) (err error) {
	// Parse payload
	var t Text
	if t, err = parseTextPayload(m); err != nil {
		err = fmt.Errorf("speech_to_text: parsing text payload failed: %w", err)
		return
	}

	// Custom
	if l.o.OnText != nil {
		if err = l.o.OnText(t.From, t.Text); err != nil {
			err = fmt.Errorf("speech_to_text: custom on text failed: %w", err)
			return
		}
	}
	return
}
