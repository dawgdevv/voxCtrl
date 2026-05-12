package hotkey

import "testing"

func TestHandleEventCtrlPress(t *testing.T) {
	press := make(chan bool, 1)
	release := make(chan bool, 1)
	l := &Listener{press: press, release: release}

	l.handleEvent(keyLeftCtrl, keyPress)
	if !l.ctrlHeld {
		t.Error("expected ctrlHeld to be true")
	}
	if l.active {
		t.Error("expected active to remain false (Alt not yet held)")
	}
}

func TestHandleEventAltPress(t *testing.T) {
	press := make(chan bool, 1)
	release := make(chan bool, 1)
	l := &Listener{press: press, release: release}

	l.handleEvent(keyLeftAlt, keyPress)
	if !l.altHeld {
		t.Error("expected altHeld to be true")
	}
	if l.active {
		t.Error("expected active to remain false (Ctrl not yet held)")
	}
}

func TestHandleEventCtrlAltPress(t *testing.T) {
	press := make(chan bool, 1)
	release := make(chan bool, 1)
	l := &Listener{press: press, release: release, ctrlHeld: true}

	// Alt pressed while Ctrl already held -> activate
	l.handleEvent(keyLeftAlt, keyPress)
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

func TestHandleEventAltCtrlPress(t *testing.T) {
	press := make(chan bool, 1)
	release := make(chan bool, 1)
	l := &Listener{press: press, release: release, altHeld: true}

	// Ctrl pressed while Alt already held -> activate
	l.handleEvent(keyLeftCtrl, keyPress)
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

func TestHandleEventCtrlReleaseWhileActive(t *testing.T) {
	press := make(chan bool, 1)
	release := make(chan bool, 1)
	l := &Listener{press: press, release: release, ctrlHeld: true, altHeld: true, active: true}

	l.handleEvent(keyLeftCtrl, keyRelease)
	if l.ctrlHeld {
		t.Error("expected ctrlHeld to be false")
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

func TestHandleEventAltReleaseWhileActive(t *testing.T) {
	press := make(chan bool, 1)
	release := make(chan bool, 1)
	l := &Listener{press: press, release: release, ctrlHeld: true, altHeld: true, active: true}

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

func TestHandleEventIgnoresNonKeyEvents(t *testing.T) {
	press := make(chan bool, 1)
	release := make(chan bool, 1)
	l := &Listener{press: press, release: release}

	// Random key code, not Ctrl or Alt
	l.handleEvent(999, keyPress)
	if l.ctrlHeld {
		t.Error("expected ctrlHeld to remain false")
	}
	if l.altHeld {
		t.Error("expected altHeld to remain false")
	}
	if l.active {
		t.Error("expected active to remain false")
	}
}
