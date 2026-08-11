package main

import (
	"testing"
	"time"
)

// fakePin is an in-memory stand-in for machine.Pin. Tests drive .level to
// simulate the physical signal and inspect the High/Low call counts to
// check what the controller drove out.
type fakePin struct {
	level     bool
	highCalls int
	lowCalls  int
}

func (p *fakePin) Get() bool { return p.level }
func (p *fakePin) High()     { p.highCalls++; p.level = true }
func (p *fakePin) Low()      { p.lowCalls++; p.level = false }

// testController bundles a Controller with its fake pins so tests can
// both drive inputs and assert on outputs.
type testController struct {
	c *Controller

	volUp, volDown, mute *fakePin
	clk, dt, sw          *fakePin

	sleeps []time.Duration
}

// newTestController builds a controller with idle-state fakes: encoder
// pull-ups read HIGH at rest, CLK/DT HIGH (detent position).
func newTestController() *testController {
	tc := &testController{
		volUp:   &fakePin{},
		volDown: &fakePin{},
		mute:    &fakePin{},
		clk:     &fakePin{level: true},
		dt:      &fakePin{level: true},
		sw:      &fakePin{level: true},
	}
	tc.c = newController(tc.volUp, tc.volDown, tc.mute, tc.clk, tc.dt, tc.sw, func(d time.Duration) {
		tc.sleeps = append(tc.sleeps, d)
	})
	return tc
}

func TestNewController_DrivesOutputsLowOnInit(t *testing.T) {
	tc := newTestController()

	if tc.volUp.level || tc.volDown.level || tc.mute.level {
		t.Fatalf("expected all optocoupler outputs LOW after init, got volUp=%v volDown=%v mute=%v",
			tc.volUp.level, tc.volDown.level, tc.mute.level)
	}
}

func TestPoll_NoChange_NoTrigger(t *testing.T) {
	tc := newTestController()

	tc.c.Poll()
	tc.c.Poll()
	tc.c.Poll()

	if tc.volUp.highCalls != 0 || tc.volDown.highCalls != 0 || tc.mute.highCalls != 0 {
		t.Fatalf("expected no triggers with steady inputs, got volUp=%d volDown=%d mute=%d",
			tc.volUp.highCalls, tc.volDown.highCalls, tc.mute.highCalls)
	}
}

func TestPoll_ClockwiseTurn_TriggersVolumeUp(t *testing.T) {
	tc := newTestController()

	// Clockwise: DT stays HIGH while CLK falls.
	tc.clk.level = false
	tc.c.Poll()

	if tc.volUp.highCalls != 1 {
		t.Fatalf("expected 1 volume-up trigger, got %d", tc.volUp.highCalls)
	}
	if tc.volDown.highCalls != 0 {
		t.Fatalf("expected 0 volume-down triggers, got %d", tc.volDown.highCalls)
	}
	if tc.volUp.level {
		t.Fatal("expected volume-up pin to be LOW after the pulse completes")
	}
}

func TestPoll_CounterClockwiseTurn_TriggersVolumeDown(t *testing.T) {
	tc := newTestController()

	// Counter-clockwise: DT falls together with CLK.
	tc.clk.level = false
	tc.dt.level = false
	tc.c.Poll()

	if tc.volDown.highCalls != 1 {
		t.Fatalf("expected 1 volume-down trigger, got %d", tc.volDown.highCalls)
	}
	if tc.volUp.highCalls != 0 {
		t.Fatalf("expected 0 volume-up triggers, got %d", tc.volUp.highCalls)
	}
}

func TestPoll_RisingEdgeOnCLK_DoesNotTrigger(t *testing.T) {
	tc := newTestController()

	// Take CLK low first (falling edge, consumed normally)...
	tc.clk.level = false
	tc.c.Poll()
	tc.volUp.highCalls, tc.volDown.highCalls = 0, 0 // ignore that first edge

	// ...then let it rise back to idle. A rising edge must not trigger.
	tc.clk.level = true
	tc.c.Poll()

	if tc.volUp.highCalls != 0 || tc.volDown.highCalls != 0 {
		t.Fatalf("rising edge on CLK should not trigger anything, got volUp=%d volDown=%d",
			tc.volUp.highCalls, tc.volDown.highCalls)
	}
}

func TestPoll_HeldLowCLK_TriggersOnlyOnce(t *testing.T) {
	tc := newTestController()

	tc.clk.level = false
	tc.c.Poll() // falling edge -> 1 trigger
	tc.c.Poll() // still low -> no new edge
	tc.c.Poll()

	if tc.volUp.highCalls != 1 {
		t.Fatalf("expected exactly 1 trigger while CLK stays low, got %d", tc.volUp.highCalls)
	}
}

func TestPoll_ButtonPress_TriggersMuteOnce(t *testing.T) {
	tc := newTestController()

	tc.sw.level = false
	tc.c.Poll()

	if tc.mute.highCalls != 1 {
		t.Fatalf("expected 1 mute trigger on press, got %d", tc.mute.highCalls)
	}
}

func TestPoll_ButtonHeldDown_DoesNotRetrigger(t *testing.T) {
	tc := newTestController()

	tc.sw.level = false
	// Poll repeatedly while the button stays physically held down.
	for i := 0; i < 10; i++ {
		tc.c.Poll()
	}

	if tc.mute.highCalls != 1 {
		t.Fatalf("holding the button down should trigger mute exactly once, got %d triggers", tc.mute.highCalls)
	}
}

func TestPoll_ButtonPressReleasePress_TriggersMuteTwice(t *testing.T) {
	tc := newTestController()

	tc.sw.level = false
	tc.c.Poll() // press #1

	tc.sw.level = true
	tc.c.Poll() // release

	tc.sw.level = false
	tc.c.Poll() // press #2

	if tc.mute.highCalls != 2 {
		t.Fatalf("expected 2 mute triggers for press-release-press, got %d", tc.mute.highCalls)
	}
}

func TestPoll_ButtonPress_SleepsForPulseThenDebounce(t *testing.T) {
	tc := newTestController()

	tc.sw.level = false
	tc.c.Poll()

	if len(tc.sleeps) != 2 {
		t.Fatalf("expected 2 sleep calls (pulse + debounce), got %d: %v", len(tc.sleeps), tc.sleeps)
	}
	if tc.sleeps[0] != optoPulse {
		t.Errorf("expected first sleep to be the %v opto pulse, got %v", optoPulse, tc.sleeps[0])
	}
	if tc.sleeps[1] != switchDebounce {
		t.Errorf("expected second sleep to be the %v switch debounce, got %v", switchDebounce, tc.sleeps[1])
	}
}

func TestPoll_KnobAndButton_AreIndependent(t *testing.T) {
	tc := newTestController()

	tc.clk.level = false
	tc.sw.level = false
	tc.c.Poll()

	if tc.volUp.highCalls != 1 {
		t.Errorf("expected volume-up trigger, got %d", tc.volUp.highCalls)
	}
	if tc.mute.highCalls != 1 {
		t.Errorf("expected mute trigger, got %d", tc.mute.highCalls)
	}
}
