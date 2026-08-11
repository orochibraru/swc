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

type Controller struct {
	optVolUp   machine.Pin
	optVolDown machine.Pin
	optMute    machine.Pin
	pinCLK     machine.Pin
	pinDT      machine.Pin
	pinSW      machine.Pin
	lastCLK    bool
}

func NewController() *Controller {
	c := &Controller{
		optVolUp:   PinVolUp,
		optVolDown: PinVolDown,
		optMute:    PinMute,
		pinCLK:     PinCLK,
		pinDT:      PinDT,
		pinSW:      PinSW,
	}
	c.init()
	return c
}

func (c *Controller) init() {
	// Configure outputs to drive the PC817 LEDs
	c.optVolUp.Configure(machine.PinConfig{Mode: machine.PinOutput})
	c.optVolDown.Configure(machine.PinConfig{Mode: machine.PinOutput})
	c.optMute.Configure(machine.PinConfig{Mode: machine.PinOutput})

	c.optVolUp.Low()
	c.optVolDown.Low()
	c.optMute.Low()

	// Configure inputs for the KY-040 rotary encoder
	c.pinCLK.Configure(machine.PinConfig{Mode: machine.PinInputPullup})
	c.pinDT.Configure(machine.PinConfig{Mode: machine.PinInputPullup})
	c.pinSW.Configure(machine.PinConfig{Mode: machine.PinInputPullup})

	c.lastCLK = c.pinCLK.Get()
}

// Pulses the target optocoupler HIGH for 60ms to let Pioneer head unit register the pulse
func (c *Controller) triggerOpto(pin machine.Pin) {
	pin.High()
	time.Sleep(60 * time.Millisecond)
	pin.Low()
}

func (c *Controller) Poll() {
	currentCLK := c.pinCLK.Get()

	// Falling edge on CLK signal means knob was turned
	if c.lastCLK && !currentCLK {
		if c.pinDT.Get() != currentCLK {
			// Clockwise rotation -> Volume Up
			c.triggerOpto(c.optVolUp)
		} else {
			// Counter-clockwise rotation -> Volume Down
			c.triggerOpto(c.optVolDown)
		}
	}
	c.lastCLK = currentCLK

	// Encoder button press -> Mute
	if !c.pinSW.Get() {
		c.triggerOpto(c.optMute)
		// Simple software debounce delay
		time.Sleep(200 * time.Millisecond)
	}
}

func main() {
	controller := NewController()

	// Main execution loop
	for {
		controller.Poll()
		time.Sleep(1 * time.Millisecond)
	}
}
