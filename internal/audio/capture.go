package audio

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"
	"time"
)

const (
	wavPath     = "/tmp/voxctrl_input.wav"
	sampleRate  = 16000 // Hz — Whisper expects 16kHz
	channels    = 1     // mono
	bitDepth    = "S16_LE"
	maxDuration = 10 * time.Second       // hard cap — safety net
	minDuration = 500 * time.Millisecond // discard accidental key taps
)

// Recorder captures microphone input via arecord (ALSA).
// It starts on a press signal and stops on a release signal or timeout.
type Recorder struct {
	device string // ALSA device e.g. "default" or "hw:1,0"
}

// NewRecorder creates a Recorder using the default ALSA capture device.
// Pass a specific device string (e.g. "hw:1,0") if needed, or "" for default.
func NewRecorder(device string) *Recorder {
	if device == "" {
		device = "default"
	}
	return &Recorder{device: device}
}

// Record starts arecord, waits for the release signal or timeout,
// kills the process, and returns the path to the WAV file.
//
// The stop channel is the same hotkeyRelease channel from main —
// when the user releases either Ctrl or Alt, this unblocks and recording stops.
func (r *Recorder) Record(stop <-chan bool) (string, error) {
	// Clean up any previous recording
	_ = os.Remove(wavPath)

	start := time.Now()

	// Build arecord command
	// -D device        ALSA device to use
	// -f S16_LE        16-bit signed little-endian samples
	// -r 16000         16kHz sample rate (Whisper requirement)
	// -c 1             mono channel
	// -t wav           output format
	// wavPath          output file
	cmd := exec.Command("arecord",
		"-D", r.device,
		"-f", bitDepth,
		"-r", fmt.Sprintf("%d", sampleRate),
		"-c", fmt.Sprintf("%d", channels),
		"-t", "wav",
		wavPath,
	)

	// Separate process group so we can kill cleanly without SIGKILL to parent
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Pipe stderr so arecord startup errors surface in our logs
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("arecord start: %w", err)
	}

	log.Printf("[audio] Recording started (device=%s, max=%s)", r.device, maxDuration)

	// Wait for key release OR timeout — whichever comes first
	select {
	case <-stop:
		log.Println("[audio] Key released — stopping recording")
	case <-time.After(maxDuration):
		log.Printf("[audio] Max duration (%s) reached — stopping recording", maxDuration)
	}

	elapsed := time.Since(start)

	// Stop arecord cleanly by sending SIGTERM to its process group
	if err := stopProcess(cmd); err != nil {
		log.Printf("[audio] Warning: could not stop arecord cleanly: %v", err)
	}

	// Discard recordings shorter than minDuration — likely accidental trigger
	if elapsed < minDuration {
		_ = os.Remove(wavPath)
		return "", fmt.Errorf("recording too short (%dms) — discarded", elapsed.Milliseconds())
	}

	// Verify the output file exists and has real content
	if err := validateWav(wavPath); err != nil {
		return "", fmt.Errorf("wav validation: %w", err)
	}

	log.Printf("[audio] Recording complete: %dms → %s", elapsed.Milliseconds(), wavPath)
	return wavPath, nil
}

// stopProcess sends SIGTERM to the arecord process group, allowing it to
// flush the WAV header properly before exiting. Falls back to SIGKILL.
func stopProcess(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}

	// Send SIGTERM to the entire process group (negative PID)
	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err != nil {
		// fallback: kill just the process
		return cmd.Process.Signal(syscall.SIGTERM)
	}

	// SIGTERM to process group — lets arecord write the WAV header cleanly
	if err := syscall.Kill(-pgid, syscall.SIGTERM); err != nil {
		// fallback
		return cmd.Process.Kill()
	}

	// Give it 500ms to exit cleanly, then force kill
	done := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(500 * time.Millisecond):
		log.Println("[audio] arecord did not exit cleanly, sending SIGKILL")
		return syscall.Kill(-pgid, syscall.SIGKILL)
	}
}

// validateWav checks that the output file exists and is large enough
// to contain actual audio data (WAV header = 44 bytes minimum).
func validateWav(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("wav file missing: %w", err)
	}
	if info.Size() < 44 {
		return fmt.Errorf("wav file too small (%d bytes) — no audio captured", info.Size())
	}
	return nil
}

// ListDevices prints all available ALSA capture devices to stdout.
// Useful for debugging when audio capture fails.
// Call via: audio.ListDevices()
func ListDevices() {
	out, err := exec.Command("arecord", "-l").Output()
	if err != nil {
		log.Printf("[audio] Could not list devices: %v", err)
		return
	}
	log.Printf("[audio] Available capture devices:\n%s", string(out))
}
