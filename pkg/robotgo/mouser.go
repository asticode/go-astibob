package astirobotgo

import "github.com/go-vgo/robotgo"

// Mouser represents an object capable of interacting with a mouse
type Mouser struct{}

// NewMouser creates a new mouser
func NewMouser() *Mouser {
	return &Mouser{}
}

// ClickLeft clicks the left button of the mouse
func (m Mouser) ClickLeft(double bool) {
	robotgo.MouseClick("left", double)
}

// ClickMiddle clicks the middle button of the mouse
func (m Mouser) ClickMiddle(double bool) {
	robotgo.MouseClick("middle", double)
}

// ClickRight clicks the right button of the mouse
func (m Mouser) ClickRight(double bool) {
	robotgo.MouseClick("right", double)
}

// Move moves the mouse
func (m Mouser) Move(x, y int) {
	robotgo.MoveMouseSmooth(x, y)
}

// ScrollDown scrolls down the mouse
func (m Mouser) ScrollDown(x int) {
	robotgo.ScrollMouse(x, "down")
}

// ScrollUp scrolls up the mouse
func (m Mouser) ScrollUp(x int) {
	robotgo.ScrollMouse(x, "up")
}
