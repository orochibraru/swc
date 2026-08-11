//go:build tinygo

package main

import (
	"machine"
	"time"
)

// GPIO pin definitions for PC817 optocouplers
const (
	PinVolUp   = machine.GP16
	PinVolDown = machine.GP17
	PinMute    = machine.GP21
)

// GPIO pin definitions for KY-040 rotary encoder
const (
	PinCLK = machine.GP14
	PinDT  = machine.GP15
	PinSW  = machine.GP20
)

// NewController configures the real GPIO pins and returns a Controller
// wired up to them.
func NewController() *Controller {
	outputs := [...]machine.Pin{PinVolUp, PinVolDown, PinMute}
	for _, p := range outputs {
		p.Configure(machine.PinConfig{Mode: machine.PinOutput})
	}

	inputs := [...]machine.Pin{PinCLK, PinDT, PinSW}
	for _, p := range inputs {
		p.Configure(machine.PinConfig{Mode: machine.PinInputPullup})
	}

	return newController(PinVolUp, PinVolDown, PinMute, PinCLK, PinDT, PinSW, time.Sleep)
}

func main() {
	controller := NewController()

	// Main execution loop
	for {
		controller.Poll()
		time.Sleep(1 * time.Millisecond)
	}
}
