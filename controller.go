package main

import "time"

// pin is the subset of machine.Pin's behavior the controller depends on.
// machine.Pin satisfies this implicitly, and tests supply a fake in its
// place, so the polling logic can run on the host without tinygo or real
// hardware.
type pin interface {
	Get() bool
	High()
	Low()
}

const (
	// optoPulse is how long an output pin is held HIGH to drive a PC817
	// optocoupler LED long enough for the Pioneer head unit to register it.
	optoPulse = 60 * time.Millisecond

	// switchDebounce is an extra pause after a mute trigger to ride out
	// mechanical contact bounce on the encoder's push switch.
	switchDebounce = 200 * time.Millisecond
)

// Controller reads a KY-040 rotary encoder and drives three PC817
// optocouplers that emulate a Pioneer steering-wheel-control input.
type Controller struct {
	optVolUp   pin
	optVolDown pin
	optMute    pin

	pinCLK pin
	pinDT  pin
	pinSW  pin

	lastCLK bool
	lastSW  bool

	sleep func(time.Duration)
}

// newController wires up a Controller from pins and a sleep function.
// Kept unexported and dependency-injected so tests can pass in fakes;
// the tinygo build exposes NewController() to wire real hardware.
func newController(optVolUp, optVolDown, optMute, pinCLK, pinDT, pinSW pin, sleep func(time.Duration)) *Controller {
	c := &Controller{
		optVolUp:   optVolUp,
		optVolDown: optVolDown,
		optMute:    optMute,
		pinCLK:     pinCLK,
		pinDT:      pinDT,
		pinSW:      pinSW,
		sleep:      sleep,
	}

	c.optVolUp.Low()
	c.optVolDown.Low()
	c.optMute.Low()

	c.lastCLK = c.pinCLK.Get()
	c.lastSW = c.pinSW.Get()

	return c
}

// triggerOpto pulses the target optocoupler HIGH for optoPulse.
func (c *Controller) triggerOpto(p pin) {
	p.High()
	c.sleep(optoPulse)
	p.Low()
}

// Poll reads the encoder and switch once and drives the matching
// optocoupler on state changes. Call it repeatedly from the main loop.
func (c *Controller) Poll() {
	currentCLK := c.pinCLK.Get()

	// Falling edge on CLK means the knob was turned one detent.
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

	currentSW := c.pinSW.Get()

	// Falling edge on SW means the button was just pressed. Edge-triggering
	// (rather than reacting to the level every poll) makes one physical
	// press produce exactly one mute trigger, no matter how long it's held.
	if c.lastSW && !currentSW {
		c.triggerOpto(c.optMute)
		c.sleep(switchDebounce)
	}
	c.lastSW = currentSW
}
