package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/dawgdevv/voxctrl/internal/audio"
	"github.com/dawgdevv/voxctrl/internal/executor"
	"github.com/dawgdevv/voxctrl/internal/hotkey"
	"github.com/dawgdevv/voxctrl/internal/intent"
	"github.com/dawgdevv/voxctrl/internal/notify"
	"github.com/dawgdevv/voxctrl/internal/session"
	"github.com/dawgdevv/voxctrl/internal/stt"
)

func main() {
	log.Println("[VoxCtrl] Starting daemon...")

	// ── Config from environment ──────────────────────────────────────────────
	// VOXCTRL_DEVICE = evdev keyboard device path
	// e.g. VOXCTRL_DEVICE=/dev/input/event3
	// Leave unset for auto-detection (recommended)
	inputDevice := os.Getenv("VOXCTRL_DEVICE")

	// VOXCTRL_ALSA_DEVICE = ALSA capture device
	// e.g. VOXCTRL_ALSA_DEVICE=hw:1,0
	// Leave unset to use "default"
	alsaDevice := os.Getenv("VOXCTRL_ALSA_DEVICE")

	// ── Initialize subsystems ────────────────────────────────────────────────
	notifier := notify.New()

	store, err := session.NewStore()
	if err != nil {
		log.Fatalf("[VoxCtrl] Failed to init session store: %v", err)
	}
	defer store.Close()

	registry, err := intent.NewRegistry("config/commands.yaml")
	if err != nil {
		log.Fatalf("[VoxCtrl] Failed to load command registry: %v", err)
	}

	parser := intent.NewParser(registry)
	runner := executor.NewRunner(notifier)
	transcriber := stt.NewWhisper("/usr/local/share/whisper/ggml-base.en.bin")
	recorder := audio.NewRecorder(alsaDevice)

	// ── Channels — the pipeline backbone ─────────────────────────────────────
	//
	// hotkeyPress   — hotkey listener  → audio capture goroutine
	// hotkeyRelease — hotkey listener  → audio capture goroutine (stop signal)
	// audioReady    — audio capture    → STT goroutine
	// transcript    — STT              → intent parser goroutine
	// actionReady   — intent parser    → executor goroutine
	//
	// Buffer size 1 on each channel: if a stage is still processing the
	// previous command, new signals are dropped with a log warning rather
	// than blocking the hotkey listener.

	hotkeyPress := make(chan bool, 1)
	hotkeyRelease := make(chan bool, 1)
	audioReady := make(chan string, 1)
	transcript := make(chan string, 1)
	actionReady := make(chan executor.Action, 1)

	// ── Hotkey listener ───────────────────────────────────────────────────────
	hk, err := hotkey.NewListener(hotkeyPress, hotkeyRelease, inputDevice)
	if err != nil {
		log.Fatalf("[VoxCtrl] Hotkey init failed: %v\n"+
			"Hint: make sure you are in the 'input' group:\n"+
			"  sudo usermod -aG input $USER  (then log out and back in)\n"+
			"Or set VOXCTRL_DEVICE=/dev/input/eventX manually.", err)
	}
	go hk.Listen()

	// ── Stage 1: Audio capture ────────────────────────────────────────────────
	// Waits for hotkeyPress, records until hotkeyRelease or timeout,
	// sends WAV path downstream.
	go func() {
		for {
			<-hotkeyPress
			log.Println("[audio] Hotkey pressed — starting capture")
			notifier.Info("Listening...")

			wavPath, err := recorder.Record(hotkeyRelease)
			if err != nil {
				log.Printf("[audio] Capture error: %v", err)
				notifier.Error("Audio capture failed: " + err.Error())
				continue
			}

			select {
			case audioReady <- wavPath:
			default:
				log.Println("[audio] STT stage busy — dropping audio")
				notifier.Error("Still processing previous command")
			}
		}
	}()

	// ── Stage 2: Speech-to-text ───────────────────────────────────────────────
	go func() {
		for wavPath := range audioReady {
			log.Println("[stt] Transcribing...")

			text, err := transcriber.Transcribe(wavPath)
			if err != nil {
				log.Printf("[stt] Error: %v", err)
				notifier.Error("Transcription failed")
				continue
			}

			if text == "" {
				log.Println("[stt] Empty transcript — nothing spoken?")
				notifier.Info("Didn't catch that — try again")
				continue
			}

			log.Printf("[stt] Transcript: %q", text)

			select {
			case transcript <- text:
			default:
				log.Println("[stt] Intent stage busy — dropping transcript")
			}
		}
	}()

	// ── Stage 3: Intent resolution ────────────────────────────────────────────
	go func() {
		for text := range transcript {
			action, confidence, err := parser.Resolve(text)
			if err != nil || confidence < 0.75 {
				log.Printf("[intent] No match for %q (confidence=%.2f)", text, confidence)
				notifier.Error("Not recognised: \"" + text + "\"")
				continue
			}

			log.Printf("[intent] Matched %q → %q (confidence=%.2f)", text, action.Name(), confidence)

			select {
			case actionReady <- action:
			default:
				log.Println("[intent] Executor busy — dropping action")
			}
		}
	}()

	// ── Stage 4: Execution + session logging ──────────────────────────────────
	go func() {
		for action := range actionReady {
			entry := store.NewEntry(action.Name())
			err := runner.Run(action)
			entry.Finish(err)

			if logErr := store.Save(entry); logErr != nil {
				log.Printf("[session] Failed to log entry: %v", logErr)
			}
		}
	}()

	// ── Ready ─────────────────────────────────────────────────────────────────
	notifier.Info("VoxCtrl ready — hold Alt+V to speak")
	log.Println("[VoxCtrl] Daemon running. Ctrl+C to stop.")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	log.Println("[VoxCtrl] Shutting down.")
}
