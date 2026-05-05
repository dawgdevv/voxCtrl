package hotkey

import "testing"

func TestHandleEventAltPress(t *testing.T) {
	press := make(chan bool, 1)
	release := make(chan bool, 1)
	l := &Listener{press: press, release: release}

	// Press left Alt
	l.handleEvent(keyLeftAlt, keyPress)
	if !l.altHeld {
		t.Error("expected altHeld to be true")
	}
}

func TestHandleEventAltRelease(t *testing.T) {
	press := make(chan bool, 1)
	release := make(chan bool, 1)
	l := &Listener{press: press, release: release, altHeld: true, active: true}

	// Release left Alt while active
	l.handleEvent(keyLeftAlt, keyRelease)
	if l.altHeld {
		t.Error("expected altHeld to be false")
	}
	if l.active {
		t.Error("expected active to be false")
	}

	select {
	case <-release:
		// good
	default:
		t.Error("expected release signal")
	}
}

func TestHandleEventHotkeyPress(t *testing.T) {
	press := make(chan bool, 1)
	release := make(chan bool, 1)
	l := &Listener{press: press, release: release, altHeld: true}

	l.handleEvent(keyV, keyPress)
	if !l.active {
		t.Error("expected active to be true")
	}

	select {
	case <-press:
		// good
	default:
		t.Error("expected press signal")
	}
}

func TestHandleEventHotkeyRelease(t *testing.T) {
	press := make(chan bool, 1)
	release := make(chan bool, 1)
	l := &Listener{press: press, release: release, altHeld: true, active: true}

	l.handleEvent(keyV, keyRelease)
	if l.active {
		t.Error("expected active to be false")
	}

	select {
	case <-release:
		// good
	default:
		t.Error("expected release signal")
	}
}

func TestHandleEventIgnoresNonKeyEvents(t *testing.T) {
	press := make(chan bool, 1)
	release := make(chan bool, 1)
	l := &Listener{press: press, release: release}

	// Random key code, not Alt or V
	l.handleEvent(999, keyPress)
	if l.altHeld {
		t.Error("expected altHeld to remain false")
	}
	if l.active {
		t.Error("expected active to remain false")
	}
}
