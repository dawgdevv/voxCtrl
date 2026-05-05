package hotkey

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// inputEvent mirrors the Linux kernel input_event struct (24 bytes on 64-bit).
// Defined in /usr/include/linux/input.h
type inputEvent struct {
	Sec   uint64 // timeval seconds
	Usec  uint64 // timeval microseconds
	Type  uint16 // EV_KEY = 1
	Code  uint16 // key code e.g. KEY_V = 47
	Value int32  // 0=release, 1=press, 2=repeat
}

// Key codes from linux/input-event-codes.h
const (
	evKey      = 1 // event type: key
	keyPress   = 1 // value: key pressed
	keyRelease = 0 // value: key released

	keyLeftAlt  = 56
	keyRightAlt = 100
	keyV        = 47 // KEY_V

	// Current hotkey: Alt + V
	// To change: update keyTrigger to any key code you want.
	// Common alternatives:
	//   keyF9  = 67   → Alt+F9
	//   keyF10 = 68   → Alt+F10
	//   keySpace = 57 → Alt+Space
	keyTrigger = keyV
)

// Listener watches /dev/input for the global hotkey and signals press/release.
// Uses evdev — works on X11, Wayland, and any Linux environment.
type Listener struct {
	press      chan<- bool
	release    chan<- bool
	devicePath string // path to keyboard device e.g. /dev/input/event3

	// internal state
	altHeld bool
	active  bool // true while hotkey is being held
}

// NewListener creates a Listener. If devicePath is empty, it auto-detects
// the first keyboard device under /dev/input/by-path or /dev/input/by-id.
func NewListener(press, release chan<- bool, devicePath string) (*Listener, error) {
	if devicePath == "" {
		detected, err := detectKeyboard()
		if err != nil {
			return nil, fmt.Errorf("hotkey: keyboard detection failed: %w", err)
		}
		devicePath = detected
	}
	log.Printf("[hotkey] Using input device: %s", devicePath)
	return &Listener{
		press:      press,
		release:    release,
		devicePath: devicePath,
	}, nil
}

// Listen opens the input device and blocks, reading events forever.
// Call in a goroutine: go listener.Listen()
func (l *Listener) Listen() {
	for {
		if err := l.readLoop(); err != nil {
			log.Printf("[hotkey] Read loop error: %v — retrying in 2s", err)
			time.Sleep(2 * time.Second)
		}
	}
}

// readLoop opens the device file and processes events until an error occurs.
func (l *Listener) readLoop() error {
	f, err := os.Open(l.devicePath)
	if err != nil {
		return fmt.Errorf("open device %s: %w", l.devicePath, err)
	}
	defer f.Close()

	log.Printf("[hotkey] Listening on %s (Alt+V to activate)", l.devicePath)

	var ev inputEvent
	for {
		// Each input_event is exactly 24 bytes on 64-bit Linux.
		if err := binary.Read(f, binary.LittleEndian, &ev); err != nil {
			return fmt.Errorf("read event: %w", err)
		}

		// We only care about key events (type == EV_KEY)
		if ev.Type != evKey {
			continue
		}

		l.handleEvent(ev.Code, ev.Value)
	}
}

// handleEvent updates modifier state and fires press/release signals.
func (l *Listener) handleEvent(code uint16, value int32) {
	// Track Alt modifier (left or right Alt)
	if code == keyLeftAlt || code == keyRightAlt {
		l.altHeld = (value == keyPress || value == 2) // 2 = auto-repeat

		// If Alt is released while hotkey is active, treat as hotkey release.
		// This handles the case where the user releases Alt before the trigger key.
		if value == keyRelease && l.active {
			l.active = false
			log.Println("[hotkey] Alt released — stopping voice input")
			select {
			case l.release <- true:
			default:
			}
		}
	}

	// Hotkey trigger key: V (or whatever keyTrigger is set to)
	if code == keyTrigger {
		if value == keyPress && l.altHeld && !l.active {
			// Alt+V pressed — start recording
			l.active = true
			log.Println("[hotkey] Alt+V pressed — activating voice input")
			select {
			case l.press <- true:
			default:
				log.Println("[hotkey] Pipeline busy, ignoring press")
				l.active = false
			}
		} else if value == keyRelease && l.active {
			// V released — stop recording
			l.active = false
			log.Println("[hotkey] V released — stopping voice input")
			select {
			case l.release <- true:
			default:
			}
		}
	}
}

// detectKeyboard finds the first keyboard input device.
// It prefers /dev/input/by-path entries with "kbd" in the name,
// falling back to scanning /dev/input/event* for devices that emit EV_KEY.
func detectKeyboard() (string, error) {
	// Strategy 1: look in /dev/input/by-path for a keyboard symlink
	byPath := "/dev/input/by-path"
	if entries, err := os.ReadDir(byPath); err == nil {
		for _, e := range entries {
			name := e.Name()
			if strings.Contains(name, "kbd") || strings.Contains(name, "keyboard") {
				resolved, err := filepath.EvalSymlinks(filepath.Join(byPath, name))
				if err == nil {
					return resolved, nil
				}
			}
		}
	}

	// Strategy 2: look in /dev/input/by-id
	byID := "/dev/input/by-id"
	if entries, err := os.ReadDir(byID); err == nil {
		for _, e := range entries {
			name := e.Name()
			if strings.Contains(name, "kbd") || strings.Contains(name, "keyboard") {
				resolved, err := filepath.EvalSymlinks(filepath.Join(byID, name))
				if err == nil {
					return resolved, nil
				}
			}
		}
	}

	// Strategy 3: scan /dev/input/event* and check capabilities via /proc/bus/input/devices
	device, err := scanInputDevices()
	if err == nil {
		return device, nil
	}

	return "", fmt.Errorf("no keyboard device found — set VOXCTRL_DEVICE env var manually")
}

// scanInputDevices reads /proc/bus/input/devices to find a keyboard.
// A keyboard reports EV_KEY (0x1) and has many key bits set.
func scanInputDevices() (string, error) {
	data, err := os.ReadFile("/proc/bus/input/devices")
	if err != nil {
		return "", err
	}

	var currentHandlers []string
	var isKeyboard bool

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "H: Handlers=") {
			// e.g. "H: Handlers=sysrq kbd event3 leds"
			handlers := strings.TrimPrefix(line, "H: Handlers=")
			currentHandlers = strings.Fields(handlers)
			isKeyboard = false
			for _, h := range currentHandlers {
				if h == "kbd" {
					isKeyboard = true
				}
			}
		}

		// Blank line = end of device block
		if line == "" && isKeyboard && len(currentHandlers) > 0 {
			for _, h := range currentHandlers {
				if strings.HasPrefix(h, "event") {
					return "/dev/input/" + h, nil
				}
			}
			isKeyboard = false
			currentHandlers = nil
		}
	}

	return "", fmt.Errorf("no keyboard found in /proc/bus/input/devices")
}
