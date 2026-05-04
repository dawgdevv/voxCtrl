package hotkey

import "log"

type Listener struct {
	press   chan<- bool
	release chan<- bool
}

func NewListener(press, release chan<- bool) *Listener {
	return &Listener{press: press, release: release}
}

// will implement this later, for now just a stub to show that the listener is working
func (l *Listener) Listen() {
	log.Println("[hotkey] Listening for hotkey events...")
}
