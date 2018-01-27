package astirobotgo

import (
	"math/rand"
	"time"

	"github.com/go-vgo/robotgo"
)

// Keyboarder represents an object capable of interacting with a keyboard
type Keyboarder struct{}

// NewKeyboarder creates a new keyboarder
func NewKeyboarder() *Keyboarder {
	return &Keyboarder{}
}

// Press press keys simultaneously
func (k Keyboarder) Press(keys ...string) {
	if len(keys) == 0 {
		return
	}
	f := keys[0]
	var r []string
	if len(keys) > 1 {
		r = keys[1:]
	}
	robotgo.KeyTap(f, r)
}

// Type types a string with a delay
func (k Keyboarder) Type(s string) {
	for i := 0; i < len([]rune(s)); i++ {
		ustr := uint32(robotgo.CharCodeAt(s, i))
		robotgo.UnicodeType(ustr)
		time.Sleep(time.Duration(rand.Intn(50)+50) * time.Millisecond)
	}
}
