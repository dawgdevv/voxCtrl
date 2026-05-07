package tray

import (
	"fmt"
	"image/color"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"fyne.io/systray"
)

// Tray is a native system-tray manager that replaces notify-send.
// It shows colour-coded status icons and a built-in log viewer.
type Tray struct {
	mu            sync.Mutex
	logs          []string
	logFile       string
	logHandle     *os.File
	quitCh        chan struct{}

	// pre-rendered PNG icons
	greyIcon  []byte
	whiteIcon []byte
	greenIcon []byte
	redIcon   []byte

	statusItem *systray.MenuItem
	logItems   []*systray.MenuItem
}

// New creates the tray manager, generating icons and opening the log file.
func New() *Tray {
	dir := filepath.Join(os.Getenv("HOME"), ".local", "share", "voxctrl")
	_ = os.MkdirAll(dir, 0755)
	logPath := filepath.Join(dir, "voxctrl.log")

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		// Log to stderr if we can't open the file
		fmt.Fprintf(os.Stderr, "[tray] cannot open log file: %v\n", err)
	}

	return &Tray{
		logs:      make([]string, 0, 100),
		logFile:   logPath,
		logHandle: f,
		quitCh:    make(chan struct{}),
		greyIcon:  generateCircleIcon(color.RGBA{128, 128, 128, 255}, 64),
		whiteIcon: generateCircleIcon(color.RGBA{255, 255, 255, 255}, 64),
		greenIcon: generateCircleIcon(color.RGBA{76, 175, 80, 255}, 64),
		redIcon:   generateCircleIcon(color.RGBA{244, 67, 54, 255}, 64),
	}
}

// Run starts the system tray event loop. It blocks; call in a goroutine.
func (t *Tray) Run() {
	systray.Run(t.onReady, t.onExit)
}

// WaitForQuit returns a channel that closes when the tray Quit menu is clicked.
func (t *Tray) WaitForQuit() <-chan struct{} {
	return t.quitCh
}

// Logf records a timestamped message to the internal buffer and disk log.
func (t *Tray) Logf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	ts := time.Now().Format("15:04:05")
	line := fmt.Sprintf("[%s] %s", ts, msg)

	t.mu.Lock()
	t.logs = append(t.logs, line)
	if len(t.logs) > 100 {
		t.logs = t.logs[len(t.logs)-100:]
	}
	t.mu.Unlock()

	if t.logHandle != nil {
		_, _ = fmt.Fprintln(t.logHandle, line)
	}

	t.updateLogMenu()
}

// Idle sets the tray to grey — the default waiting state.
func (t *Tray) Idle() {
	if t.statusItem != nil {
		systray.SetIcon(t.greyIcon)
		systray.SetTooltip("VoxCtrl — Idle")
		t.statusItem.SetTitle("Status: Idle")
	}
}

// Active sets the tray to white — while recording/listening.
func (t *Tray) Active() {
	if t.statusItem != nil {
		systray.SetIcon(t.whiteIcon)
		systray.SetTooltip("VoxCtrl — Listening...")
		t.statusItem.SetTitle("Status: Listening...")
	}
	t.Logf("Listening...")
}

// Success sets the tray to green and reverts to idle after 3s.
func (t *Tray) Success(msg string) {
	if t.statusItem != nil {
		systray.SetIcon(t.greenIcon)
		systray.SetTooltip("VoxCtrl — Success: " + msg)
		t.statusItem.SetTitle("Status: " + msg)
	}
	t.Logf("SUCCESS: %s", msg)
	t.revertToIdle(3 * time.Second)
}

// Error sets the tray to red and reverts to idle after 5s.
func (t *Tray) Error(msg string) {
	if t.statusItem != nil {
		systray.SetIcon(t.redIcon)
		systray.SetTooltip("VoxCtrl — Error: " + msg)
		t.statusItem.SetTitle("Status: " + truncate(msg, 40))
	}
	t.Logf("ERROR: %s", msg)
	t.revertToIdle(5 * time.Second)
}

// Info just logs a message without changing the icon.
func (t *Tray) Info(msg string) {
	t.Logf("INFO: %s", msg)
}

// revertToIdle schedules a return to the grey idle icon.
func (t *Tray) revertToIdle(after time.Duration) {
	time.AfterFunc(after, func() {
		t.Idle()
	})
}

// onReady is called by systray when the tray icon is ready to be drawn.
func (t *Tray) onReady() {
	systray.SetIcon(t.greyIcon)
	systray.SetTooltip("VoxCtrl — Voice Command Daemon")
	systray.SetTitle("Vox")

	t.statusItem = systray.AddMenuItem("Status: Idle", "")
	t.statusItem.Disable()

	systray.AddSeparator()

	mLogs := systray.AddMenuItem("Recent Logs", "")
	mLogs.Disable()
	for i := 0; i < 10; i++ {
		mi := systray.AddMenuItem("", "")
		mi.Disable()
		mi.Hide()
		t.logItems = append(t.logItems, mi)
	}

	systray.AddSeparator()

	mOpen := systray.AddMenuItem("Open Full Log", "Open log file in default editor")
	go func() {
		for range mOpen.ClickedCh {
			_ = exec.Command("xdg-open", t.logFile).Start()
		}
	}()

	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Shutdown VoxCtrl")
	go func() {
		for range mQuit.ClickedCh {
			systray.Quit()
		}
	}()
}

// onExit is called by systray when the tray is shutting down.
func (t *Tray) onExit() {
	if t.logHandle != nil {
		_ = t.logHandle.Close()
	}
	close(t.quitCh)
}

// updateLogMenu refreshes the Recent Logs submenu with the latest entries.
func (t *Tray) updateLogMenu() {
	t.mu.Lock()
	defer t.mu.Unlock()

	count := len(t.logs)
	if count > len(t.logItems) {
		count = len(t.logItems)
	}

	for i := 0; i < len(t.logItems); i++ {
		if i < count {
			idx := len(t.logs) - 1 - i
			t.logItems[i].SetTitle(t.logs[idx])
			t.logItems[i].Show()
		} else {
			t.logItems[i].Hide()
		}
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
